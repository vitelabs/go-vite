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

type fileConn struct {
	net2.Conn
	id     peerId
	busy   int32 // atomic
	t      int64 // timestamp
	speed  int64 // download speed
	parser fileParser
	closed int32
}

func (f *fileConn) isBusy() bool {
	return atomic.LoadInt32(&f.busy) == 1
}

func (f *fileConn) download(file File, rec blockReceiver) (outerr *downloadError) {
	atomic.StoreInt32(&f.busy, 1)

	defer f.idle()

	getFiles := &message.GetFiles{
		Names: []string{file.Filename},
	}

	msg, err := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)
	if err != nil {
		return &downloadError{
			code: downloadPackMsgErr,
			err:  err.Error(),
		}
	}

	f.Conn.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	if err = p2p.WriteMsg(f.Conn, msg); err != nil {
		return &downloadError{
			code: downloadSendErr,
			err:  err.Error(),
		}
	}

	var sCount, aCount uint64

	f.Conn.SetReadDeadline(time.Now().Add(fileTimeout))
	f.parser.BlockParser(f.Conn, file.BlockNumbers, func(block ledger.Block, err error) {
		// Fatal error, then close the connection
		if outerr != nil && outerr.Fatal() {
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
			err:  fmt.Sprintf("incomplete file %s %d/%d, %d/%d", file.Filename, sCount, sTotal, aCount, aTotal),
		}
	}

	return
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

type filePeerPool struct {
	mu sync.Mutex

	// for filePeer
	mf map[filename]map[peerId]struct{}
	mp map[peerId]*filePeer

	// for fileConn
	mi map[peerId]int // index
	l  fileConns      // sort by speed, from fast to slow
}

func newPool() *filePeerPool {
	return &filePeerPool{
		mf: make(map[filename]map[peerId]struct{}),
		mp: make(map[peerId]*filePeer),
		mi: make(map[peerId]int),
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

	for _, m := range fp.mf {
		delete(m, id)
	}

	delete(fp.mp, id)

	fp.delConn(id)
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

func (fp *filePeerPool) errConn(c *fileConn) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if p, ok := fp.mp[c.id]; ok {
		p.fail++

		// fail too many times, then delete the peer
		if p.fail > 3 {
			fp.delPeer(c.id)
		}
	}
}

func (fp *filePeerPool) delConn(id peerId) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

// sort list, and update index to map
func (fp *filePeerPool) sort() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	sort.Sort(fp.l)
	for i, c := range fp.l {
		fp.mi[c.id] = i
	}
}

// choose a fast file connection, or random create new conns
func (fp *filePeerPool) chooseSource(name filename) (*filePeer, *fileConn, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if peerM, ok := fp.mf[name]; ok && len(peerM) > 0 {
		totalConns := len(fp.l)

		var hasFreshPeer = false // have peers those has no file connection
		var randCreate = false   // should create new connection randomly
		var freshPeerId peerId   // if should create new connection, then will use this filePeer
		for id := range peerM {
			// this peer has no connection
			if _, ok = fp.mi[id]; !ok {
				freshPeerId = id
				hasFreshPeer = true
				randCreate = rand.Intn(10) > 5
				break
			}
		}

		for i, conn := range fp.l {
			if conn.isBusy() {
				continue
			}

			// can download file
			if _, ok = peerM[conn.id]; !ok {
				continue
			}

			if i <= totalConns/3 {
				return nil, conn, nil
			}

			if hasFreshPeer && randCreate {
				return fp.mp[freshPeerId], nil, nil
			}
		}

		if hasFreshPeer {
			return fp.mp[freshPeerId], nil, nil
		} else {
			return nil, nil, nil
		}
	}

	return nil, nil, errNoSuitablePeers
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

	term chan struct{}
	wg   sync.WaitGroup

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

	cont, err := fc.downloadFile(file)
	if cont {
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
		t.ch <- t.ctx.Err()
	default:

	}

	cont, err := fc.downloadFile(t.file)
	if cont {
		fc.wait(t)
	} else {
		t.ch <- err
	}
}

func (fc *fileClient) downloadFile(file File) (cont bool, err error) {
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
		}

		fc.log.Error(fmt.Sprintf("download file %s from %s error: %v", file.Filename, c.RemoteAddr(), derr))

		return derr
	}

	fc.log.Info(fmt.Sprintf("download file %s from %s elapse %s", file.Filename, c.RemoteAddr(), time.Now().Sub(start)))

	return nil
}

func (fc *fileClient) fatalPeer(id peerId, err error) {
	fc.pool.delPeer(id)
	if p := fc.peers.Get(id); p != nil {
		p.Report(err)
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
}

func (fc *fileClient) dialed(addr string) {
	fc.mu.Lock()
	delete(fc.dialing, addr)
	fc.mu.Unlock()
}

func (fc *fileClient) countFail(p *filePeer) {
	count := atomic.AddInt32(&p.fail, 1)
	if count >= 3 {
		fc.pool.delPeer(p.id)
	}
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
		fc.countFail(p)
		return nil, err
	}

	c = &fileConn{
		Conn:   tcp,
		id:     p.id,
		parser: fc.parser,
	}

	err = fc.pool.addConn(c)
	if err != nil {
		// already exist a file connection
		c.close()
	}

	return
}
