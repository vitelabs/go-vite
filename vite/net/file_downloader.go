package net

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	net2 "net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

var errNoSuitablePeers = errors.New("no suitable peers")
var errFileConnExist = errors.New("fileConn has exist")
var errFileConnClosed = errors.New("file connection has closed")
var errPeerDialing = errors.New("peer is dialing")

type fileConnState byte

const (
	fileConnStateNew fileConnState = iota
	fileConnStateIdle
	fileConnStateBusy
	fileConnStateClosed
)

type downloadErrorCode byte

const (
	downloadIncompleteErr downloadErrorCode = iota + 1
	downloadPackMsgErr
	downloadSendErr
	downloadReceiveErr
	downloadParseErr
	downloadOtherErr
)

type downloadError struct {
	code downloadErrorCode
	err  string
}

func (e downloadError) Fatal() bool {
	return e.code == downloadReceiveErr
}

func (e downloadError) Error() string {
	return e.err
}

type fileParser interface {
	BlockParser(reader io.Reader, blockNum uint64, processFunc func(block ledger.Block, err error))
}

type FileConnStatus struct {
	Id    string
	Addr  string
	Speed float64
}

type fileConn struct {
	net2.Conn
	id     peerId
	busy   int32   // atomic
	t      int64   // timestamp
	speed  float64 // download speed, byte/s
	parser fileParser
	closed int32
	log    log15.Logger
}

func newFileConn(conn net2.Conn, id peerId, parser fileParser, log log15.Logger) *fileConn {
	return &fileConn{
		Conn:   conn,
		id:     id,
		parser: parser,
		log:    log,
	}
}

func (f *fileConn) status() FileConnStatus {
	return FileConnStatus{
		Id:    f.id,
		Addr:  f.RemoteAddr().String(),
		Speed: f.speed,
	}
}

func (f *fileConn) isBusy() bool {
	return atomic.LoadInt32(&f.busy) == 1
}

func (f *fileConn) download(file File, rec blockReceiver) (outerr *downloadError) {
	f.setBusy()
	defer f.idle()

	f.log.Info(fmt.Sprintf("begin download <file %s> from %s", file.Filename, f.RemoteAddr()))

	getFiles := &message.GetFiles{
		Names: []string{file.Filename},
	}

	msg, err := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)
	if err != nil {
		f.log.Error(fmt.Sprintf("pack GetFilesMsg<file %s> to %s error: %v", file.Filename, f.RemoteAddr(), err))
		return &downloadError{
			code: downloadPackMsgErr,
			err:  err.Error(),
		}
	}

	f.Conn.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	if err = p2p.WriteMsg(f.Conn, msg); err != nil {
		f.log.Error(fmt.Sprintf("write GetFilesMsg<file %s> to %s error: %v", file.Filename, f.RemoteAddr(), err))
		return &downloadError{
			code: downloadSendErr,
			err:  err.Error(),
		}
	}

	var sCount, aCount uint64

	start := time.Now()
	// todo fileTimeout can be a flexible value, like calc through fileSize and download speed
	f.Conn.SetReadDeadline(time.Now().Add(fileTimeout))
	f.parser.BlockParser(f.Conn, file.BlockNumbers, func(block ledger.Block, err error) {
		// Fatal error, then close the connection to interrupt the stream
		if outerr != nil && outerr.Fatal() {
			f.log.Error(fmt.Sprintf("download <file %s> from %s error: %v, close connection", file.Filename, f.RemoteAddr(), outerr))
			f.close()
		}

		switch block.(type) {
		case *ledger.SnapshotBlock:
			block := block.(*ledger.SnapshotBlock)
			err = rec.receiveSnapshotBlock(block)

			if block.Height >= file.StartHeight && block.Height <= file.EndHeight {
				sCount++
			}

		case *ledger.AccountBlock:
			block := block.(*ledger.AccountBlock)
			aCount++
			err = rec.receiveAccountBlock(block)
		}

		if err != nil {
			outerr = &downloadError{
				code: downloadReceiveErr,
				err:  err.Error(),
			}
		}
	})

	sTotal := file.EndHeight - file.StartHeight + 1
	aTotal := file.BlockNumbers - sTotal

	if sCount < sTotal || aCount < aTotal {
		outerr = &downloadError{
			code: downloadIncompleteErr,
			err:  fmt.Sprintf("incomplete <file %s> %d/%d, %d/%d", file.Filename, sCount, sTotal, aCount, aTotal),
		}
	}

	if outerr != nil {
		f.speed = 0
	} else {
		// bytes/s
		f.speed = float64(file.FileSize) / (time.Now().Sub(start).Seconds() + 1)
	}

	return
}

func (f *fileConn) setBusy() {
	atomic.StoreInt32(&f.busy, 1)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *fileConn) idle() {
	atomic.StoreInt32(&f.busy, 0)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *fileConn) close() error {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return f.Conn.Close()
	}

	return errFileConnClosed
}

type fileConns []*fileConn

func (fl fileConns) Len() int {
	return len(fl)
}

func (fl fileConns) Less(i, j int) bool {
	return fl[i].speed > fl[j].speed
}

