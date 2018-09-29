package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

var fReadTimeout = 4 * time.Second
var fWriteTimeout = 2 * time.Second

type fileServer struct {
	ln     net.Listener
	record map[uint64]struct{} // use to record nonce
	term   chan struct{}
	log    log15.Logger
	wg     sync.WaitGroup
	chain  Chain
}

func newFileServer(port uint16, chain Chain) (*fileServer, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.FormatUint(uint64(port), 10))

	if err != nil {
		return nil, err
	}

	return &fileServer{
		ln:     ln,
		record: make(map[uint64]struct{}),
		term:   make(chan struct{}),
		log:    log15.New("module", "net/fileServer"),
		chain:  chain,
	}, nil
}

func (s *fileServer) start() {
	s.wg.Add(1)
	go s.readLoop()
}

func (s *fileServer) stop() {
	select {
	case <-s.term:
	default:
		close(s.term)
		s.wg.Wait()
	}
}

func (s *fileServer) readLoop() {
	defer s.ln.Close()
	defer s.wg.Done()

	for {
		select {
		case <-s.term:
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}

			s.wg.Add(1)
			go s.handleConn(conn)
		}
	}
}

func (f *fileServer) handleConn(conn net.Conn) {
	defer conn.Close()
	defer f.wg.Done()

	for {
		select {
		case <-f.term:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(fReadTimeout))
			msg, err := p2p.ReadMsg(conn, true)
			if err != nil {
				f.log.Error(fmt.Sprintf("read message error: %v", err))
				return
			}

			code := cmd(msg.Cmd)
			if code == GetFilesCode {
				req := new(message.GetFiles)
				err = req.Deserialize(msg.Payload)

				if err != nil {
					f.log.Error(fmt.Sprintf("parse message %s error: %v", GetFilesCode, err))
					return
				}

				// send files
				for _, filename := range req.Names {
					conn.SetWriteDeadline(time.Now().Add(fWriteTimeout))
					n, err := io.Copy(conn, f.chain.Compressor().FileReader(filename))

					if err != nil {
						f.log.Error(fmt.Sprintf("send file<%s> to %s error: %v", req.Names, conn.RemoteAddr(), err))
						return
					} else {
						f.log.Info(fmt.Sprintf("send file<%s> %d bytes to %s done", req.Names, n, conn.RemoteAddr()))
					}
				}
			} else if code == ExceptionCode {
				exp, err := message.DeserializeException(msg.Payload)
				if err != nil {
					f.log.Error(fmt.Sprintf("deserialize exception message error: %v", err))
					return
				}
				f.log.Info(exp.String())
				return
			} else {
				f.log.Error(fmt.Sprintf("unexpected message code: %d", msg.Cmd))
			}
		}
	}
}

// @section fileClient

type connContext struct {
	net.Conn
	req   *fileReq
	addr  string
	idle  bool
	idleT time.Time
}

type delCtxEvent struct {
	ctx *connContext
	err error
}

type fileReq struct {
	files []*message.File
	nonce uint64
	peer  *Peer
	rec   receiveBlocks
	done  func(err error)
}

func (r *fileReq) addr() string {
	return r.peer.FileAddress().String()
}

type fileClient struct {
	conns    map[string]*connContext
	_request chan *fileReq
	idle     chan *connContext
	delConn  chan *delCtxEvent
	chain    Chain
	term     chan struct{}
	log      log15.Logger
	wg       sync.WaitGroup
}

func newFileClient(chain Chain) *fileClient {
	return &fileClient{
		conns:    make(map[string]*connContext),
		_request: make(chan *fileReq, 4),
		idle:     make(chan *connContext, 1),
		delConn:  make(chan *delCtxEvent, 1),
		chain:    chain,
		term:     make(chan struct{}),
		log:      log15.New("module", "net/fileClient"),
	}
}

func (fc *fileClient) start() {
	fc.wg.Add(1)
	go fc.loop()
}

func (fc *fileClient) stop() {
	select {
	case <-fc.term:
	default:
		close(fc.term)
		fc.wg.Wait()
	}
}

