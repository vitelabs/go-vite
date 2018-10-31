package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"io"
	net2 "net"
	"strconv"
	"sync"
	"time"
)

var fReadTimeout = 20 * time.Second
var fWriteTimeout = 20 * time.Second

type fileServer struct {
	port   uint16
	ln     net2.Listener
	record map[uint64]struct{} // use to record nonce
	term   chan struct{}
	log    log15.Logger
	wg     sync.WaitGroup
	chain  Chain
}

func newFileServer(port uint16, chain Chain) *fileServer {
	return &fileServer{
		port:   port,
		record: make(map[uint64]struct{}),
		log:    log15.New("module", "net/fileServer"),
		chain:  chain,
	}
}

func (s *fileServer) start() error {
	ln, err := net2.Listen("tcp", "0.0.0.0:"+strconv.FormatUint(uint64(s.port), 10))

	if err != nil {
		return err
	}

	s.ln = ln
	s.term = make(chan struct{})

	s.wg.Add(1)
	common.Go(s.listenLoop)

	return nil
}

func (s *fileServer) stop() {
	if s.term == nil {
		return
	}

	select {
	case <-s.term:
	default:
		close(s.term)

		if s.ln != nil {
			s.ln.Close()
		}

		s.wg.Wait()
	}
}

func (s *fileServer) listenLoop() {
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
			common.Go(func() {
				s.handleConn(conn)
			})
		}
	}
}

func (s *fileServer) handleConn(conn net2.Conn) {
	defer conn.Close()
	defer s.wg.Done()

	for {
		select {
		case <-s.term:
			return
		default:
			//conn.SetWriteDeadline(time.Now().Add(fReadTimeout))
			msg, err := p2p.ReadMsg(conn)
			if err != nil {
				s.log.Error(fmt.Sprintf("read message from %s error: %v", conn.RemoteAddr(), err))
				return
			}

			code := ViteCmd(msg.Cmd)
			if code != GetFilesCode {
				s.log.Error(fmt.Sprintf("got %d, need %d", code, GetFilesCode))
				return
			}

			req := new(message.GetFiles)
			if err = req.Deserialize(msg.Payload); err != nil {
				s.log.Error(fmt.Sprintf("parse message %s from %s error: %v", code, conn.RemoteAddr(), err))
				return
			}

			s.log.Info(fmt.Sprintf("receive %s from %s", req, conn.RemoteAddr()))

			// send files
			var n int64
			for _, filename := range req.Names {
				n, err = io.Copy(conn, s.chain.Compressor().FileReader(filename))

				if err != nil {
					s.log.Error(fmt.Sprintf("send file<%s> to %s error: %v", filename, conn.RemoteAddr(), err))
					return
				} else {
					s.log.Info(fmt.Sprintf("send file<%s> %d bytes to %s done", filename, n, conn.RemoteAddr()))
				}
			}
		}
	}
}

// @section fileClient

type connContext struct {
	net2.Conn
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
	dialer   *net2.Dialer
}

func newFileClient(chain Chain) *fileClient {
	return &fileClient{
		conns:    make(map[string]*connContext),
		_request: make(chan *fileRequest, 4),
		idle:     make(chan *connContext, 1),
		delConn:  make(chan *delCtxEvent, 1),
		chain:    chain,
		log:      log15.New("module", "net/fileClient"),
		dialer:   &net2.Dialer{Timeout: 3 * time.Second},
	}
}

func (fc *fileClient) start() {
	fc.term = make(chan struct{})

	fc.wg.Add(1)
	common.Go(fc.loop)
}

func (fc *fileClient) stop() {
	if fc.term == nil {
		return
	}

	select {
	case <-fc.term:
	default:
		close(fc.term)
		fc.wg.Wait()
	}
}

