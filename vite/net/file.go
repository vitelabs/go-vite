package net

import (
	"fmt"
	"github.com/pkg/errors"
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
	go s.readLoop()

	return nil
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

func (f *fileServer) handleConn(conn net2.Conn) {
	defer conn.Close()
	defer f.wg.Done()

	for {
		select {
		case <-f.term:
			return
		default:
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
					var n int64
					n, err = io.Copy(conn, f.chain.Compressor().FileReader(filename))

					if err != nil {
						f.log.Error(fmt.Sprintf("send file<%s> to %s error: %v", filename, conn.RemoteAddr(), err))
						return
					} else {
						f.log.Info(fmt.Sprintf("send file<%s> %d bytes to %s done", filename, n, conn.RemoteAddr()))
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
}

func newFileClient(chain Chain) *fileClient {
	return &fileClient{
		conns:    make(map[string]*connContext),
		_request: make(chan *fileRequest, 4),
		idle:     make(chan *connContext, 1),
		delConn:  make(chan *delCtxEvent, 1),
		chain:    chain,
		log:      log15.New("module", "net/fileClient"),
	}
}

func (fc *fileClient) start() {
	fc.term = make(chan struct{})

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
	select {
	case <-fc.term:
	case fc._request <- r:
	}
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
				conn, err := net2.Dial("tcp", addr)
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

	//ctx.SetWriteDeadline(time.Now().Add(3 * time.Second))
	err = p2p.WriteMsg(ctx.Conn, true, &p2p.Msg{
		CmdSetID: CmdSet,
		Cmd:      uint64(GetFilesCode),
		Size:     uint64(len(data)),
		Payload:  data,
	})
	if err != nil {
		fc.log.Error(fmt.Sprintf("send fileRequest %s to %s error: %v", ctx.req, ctx.addr, err))
		req.Done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
		return
	}

	fc.log.Info(fmt.Sprintf("send fileRequest %s to %s done", ctx.req, ctx.addr))

	sblocks, ablocks, err := fc.readBlocks(ctx)
	if err != nil {
		fc.log.Error(fmt.Sprintf("read blocks from %s error: %v", ctx.addr, err))
		req.rec(sblocks, ablocks)
		req.Done(err)
		fc.delConn <- &delCtxEvent{ctx, err}
	} else {
		fc.log.Info(fmt.Sprintf("read blocks from %s done", ctx.addr))
		fc.idle <- ctx
		req.rec(sblocks, ablocks)
		req.Done(nil)
	}
}

//var errResTimeout = errors.New("wait for file response timeout")
var errFlieClientStopped = errors.New("fileClient stopped")

func (fc *fileClient) readBlocks(ctx *connContext) (sblocks []*ledger.SnapshotBlock, ablocks []*ledger.AccountBlock, err error) {
	select {
	case <-fc.term:
		err = errFlieClientStopped
		return
	default:
		// total blocks: snapshotblocks & accountblocks
		var total, sTotal, sCount, aTotal, aCount uint64

		start, end := ctx.req.from, ctx.req.to
		sTotal = end - start + 1

		for _, file := range ctx.req.files {
			total += file.BlockNumbers
			aTotal += file.BlockNumbers - (file.EndHeight - file.StartHeight + 1)
		}

		// snapshot block is sorted
		sblocks = make([]*ledger.SnapshotBlock, sTotal)
		ablocks = make([]*ledger.AccountBlock, aTotal)

		// set read deadline
		//ctx.SetReadDeadline(time.Now().Add(total * int64(time.Millisecond)))
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

				index := block.Height - start
				if sblocks[index] == nil {
					sblocks[index] = block
					sCount++
					ctx.req.current = block.Height
				}

			case *ledger.AccountBlock:
				block := block.(*ledger.AccountBlock)
				if block == nil {
					return
				}

				if usableAccountBlock {
					ablocks[aCount] = block
					aCount++
				}
			}
		})

		if sCount == sTotal {
			fc.log.Info(fmt.Sprintf("read %d snapshotblocks, %d accountblocks", sCount, aCount))
		} else {
			err = fmt.Errorf("read %d/%d snapshotblocks, %d accountblocks", sCount, sTotal, aCount)
		}

		return sblocks[:sCount], ablocks[:aCount], nil
	}
}