func (fc *fileClient) request(r *fileReq) {
	fc._request <- r
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	wait := make([]*fileReq, 0, 10)

	idleTimeout := 5 * time.Second
	ticker := time.NewTicker(idleTimeout)
	defer ticker.Stop()

	exp := message.FileTransDone
	data, _ := exp.Serialize()
	noNeed := &p2p.Msg{
		CmdSetID: CmdSet,
		Cmd:      uint64(ExceptionCode),
		Size:     uint64(len(data)),
		Payload:  data,
	}

	delCtx := func(ctx *connContext, hasErr bool) {
		if !hasErr {
			ctx.SetWriteDeadline(time.Now().Add(fWriteTimeout))
			p2p.WriteMsg(ctx, true, noNeed)
		}

		delete(fc.conns, ctx.addr)
		ctx.Close()
	}

loop:
	for {
		select {
		case <-fc.term:
			break loop

		case req := <-fc._request:
			addr := req.addr()
			var ctx *connContext
			var ok bool
			if ctx, ok = fc.conns[addr]; !ok {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					req.done(err)
					break
				}
				ctx = &connContext{
					Conn: conn,
					addr: addr,
					idle: true,
				}
				fc.conns[addr] = ctx
			}

			if ctx.idle {
				ctx.req = req
				fc.wg.Add(1)
				go fc.exe(ctx)
			} else {
				wait = append(wait, req)
			}

		case ctx := <-fc.idle:
			ctx.idle = true
			ctx.idleT = time.Now()
			for i, req := range wait {
				if req.addr() == ctx.addr {
					ctx.idle = false
					ctx.req = req

					copy(wait[i:], wait[i+1:])
					wait = wait[:len(wait)-1]
				}
			}

		case e := <-fc.delConn:
			delCtx(e.ctx, true)
			fc.log.Error(fmt.Sprintf("delete connection %s: %v", e.ctx.addr, e.err))

		case t := <-ticker.C:
			// remote the idle connection
			for _, ctx := range fc.conns {
				if ctx.idle && t.Sub(ctx.idleT) > idleTimeout {
					delCtx(ctx, false)
				}
			}
		}
	}

	for _, ctx := range fc.conns {
		delCtx(ctx, false)
	}

	for i := 0; i < len(fc.idle); i++ {
		ctx := <-fc.idle
		delCtx(ctx, false)
	}

	for i := 0; i < len(fc.delConn); i++ {
		e := <-fc.delConn
		delCtx(e.ctx, true)
	}
}

func (fc *fileClient) exe(ctx *connContext) {
	defer fc.wg.Done()

	req := ctx.req
	filenames := make([]string, len(req.files))
	for i, file := range req.files {
		filenames[i] = file.Name
	}
	msg := &message.GetFiles{
		Names: filenames,
		Nonce: req.nonce,
	}

	data, err := msg.Serialize()
	if err != nil {
		req.done(err)
		fc.idle <- ctx
		return
	}

	ctx.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	err = p2p.WriteMsg(ctx, true, &p2p.Msg{
		CmdSetID: CmdSet,
		Cmd:      uint64(GetFilesCode),
		Size:     uint64(len(data)),
		Payload:  data,
	})
	if err != nil {
		req.done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
		return
	}

	sblocks, mblocks, err := fc.readBlocks(ctx)
	if err != nil {
		req.done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
	} else {
		fc.idle <- ctx
		req.rec(sblocks, mblocks)
		req.done(nil)
	}
}

var errResTimeout = errors.New("wait for file response timeout")
var errFlieClientStopped = errors.New("fileClient stopped")

func (fc *fileClient) readBlocks(ctx *connContext) (sblocks []*ledger.SnapshotBlock, mblocks map[types.Address][]*ledger.AccountBlock, err error) {
	select {
	case <-fc.term:
		err = errFlieClientStopped
		return
	default:
		sblocks = make([]*ledger.SnapshotBlock, 0, ctx.req.file.End-ctx.req.file.Start+1)
		mblocks = make(map[types.Address][]*ledger.AccountBlock)

		fc.chain.Compressor().BlockParser(ctx, func(block ledger.Block, err error) {
			if err != nil {
				return
			}

			switch block.(type) {
			case *ledger.SnapshotBlock:
				sblocks = append(sblocks, block.(*ledger.SnapshotBlock))
			case *ledger.AccountBlock:
				ablock := block.(*ledger.AccountBlock)
				addr := ablock.AccountAddress
				mblocks[addr] = append(mblocks[addr], ablock)
			default:
				// nothing
			}
		})

		return
	}
}
