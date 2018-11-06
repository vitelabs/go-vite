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
	"sort"
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
			//conn.SetReadDeadline(time.Now().Add(fReadTimeout))
			msg, err := p2p.ReadMsg(conn)
			if err != nil {
				s.log.Warn(fmt.Sprintf("read message from %s error: %v", conn.RemoteAddr(), err))
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
	done   bool // file request done
}

type filesEvent struct {
	files
	sender Peer
}

type fileClient struct {
	idleChan  chan *conn
	delChan   chan *conn
	filesChan chan *filesEvent
	slots     chan struct{} // control concurrency

	chain Chain

	handler blockReceiver

	dialer *net2.Dialer

	pool cPool

	term chan struct{}
	log  log15.Logger
	wg   sync.WaitGroup

	target uint64
	should bool
	busy   bool
}

type cPool interface {
	exec(request *chunkRequest)
	start()
}

func newFileClient(chain Chain, pool cPool, handler blockReceiver) *fileClient {
	return &fileClient{
		idleChan:  make(chan *conn, 1),
		delChan:   make(chan *conn, 1),
		filesChan: make(chan *filesEvent, 10),
		slots:     make(chan struct{}, 1),
		chain:     chain,
		log:       log15.New("module", "net/fileClient"),
		dialer:    &net2.Dialer{Timeout: 3 * time.Second},
		pool:      pool,
		handler:   handler,
		should:    true,
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

func (fc *fileClient) threshold(current uint64) {
	if current+3600 > fc.target {
		fc.should = true
	} else {
		fc.should = false
	}
}

func (fc *fileClient) occupy() {
	fc.slots <- struct{}{}
}

func (fc *fileClient) release() {
	<-fc.slots
}

func (fc *fileClient) removePeer(conns map[string]*conn, fRecord map[string]*fileState, pFiles map[string]files, sender Peer) {
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

func (fc *fileClient) usePeer(conns map[string]*conn, record map[string]*fileState, pFiles map[string]files, sender Peer) {
	id := sender.ID()
	// peer is busy
	if c, ok := conns[id]; ok && !c.idle {
		return
	}

	if files, ok := pFiles[sender.ID()]; ok {
		// retrieve files from low to high
		for _, file := range files {
			if r, ok := record[file.Filename]; ok {
				if r.state == reqWaiting || r.state == reqError {
					r.state = reqPending
					if err := fc.doRequest(conns, file, sender); err != nil {
						r.state = reqError
						fc.removePeer(conns, record, pFiles, sender)
					}

					break
				}
			}
		}
	}
}

func (fc *fileClient) requestFile(conns map[string]*conn, record map[string]*fileState, pFiles map[string]files, file *ledger.CompressedFileMeta) {
	if r, ok := record[file.Filename]; ok {
		var id string
		for _, peer := range r.peers {
			id = peer.ID()
			if c, ok := conns[id]; ok && !c.idle {
				continue
			}

			r.state = reqPending
			if err := fc.doRequest(conns, file, peer); err != nil {
				r.state = reqError
				fc.removePeer(conns, record, pFiles, peer)
			} else {
				fc.target = file.EndHeight
				return
			}
		}
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
			idle:  true,
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

func (fc *fileClient) nextFile(fileList files, record map[string]*fileState) (file *ledger.CompressedFileMeta) {
	for _, file := range fileList {
		if r, ok := record[file.Filename]; ok {
			if r.state == reqWaiting || r.state == reqError {
				return file
			}
		}
	}

	return nil
}

func (fc *fileClient) gotFiles(fs files, sender Peer) {
	select {
	case <-fc.term:
	case fc.filesChan <- &filesEvent{fs, sender}:
	}
}

type fileState struct {
	file *ledger.CompressedFileMeta
	peers
	state reqState
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	conns := make(map[string]*conn)
	record := make(map[string]*fileState)
	pFiles := make(map[string]files)
	fileList := make(files, 0, 10)

	idleTimeout := 20 * time.Second
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
			pFiles[sender.ID()] = files
			for _, file := range files {
				if _, ok := record[file.Filename]; !ok {
					record[file.Filename] = &fileState{file: file}
				}
				record[file.Filename].peers = append(record[file.Filename].peers, sender)
			}

			fileList = fileList[:0]
			for _, r := range record {
				fileList = append(fileList, r.file)
			}
			sort.Sort(fileList)

			//fc.usePeer(conns, record, pFiles, sender)

		case ctx := <-fc.idleChan: // a job done
			if r, ok := record[ctx.file.Filename]; ok {
				if ctx.done {
					r.state = reqDone
				} else {
					r.state = reqError
				}
			}

			ctx.idle = true
			ctx.idleT = time.Now()

			//fc.usePeer(conns, record, pFiles, ctx.peer)

		case conn := <-fc.delChan: // a job error
			delConn(conn)
			fc.log.Error(fmt.Sprintf("delete connection %s", conn.RemoteAddr()))

			file := conn.file
			if file == nil {
				break
			}

			// retry
			if err := fc.doRequest(conns, file, conn.peer); err == nil {
				break
			}

			// clean
			fc.removePeer(conns, record, pFiles, conn.peer)

			if r, ok := record[file.Filename]; ok {
				miss := file.EndHeight - conn.height
				if miss > file2Chunk {
					r.state = reqError
					// retry file
					fc.requestFile(conns, record, pFiles, file)
				} else {
					r.state = reqDone
					// use chunk
					fc.pool.exec(&chunkRequest{
						from: conn.height + 1,
						to:   file.EndHeight,
					})
				}
			}

		case <-ticker.C:
			if fc.busy {
				break
			}

			if fc.should {
				if file := fc.nextFile(fileList, record); file == nil {
					fc.pool.start()
					break loop
				} else {
					fc.busy = true
					//fc.occupy()
					fc.requestFile(conns, record, pFiles, file)
				}
			}

			//for _, conn := range conns {
			//	if conn.idle && t.Sub(conn.idleT) > idleTimeout {
			//		delConn(conn)
			//		fc.log.Warn(fmt.Sprintf("delete idle connection %s", conn.RemoteAddr()))
			//	}
			//}

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
	//defer fc.release()
	defer func() {
		fc.busy = false
	}()

	select {
	case <-fc.term:
		return
	default:
		// next
	}

	ctx.idle = false

	getFiles := &message.GetFiles{
		Names: []string{ctx.file.Filename},
	}

	msg, err := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)

	if err != nil {
		ctx.done = false
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
		ctx.done = true
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