func (fl fileConns) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

func (fl fileConns) del(i int) fileConns {
	total := len(fl)
	if i < total {
		fl[i].close()

		if i != total-1 {
			copy(fl[i:], fl[i+1:])
		}
		return fl[:total-1]
	}

	return fl
}

type filePeer struct {
	id   peerId
	addr string
	fail int32
}

type FilePoolStatus struct {
	Conns []FileConnStatus
}

type filePeerPool struct {
	mu sync.Mutex

	mf map[filename]map[peerId]struct{} // find which peer can download the spec file
	mp map[peerId]*filePeer             // manage peers

	mi map[peerId]int // manage fileConns, corresponding value is the index of `filePeerPool.l`
	l  fileConns      // fileConns sort by speed, from fast to slow
}

func newPool() *filePeerPool {
	return &filePeerPool{
		mf: make(map[filename]map[peerId]struct{}),
		mp: make(map[peerId]*filePeer),
		mi: make(map[peerId]int),
	}
}

func (fp *filePeerPool) status() FilePoolStatus {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	conns := make([]FileConnStatus, len(fp.l))

	for i := 0; i < len(fp.l); i++ {
		conns[i] = fp.l[i].status()
	}

	return FilePoolStatus{
		Conns: conns,
	}
}

func (fp *filePeerPool) addPeer(files []filename, addr string, id peerId) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	for _, name := range files {
		if _, ok := fp.mf[name]; !ok {
			fp.mf[name] = make(map[peerId]struct{})
		}
		fp.mf[name][id] = struct{}{}
	}

	if _, ok := fp.mp[id]; !ok {
		fp.mp[id] = &filePeer{id, addr, 0}
	} else {
		fp.mp[id].addr = addr
	}
}

// delete filePeer and connection
func (fp *filePeerPool) delPeer(id peerId) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delPeerLocked(id)
}

func (fp *filePeerPool) delPeerLocked(id peerId) {
	for _, m := range fp.mf {
		delete(m, id)
	}

	delete(fp.mp, id)

	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

func (fp *filePeerPool) addConn(c *fileConn) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, ok := fp.mi[c.id]; ok {
		return errFileConnExist
	}

	fp.l = append(fp.l, c)
	fp.mi[c.id] = len(fp.l) - 1
	return nil
}

func (fp *filePeerPool) catch(id peerId) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if p, ok := fp.mp[id]; ok {
		p.fail++

		// fail too many times, then delete the peer
		if p.fail > 3 {
			fp.delPeerLocked(id)
		}
	}
}

// sort list, and update index to map
func (fp *filePeerPool) sort() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sortLocked()
}

func (fp *filePeerPool) sortLocked() {
	sort.Sort(fp.l)
	for i, c := range fp.l {
		fp.mi[c.id] = i
	}
}

// choose a fast fileConn, or create a new conn randomly
func (fp *filePeerPool) chooseSource(name filename) (*filePeer, *fileConn, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if peerM, ok := fp.mf[name]; ok && len(peerM) > 0 {
		totalConns := len(fp.l)

		var hasFreshPeer = false  // whether have peers those has no file connection
		var createNewConn = false // whether create new connection randomly if have no fast idle fileConns
		var freshPeerId peerId    // if should create new connection, then will use this filePeer
		for id := range peerM {
			// this peer has no connection
			if _, ok = fp.mi[id]; !ok {
				freshPeerId = id
				hasFreshPeer = true
				createNewConn = rand.Intn(10) > 5
				break
			}
		}

		// sort first
		fp.sortLocked()

		for i, conn := range fp.l {
			if conn.isBusy() {
				continue
			}

			// can`t download this file
			if _, ok = peerM[conn.id]; !ok {
				continue
			}

			// the fileConn is fast enough
			if i <= totalConns/3 {
				return nil, conn, nil
			}

			// the fileConn is not so fast
			if hasFreshPeer && createNewConn {
				return fp.mp[freshPeerId], nil, nil
			} else {
				return nil, conn, nil
			}
		}

		if hasFreshPeer {
			return fp.mp[freshPeerId], nil, nil
		} else {
			// all peers are busy, the file downloading should be wait
			return nil, nil, nil
		}
	}

	// no peers
	return nil, nil, errNoSuitablePeers
}

func (fp *filePeerPool) reset() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.mf = make(map[filename]map[peerId]struct{})
	fp.mp = make(map[peerId]*filePeer)
	fp.mi = make(map[peerId]int)

	for _, c := range fp.l {
		c.close()
	}

	fp.l = nil
}

type asyncFileTask struct {
	file File
	ch   chan error
	ctx  context.Context
}
type asyncFileTasks []asyncFileTask

func (l asyncFileTasks) Len() int {
	return len(l)
}

func (l asyncFileTasks) Less(i, j int) bool {
	return l[i].file.StartHeight < l[j].file.StartHeight
}

func (l asyncFileTasks) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type filename = string
type peerId = string

