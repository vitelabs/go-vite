package net

import (
	"fmt"
	"io"
	"math/rand"
	net2 "net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

const fReadTimeout = 20 * time.Second
const fWriteTimeout = 20 * time.Second
const fileTimeout = 5 * time.Minute

type fileServer struct {
	addr   string
	ln     net2.Listener
	record map[uint64]struct{} // use to record nonce
	term   chan struct{}
	log    log15.Logger
	wg     sync.WaitGroup
	chain  Chain

	mu    sync.Mutex
	conns map[string]net2.Conn
}

func newFileServer(addr string, chain Chain) *fileServer {
	return &fileServer{
		addr:   addr,
		record: make(map[uint64]struct{}),
		log:    log15.New("module", "net/fileServer"),
		chain:  chain,
		conns:  make(map[string]net2.Conn),
	}
}

func (s *fileServer) start() error {
	ln, err := net2.Listen("tcp", s.addr)

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

		s.mu.Lock()
		for addr, c := range s.conns {
			c.Close()
			delete(s.conns, addr)
		}
		s.mu.Unlock()

		s.wg.Wait()
	}
}

func (s *fileServer) listenLoop() {
	defer s.wg.Done()

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ne, ok := err.(net2.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				time.Sleep(tempDelay)

				continue
			}

			return
		}

		s.wg.Add(1)
		common.Go(func() {
			s.handleConn(conn)
		})
	}
}

