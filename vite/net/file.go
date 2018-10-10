package net

import (
	"encoding/hex"
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

var fReadTimeout = 20 * time.Second
var fWriteTimeout = 20 * time.Second

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
			//conn.SetReadDeadline(time.Now().Add(fReadTimeout))
			msg, err := p2p.ReadMsg(conn, true)
			if err != nil {
				f.log.Error(fmt.Sprintf("read message from %s error: %v", conn.RemoteAddr(), err))
				return
			}

			code := cmd(msg.Cmd)
			if code == GetFilesCode {
				req := new(message.GetFiles)
				err = req.Deserialize(msg.Payload)

				if err != nil {
					f.log.Error(fmt.Sprintf("parse message %s from %s error: %v", code, conn.RemoteAddr(), err))
					return
				}

				f.log.Info(fmt.Sprintf("receive message %s from %s", code, conn.RemoteAddr()))

				// send files
				for _, filename := range req.Names {
					//conn.SetWriteDeadline(time.Now().Add(fWriteTimeout))
					var n int64
					n, err = io.Copy(conn, f.chain.Compressor().FileReader(filename))

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
				f.log.Warn(fmt.Sprintf("got exception %v", exp))
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
	req   *fileRequest
	addr  string
	idle  bool
	idleT time.Time
}

type delCtxEvent struct {
	ctx *connContext
	err error
}

type fileClient struct {
	conns    map[string]*connContext
	_request chan *fileRequest
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
		_request: make(chan *fileRequest, 4),
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

func (fc *fileClient) request(r *fileRequest) {
	fc._request <- r
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	wait := make([]*fileRequest, 0, 10)

	idleTimeout := 30 * time.Second
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
			addr := req.Addr()
			var ctx *connContext
			var ok bool
			if ctx, ok = fc.conns[addr]; !ok {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					req.Done(err)
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
				if req.Addr() == ctx.addr {
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

	for i := 0; i < len(fc.idle); i++ {
		ctx := <-fc.idle
		delCtx(ctx, false)
	}

	for i := 0; i < len(fc.delConn); i++ {
		e := <-fc.delConn
		delCtx(e.ctx, true)
	}

	for _, ctx := range fc.conns {
		delCtx(ctx, false)
	}
}

func (fc *fileClient) exe(ctx *connContext) {
	defer fc.wg.Done()

	ctx.idle = false

	req := ctx.req
	filenames := make([]string, len(req.files))
	for i, file := range req.files {
		filenames[i] = file.Filename
	}
	msg := &message.GetFiles{
		Names: filenames,
		Nonce: req.nonce,
	}

	data, err := msg.Serialize()
	if err != nil {
		req.Done(err)
		fc.idle <- ctx
		return
	}

	//ctx.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	err = p2p.WriteMsg(ctx.Conn, true, &p2p.Msg{
		CmdSetID: CmdSet,
		Cmd:      uint64(GetFilesCode),
		Size:     uint64(len(data)),
		Payload:  data,
	})
	if err != nil {
		fc.log.Error(fmt.Sprintf("requesFiles to %s error: %v", ctx.addr, err))
		req.Done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
		return
	}

	fc.log.Info(fmt.Sprintf("requestFiles to %s done", ctx.addr))

	sblocks, ablocks, err := fc.readBlocks(ctx)
	if err != nil {
		fc.log.Error(fmt.Sprintf("read blocks from %s error: %v", ctx.addr, err))
		req.rec(sblocks, ablocks)
		req.Done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
	} else {
		fc.log.Error(fmt.Sprintf("read blocks from %s done", ctx.addr))
		fc.idle <- ctx
		req.rec(sblocks, ablocks)
		req.Done(nil)
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
		// total blocks: snapshotblocks & accountblocks
		var total, count, sTotal, sCount, start uint64
		start = ctx.req.files[0].StartHeight
		for _, file := range ctx.req.files {
			if file.StartHeight < start {
				start = file.StartHeight
			}
			total += file.BlockNumbers
			sTotal += file.EndHeight - file.StartHeight + 1
		}

		// snapshot block is sorted
		sblocks = make([]*ledger.SnapshotBlock, sTotal)
		mblocks = make(map[types.Address][]*ledger.AccountBlock)

		// set read deadline
		//ctx.SetReadDeadline(time.Now().Add(total * time.Millisecond))
		fc.chain.Compressor().BlockParser(ctx, total, func(block ledger.Block, err error) {
			if err != nil {
				return
			}

			switch block.(type) {
			case *ledger.SnapshotBlock:
				count++

				block := block.(*ledger.SnapshotBlock)
				index := block.Height - start

				if sblocks[index] == nil {
					sblocks[block.Height-start] = block
					sCount++
					ctx.req.current = block.Height
				} else {
					fc.log.Warn("got repeated snapshotblock: %s/%d", hex.EncodeToString(block.Hash[:]), block.Height)
				}
			case *ledger.AccountBlock:
				count++

				block := block.(*ledger.AccountBlock)
				mblocks[block.AccountAddress] = append(mblocks[block.AccountAddress], block)
			default:
				fc.log.Error("got other block type")
			}
		})

		if count >= total && sCount >= sTotal {
			fc.log.Info(fmt.Sprintf("got %d/%d blocks, %d/%d ablocks", count, total, sCount, sTotal))
		} else {
			err = fmt.Errorf("incomplete blocks: %d/%d, %d/%d", count, total, sCount, sTotal)
		}

		return
	}
}