type fileClient struct {
	fqueue asyncFileTasks // wait to download

	parser fileParser

	rec blockReceiver

	peers *peerSet

	pool *filePeerPool

	mu      sync.Mutex
	dialing map[string]struct{}

	dialer *net2.Dialer

	term    chan struct{}
	wg      sync.WaitGroup
	running int32

	log log15.Logger
}

func newFileClient(parser fileParser, rec blockReceiver, peers *peerSet) *fileClient {
	return &fileClient{
		parser:  parser,
		peers:   peers,
		pool:    newPool(),
		dialer:  &net2.Dialer{Timeout: 5 * time.Second},
		dialing: make(map[string]struct{}),
		rec:     rec,
		log:     log15.New("module", "net/fileClient"),
	}
}

func (fc *fileClient) addFilePeer(files []filename, sender Peer) {
	fc.pool.addPeer(files, sender.FileAddress().String(), sender.ID())
}

func (fc *fileClient) download(ctx context.Context, file File) <-chan error {
	ch := make(chan error, 1)

	wait, err := fc.downloadFile(file)
	if wait {
		fc.wait(asyncFileTask{
			file: file,
			ch:   ch,
			ctx:  ctx,
		})
	} else {
		ch <- err
	}

	return ch
}

func (fc *fileClient) runTask(t asyncFileTask) {
	select {
	case <-t.ctx.Done():
		t.ch <- nil
		return
	default:

	}

	cont, err := fc.downloadFile(t.file)
	if cont {
		fc.wait(t)
	} else {
		t.ch <- err
	}
}

func (fc *fileClient) downloadFile(file File) (wait bool, err error) {
	var p *filePeer
	var c *fileConn
	if p, c, err = fc.pool.chooseSource(file.Filename); err != nil {
		fc.log.Error(fmt.Sprintf("no suitable peers to download file %s", file.Filename))
		// no peers
		return false, err
	} else if c != nil {
		if err = fc.doJob(c, file); err == nil {
			// downloaded
			return false, nil
		}
	} else if p != nil {
		if c, err = fc.createConn(p); err == nil {
			if err = fc.doJob(c, file); err == nil {
				// downloaded
				return false, nil
			}
		}
	}

	return true, err
}

func (fc *fileClient) wait(t asyncFileTask) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.fqueue = append(fc.fqueue, t)
	sort.Sort(fc.fqueue)
}

func (fc *fileClient) doJob(c *fileConn, file File) error {
	start := time.Now()
	if derr := c.download(file, fc.rec); derr != nil {
		if derr.Fatal() {
			fc.fatalPeer(c.id, derr)
		} else {
			fc.pool.catch(c.id)
		}

		fc.log.Error(fmt.Sprintf("download <file %s> from %s error: %v", file.Filename, c.RemoteAddr(), derr))

		return derr
	}

	fc.log.Info(fmt.Sprintf("download <file %s> from %s elapse %s", file.Filename, c.RemoteAddr(), time.Now().Sub(start)))

	return nil
}

func (fc *fileClient) fatalPeer(id peerId, err error) {
	fc.pool.delPeer(id)
	if p := fc.peers.Get(id); p != nil {
		p.Report(err)
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
	if atomic.CompareAndSwapInt32(&fc.running, 1, 0) {
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
}

func (fc *fileClient) loop() {
	defer fc.wg.Done()

	ticker := time.NewTicker(500 * time.Microsecond)
	defer ticker.Stop()

Loop:
	for {
		select {
		case <-fc.term:
			break Loop
		case <-ticker.C:
			var t asyncFileTask
			fc.mu.Lock()
			if len(fc.fqueue) > 0 {
				t = fc.fqueue[0]
				fc.fqueue = fc.fqueue[1:]
			}
			fc.mu.Unlock()

			if t.file != nil {
				go fc.runTask(t)
			}
		}
	}

	fc.pool.reset()
}

func (fc *fileClient) dialed(addr string) {
	fc.mu.Lock()
	delete(fc.dialing, addr)
	fc.mu.Unlock()
}

// tcp dial error
func (fc *fileClient) createConn(p *filePeer) (c *fileConn, err error) {
	addr := p.addr

	fc.mu.Lock()
	if _, ok := fc.dialing[addr]; ok {
		fc.mu.Unlock()
		return nil, errPeerDialing
	}
	fc.dialing[addr] = struct{}{}
	fc.mu.Unlock()

	tcp, err := fc.dialer.Dial("tcp", addr)
	fc.dialed(addr)

	if err != nil {
		fc.pool.catch(p.id)
		return nil, err
	}

	c = newFileConn(tcp, p.id, fc.parser, fc.log)

	err = fc.pool.addConn(c)
	if err != nil {
		// already exist a file connection
		c.close()
	}

	return
}

func (fc *fileClient) status() FileClientStatus {
	fc.mu.Lock()
	queue := make([]File, len(fc.fqueue))
	for i := 0; i < len(fc.fqueue); i++ {
		queue[i] = fc.fqueue[i].file
	}
	fc.mu.Unlock()

	return FileClientStatus{
		FPool: fc.pool.status(),
		Queue: queue,
	}
}

type FileClientStatus struct {
	FPool FilePoolStatus
	Queue []File
}