func (fc *fileClient) request(r *fileRequest) {
	select {
	case <-fc.term:
		fc.log.Warn(fmt.Sprintf("fc has stopped, can`t request %s", r))
	case fc._request <- r:
	}
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	wait := make([]*fileRequest, 0, 10)

	idleTimeout := 30 * time.Second
	ticker := time.NewTicker(idleTimeout)
	defer ticker.Stop()

	delCtx := func(ctx *connContext) {
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
				conn, err := fc.dialer.Dial("tcp", addr)
				if err != nil {
					req.Catch(err)
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
				common.Go(func() {
					fc.exe(ctx)
				})
			} else {
				wait = append(wait, req)
			}

		case ctx := <-fc.idle:
			ctx.idle = true
			ctx.idleT = time.Now()
			for i, req := range wait {
				if req.Addr() == ctx.addr {
					if i != len(wait)-1 {
						copy(wait[i:], wait[i+1:])
					}
					wait = wait[:len(wait)-1]

					ctx.req = req
					common.Go(func() {
						fc.exe(ctx)
					})
				}
			}

		case e := <-fc.delConn:
			delCtx(e.ctx)
			fc.log.Error(fmt.Sprintf("delete connection %s: %v", e.ctx.addr, e.err))

		case t := <-ticker.C:
			// remote the idle connection
			for _, ctx := range fc.conns {
				if ctx.idle && t.Sub(ctx.idleT) > idleTimeout {
					delCtx(ctx)
					fc.log.Warn(fmt.Sprintf("delete idle connection %s", ctx.addr))
				}
			}
		}
	}

	for i := 0; i < len(fc.idle); i++ {
		ctx := <-fc.idle
		delCtx(ctx)
	}

	for i := 0; i < len(fc.delConn); i++ {
		e := <-fc.delConn
		delCtx(e.ctx)
	}

	for _, ctx := range fc.conns {
		delCtx(ctx)
	}
}

func (fc *fileClient) exe(ctx *connContext) {
	ctx.idle = false

	req := ctx.req
	filenames := make([]string, len(req.files))
	for i, file := range req.files {
		filenames[i] = file.Filename
	}

	getFiles := &message.GetFiles{filenames, req.nonce}
	msg, err := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)

	if err != nil {
		fc.log.Error(fmt.Sprintf("send %s to %s error: %v", getFiles, ctx.addr, err))
		req.Catch(err)
		fc.idle <- ctx
		return
	}

	//ctx.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	if err = p2p.WriteMsg(ctx.Conn, msg); err != nil {
		fc.log.Error(fmt.Sprintf("send %s to %s error: %v", getFiles, ctx.addr, err))
		req.Catch(err)
		fc.delConn <- &delCtxEvent{ctx, err}
		return
	}

	fc.log.Info(fmt.Sprintf("send %s to %s done", getFiles, ctx.addr))

	sCount, aCount, err := fc.readBlocks(ctx)
	if err != nil {
		fc.log.Error(fmt.Sprintf("read blocks from %s error: %v", ctx.addr, err))
		fc.delConn <- &delCtxEvent{ctx, err}
		ctx.req.Catch(err)
	} else {
		fc.log.Info(fmt.Sprintf("read %d SnapshotBlocks %d AccountBlocks from %s", sCount, aCount, ctx.addr))
		fc.idle <- ctx
	}
}

//var errResTimeout = errors.New("wait for file response timeout")
var errFlieClientStopped = errors.New("fileClient stopped")

func (fc *fileClient) readBlocks(ctx *connContext) (uint64, uint64, error) {
	select {
	case <-fc.term:
		return 0, 0, errFlieClientStopped
	default:
		// total blocks: snapshotblocks & accountblocks
		var total, sTotal, sCount, aCount uint64

		start, end := ctx.req.from, ctx.req.to
		sTotal = end - start + 1

		for _, file := range ctx.req.files {
			total += file.BlockNumbers
		}

		usableAccountBlock := false
		fc.chain.Compressor().BlockParser(ctx, total, func(block ledger.Block, err error) {
			if err != nil {
				return
			}

			switch block.(type) {
			case *ledger.SnapshotBlock:
				block := block.(*ledger.SnapshotBlock)
				if block == nil {
					return
				}

				if block.Height < start || block.Height > end {
					return
				}

				// snapshotblock is in band, so follow account blocks will be available
				usableAccountBlock = true
				sCount++
				ctx.req.rec.receiveSnapshotBlock(block)
				ctx.req.current = block.Height

			case *ledger.AccountBlock:
				block := block.(*ledger.AccountBlock)
				if block == nil {
					return
				}

				if usableAccountBlock {
					aCount++
					ctx.req.rec.receiveAccountBlock(block)
				}
			}
		})

		if sCount != sTotal {
			return sCount, aCount, fmt.Errorf("incomplete file %d/%d snapshotblocks", sCount, sTotal)
		}

		return sCount, aCount, nil
	}
}
