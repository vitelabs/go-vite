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

type conn struct {
	net2.Conn
	file   *ledger.CompressedFileMeta
	height uint64 // block has got
	peer   Peer
	idle   bool
	idleT  time.Time
}

type filesEvent struct {
	files
	sender Peer
}

type fileClient struct {
	idleChan  chan *conn
	delChan   chan *conn
	filesChan chan *filesEvent

	chain Chain

	handler reqRec

	dialer *net2.Dialer

	pool cPool

	term chan struct{}
	log  log15.Logger
	wg   sync.WaitGroup
}

type cPool interface {
	add(request *chunkRequest)
}

func newFileClient(chain Chain, pool cPool, handler reqRec) *fileClient {
	return &fileClient{
		idleChan:  make(chan *conn),
		delChan:   make(chan *conn),
		filesChan: make(chan *filesEvent),
		chain:     chain,
		log:       log15.New("module", "net/fileClient"),
		dialer:    &net2.Dialer{Timeout: 3 * time.Second},
		pool:      pool,
		handler:   handler,
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

func (fc *fileClient) removePeer(conns map[string]*conn, fRecord map[string]*fileSrc, pFiles map[string]*pFileRecord, sender Peer) {
	id := sender.ID()

	if conn, ok := conns[id]; ok {
		delete(conns, id)
		conn.Close()
	}

	delete(pFiles, id)

	for _, r := range fRecord {
		r.peers = r.peers.delete(id)
	}
}

func (fc *fileClient) usePeer(conns map[string]*conn, record map[string]*fileSrc, pFiles map[string]*pFileRecord, sender Peer) {
	id := sender.ID()
	// peer is busy
	if c, ok := conns[id]; ok && !c.idle {
		return
	}

	if fileRecord, ok := pFiles[sender.ID()]; ok {
		// retrieve the last file
		for {
			file := fileRecord.files[fileRecord.index]
			r := record[file.Filename]

			fileState := r.state
			if fileState == reqWaiting || fileState == reqError {
				if err := fc.doRequest(conns, file, sender); err != nil {
					fc.removePeer(conns, record, pFiles, sender)
				} else {
					r.state = reqPending
				}

				break
			} else if fileRecord.index == 0 {
				break
			}

			fileRecord.index--
		}
	}
}

func (fc *fileClient) requestFile(conns map[string]*conn, record map[string]*fileSrc, pFiles map[string]*pFileRecord, file *ledger.CompressedFileMeta) {
	if src, ok := record[file.Filename]; ok {
		var id string
		for _, peer := range src.peers {
			id = peer.ID()
			if c, ok := conns[id]; ok && !c.idle {
				continue
			}

			if err := fc.doRequest(conns, file, peer); err != nil {
				fc.removePeer(conns, record, pFiles, peer)
			} else {
				src.state = reqPending
				return
			}
		}

		// no peers, get chunks
		src.state = reqError
		fc.pool.add(&chunkRequest{
			from: file.StartHeight,
			to:   file.EndHeight,
		})
	}
}

func (fc *fileClient) doRequest(conns map[string]*conn, file *ledger.CompressedFileMeta, sender Peer) error {
	id := sender.ID()

	if _, ok := conns[id]; !ok {
		tcp, err := fc.dialer.Dial("tcp", sender.FileAddress().String())
		if err != nil {
			return err
		}

		conns[id] = &conn{
			Conn:  tcp,
			peer:  sender,
			idle:  false,
			idleT: time.Now(),
		}
	}

	conns[id].file = file

	// exec
	common.Go(func() {
		fc.exec(conns[id])
	})

	return nil
}

func (fc *fileClient) gotFiles(fs files, sender Peer) {
	select {
	case <-fc.term:
	case fc.filesChan <- &filesEvent{fs, sender}:
	}
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	conns := make(map[string]*conn)
	record := make(map[string]*fileSrc)
	pFiles := make(map[string]*pFileRecord)

	idleTimeout := 30 * time.Second
	ticker := time.NewTicker(idleTimeout)
	defer ticker.Stop()

	delConn := func(ctx *conn) {
		delete(conns, ctx.peer.ID())
		ctx.Close()
	}

loop:
	for {
		select {
		case <-fc.term:
			break loop

		case e := <-fc.filesChan: // got new files
			files, sender := e.files, e.sender
			if _, ok := pFiles[sender.ID()]; ok {
				break
			}
			pFiles[sender.ID()] = &pFileRecord{files, len(files) - 1}
			for _, file := range files {
				if _, ok := record[file.Filename]; !ok {
					record[file.Filename] = new(fileSrc)
				}
				record[file.Filename].peers = append(record[file.Filename].peers, sender)
			}

			fc.usePeer(conns, record, pFiles, sender)

		case ctx := <-fc.idleChan:
			if ctx.file != nil {
				if r, ok := record[ctx.file.Filename]; ok {
					r.state = reqDone
				}
			}

			ctx.idle = true
			ctx.idleT = time.Now()

			fc.usePeer(conns, record, pFiles, ctx.peer)

		case conn := <-fc.delChan:
			delConn(conn)
			fc.log.Error(fmt.Sprintf("delete connection %s", conn.RemoteAddr()))

			if file := conn.file; file != nil {
				if record[file.Filename].state != reqDone {
					miss := conn.file.EndHeight - conn.height
					if miss > file2Chunk {
						// retry file
						fc.requestFile(conns, record, pFiles, file)
					} else {
						record[file.Filename].state = reqDone
						// use chunk
						fc.pool.add(&chunkRequest{
							from: conn.height + 1,
							to:   conn.file.EndHeight,
						})
					}
				}
			}

		case t := <-ticker.C:
			// remote the idle connection
			for _, conn := range conns {
				if conn.idle && t.Sub(conn.idleT) > idleTimeout {
					delConn(conn)
					fc.log.Warn(fmt.Sprintf("delete idle connection %s", conn.RemoteAddr()))
				}
			}
		}
	}

	for _, ctx := range conns {
		delConn(ctx)
	}
}

func (fc *fileClient) idle(ctx *conn) {
	select {
	case <-fc.term:
	case fc.idleChan <- ctx:
	}
}

func (fc *fileClient) delete(ctx *conn) {
	select {
	case <-fc.term:
	case fc.delChan <- ctx:
	}
}

func (fc *fileClient) exec(ctx *conn) {
	ctx.idle = false

	getFiles := &message.GetFiles{
		Names: []string{ctx.file.Filename},
	}

	msg, err := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)

	if err != nil {
		fc.log.Error(fmt.Sprintf("send %s to %s error: %v", getFiles, ctx.RemoteAddr(), err))
		ctx.file = nil
		fc.idle(ctx)
		return
	}

	ctx.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	if err = p2p.WriteMsg(ctx.Conn, msg); err != nil {
		fc.log.Error(fmt.Sprintf("send %s to %s error: %v", getFiles, ctx.RemoteAddr(), err))
		fc.delete(ctx)
		return
	}

	fc.log.Info(fmt.Sprintf("send %s to %s done", getFiles, ctx.RemoteAddr()))

	if err = fc.receiveFile(ctx); err != nil {
		fc.log.Error(fmt.Sprintf("receive file from %s error: %v", ctx.RemoteAddr(), err))
		fc.delete(ctx)
	} else {
		fc.idle(ctx)
	}
}

var errFlieClientStopped = errors.New("fileClient stopped")

func (fc *fileClient) receiveFile(ctx *conn) error {
	select {
	case <-fc.term:
		return errFlieClientStopped
	default:
		// total blocks: snapshotblocks & accountblocks
		var sCount, aCount uint64

		file := ctx.file

		fc.chain.Compressor().BlockParser(ctx, file.BlockNumbers, func(block ledger.Block, err error) {
			if err != nil {
				return
			}

			switch block.(type) {
			case *ledger.SnapshotBlock:
				block := block.(*ledger.SnapshotBlock)
				if block == nil {
					return
				}

				sCount++
				fc.handler.receiveSnapshotBlock(block)
				ctx.height = block.Height

			case *ledger.AccountBlock:
				block := block.(*ledger.AccountBlock)
				if block == nil {
					return
				}

				aCount++
				fc.handler.receiveAccountBlock(block)
			}
		})

		sTotal := file.EndHeight - file.StartHeight + 1
		if sCount < sTotal {
			return fmt.Errorf("incomplete file %s %d/%d", file.Filename, sCount, sTotal)
		}

		fc.log.Info(fmt.Sprintf("receive %d SnapshotBlocks %d AccountBlocks of file %s from %s", sCount, aCount, file.Filename, ctx.RemoteAddr()))
		return nil
	}
}