func (s *fileServer) handleConn(conn net2.Conn) {
	defer conn.Close()
	defer s.wg.Done()

	s.mu.Lock()
	s.conns[conn.RemoteAddr().String()] = conn
	s.mu.Unlock()

	for {
		conn.SetReadDeadline(time.Now().Add(fReadTimeout))
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
			conn.SetWriteDeadline(time.Now().Add(fileTimeout))
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

// @section fileClient

type conn struct {
	net2.Conn
	file   *ledger.CompressedFileMeta
	height uint64 // block has got
	peer   Peer
	idle   bool
	idleT  time.Time
	done   bool  // file request done
	speed  int64 // byte/s
}

type connList []*conn

func (l connList) Len() int {
	return len(l)
}

func (l connList) Less(i, j int) bool {
	return l[i].speed > l[j].speed
}

func (l connList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type filesEvent struct {
	files
	sender Peer
}

type fileClient struct {
	idleChan  chan *conn
	delChan   chan *conn
	filesChan chan *filesEvent

	finishCallbacks []func(end uint64)

	chain Chain

	rec blockReceiver

	peers *peerSet

	dialer *net2.Dialer

	running int32
	term    chan struct{}
	log     log15.Logger
	wg      sync.WaitGroup

	to uint64
}

func newFileClient(chain Chain, rec blockReceiver, peers *peerSet) *fileClient {
	return &fileClient{
		idleChan:  make(chan *conn, 1),
		delChan:   make(chan *conn, 1),
		filesChan: make(chan *filesEvent, 10),
		chain:     chain,
		peers:     peers,
		log:       log15.New("module", "net/fileClient"),
		dialer:    &net2.Dialer{Timeout: 3 * time.Second},
		rec:       rec,
	}
}

func (fc *fileClient) start() {
	if atomic.CompareAndSwapInt32(&fc.running, 0, 1) {
		fc.term = make(chan struct{})

		fc.wg.Add(1)
		common.Go(fc.loop)
	}
}

func (fc *fileClient) stop() {
	if fc.term == nil {
		return
	}

	defer atomic.CompareAndSwapInt32(&fc.running, 1, 0)

	select {
	case <-fc.term:
	default:
		close(fc.term)
		fc.wg.Wait()
	}
}

func (fc *fileClient) threshold(to uint64) {
	fc.to = to
}

func (fc *fileClient) allFileDownloaded(end uint64) {
	for _, fn := range fc.finishCallbacks {
		fn(end)
	}
}

func (fc *fileClient) subAllFileDownloaded(fn func(uint64)) {
	fc.finishCallbacks = append(fc.finishCallbacks, fn)
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
				if file.StartHeight > fc.to {
					break
				}

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

func chooseIdleConn(list connList) *conn {
	sort.Sort(list)
	total := len(list)

	// should create new file connection
	createNew := rand.Intn(10) > 5

	for i, c := range list {
		if c.idle {
			continue
		}

		if i < total/3 {
			return c
		}

		if createNew {
			return nil
		}

		return c
	}

	return nil
}

func (fc *fileClient) requestFile(conns map[string]*conn, record map[string]*fileState, pFiles map[string]files, file *ledger.CompressedFileMeta) (err error) {
	if r, ok := record[file.Filename]; ok {
		if len(r.peers) > 0 {

			var cList connList
			var id string
			for _, p := range r.peers {
				id = p.ID()
				if c, ok := conns[id]; ok {
					cList = append(cList, c)
				}
			}

			// choose a file connection
			if c := chooseIdleConn(cList); c != nil {
				c.file = file
				fc.exec(c)
				return nil
			}

			ids := rand.Perm(len(r.peers))
			// random a idle peer
			for _, idx := range ids {
				// may be remove peers from r.peers
				if idx >= len(r.peers) {
					continue
				}

				p := r.peers[idx]
				id = p.ID()

				// connection is busy
				if c, ok := conns[id]; ok && !c.idle {
					continue
				}

				r.state = reqPending
				if err = fc.doRequest(conns, file, p); err != nil {
					r.state = reqError
					fc.removePeer(conns, record, pFiles, p)
				} else {
					// download
					return
				}
			}
		} else {
			return fmt.Errorf("no peers for file %s", file.Filename)
		}
	}

	return nil
}

var errDisconnectedPeer = errors.New("peer has disconnected, can`t download file")

func (fc *fileClient) doRequest(conns map[string]*conn, file *ledger.CompressedFileMeta, sender Peer) error {
	id := sender.ID()

	if !fc.peers.Has(id) {
		return errDisconnectedPeer
	}

	var c *conn
	var ok bool
	if c, ok = conns[id]; !ok {
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

		c = conns[id]
	}

	c.file = file
	fc.exec(c)

	return nil
}

func (fc *fileClient) nextFile(fileList files, record map[string]*fileState) (file *ledger.CompressedFileMeta) {
	for _, file = range fileList {
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

	conns := make(map[string]*conn) // peerId: conn
	record := make(map[string]*fileState)
	pFiles := make(map[string]files)
	fileList := make(files, 0, 10)

	delConn := func(c *conn) {
		delete(conns, c.peer.ID())
		c.Close()
	}

	var jobTicker <-chan time.Time

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

			if jobTicker == nil {
				t := time.NewTicker(time.Second)
				jobTicker = t.C
				defer t.Stop()
				//fc.usePeer(conns, record, pFiles, e.sender)
			}

		case c := <-fc.idleChan: // a job done
			if r, ok := record[c.file.Filename]; ok {
				if c.done {
					r.state = reqDone
				} else {
					r.state = reqError
				}
			}

			c.idle = true
			c.idleT = time.Now()

			//fc.usePeer(conns, record, pFiles, c.peer)

		case conn := <-fc.delChan: // a job error
			delConn(conn)
			fc.log.Error(fmt.Sprintf("delete file connection %s", conn.RemoteAddr()))

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

			fc.requestFile(conns, record, pFiles, file)

		case <-jobTicker:
			if file := fc.nextFile(fileList, record); file == nil {
				// some files are downloading
				done := true
				for _, c := range conns {
					if !c.idle {
						done = false
						break
					}
				}

				if done {
					// all files down
					fc.allFileDownloaded(fileList[len(fileList)-1].EndHeight)
					break loop
				}
			} else if file.StartHeight <= fc.to {
				fc.requestFile(conns, record, pFiles, file)
			}
		}
	}

	for _, c := range conns {
		delConn(c)
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
	select {
	case <-fc.term:
		return
	default:
		// next
	}

	ctx.idle = false

	common.Go(func() {
		fc.download(ctx)
	})
}

func (fc *fileClient) download(ctx *conn) {
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
		fc.log.Error(fmt.Sprintf("receive file %s from %s error: %v", ctx.file.Filename, ctx.RemoteAddr(), err))
		fc.delete(ctx)
	} else {
		ctx.done = true
		fc.idle(ctx)
		fc.rec.done(&filePiece{
			from: ctx.file.StartHeight,
			to:   ctx.file.EndHeight,
		})
	}
}

var errFlieClientStopped = errors.New("fileClient stopped")

func (fc *fileClient) receiveFile(ctx *conn) error {
	ctx.SetReadDeadline(time.Now().Add(fileTimeout))
	defer ctx.SetReadDeadline(time.Time{})

	select {
	case <-fc.term:
		return errFlieClientStopped
	default:
		// total blocks: snapshotblocks & accountblocks
		var sCount, aCount uint64

		file := ctx.file

		var fileError error

		start := time.Now()
		fc.chain.Compressor().BlockParser(ctx, file.BlockNumbers, func(block ledger.Block, err error) {
			if err != nil || fileError != nil {
				ctx.Close()
				return
			}

			switch block.(type) {
			case *ledger.SnapshotBlock:
				block := block.(*ledger.SnapshotBlock)
				if block == nil {
					return
				}

				sCount++
				fileError = fc.rec.receiveSnapshotBlock(block, ctx.peer)
				ctx.height = block.Height

			case *ledger.AccountBlock:
				block := block.(*ledger.AccountBlock)
				if block == nil {
					return
				}

				aCount++
				fileError = fc.rec.receiveAccountBlock(block, ctx.peer)
			}
		})

		if fileError != nil {
			return fileError
		}

		sTotal := file.EndHeight - file.StartHeight + 1
		if sCount < sTotal {
			return fmt.Errorf("incomplete file %s %d/%d", file.Filename, sCount, sTotal)
		}

		elapse := time.Now().Sub(start).Seconds()
		ctx.speed = file.FileSize / int64(elapse)

		fc.log.Info(fmt.Sprintf("receive %d SnapshotBlocks %d AccountBlocks of file %s from %s", sCount, aCount, file.Filename, ctx.RemoteAddr()))
		return nil
	}
}
