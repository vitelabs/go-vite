package net

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	net2 "net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
)

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

type connections []syncConnection

func (fl connections) Len() int {
	return len(fl)
}

func (fl connections) Less(i, j int) bool {
	return fl[i].speed() > fl[j].speed()
}

func (fl connections) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

func (fl connections) del(i int) connections {
	total := len(fl)
	if i < total {
		copy(fl[i:], fl[i+1:])
		return fl[:total-1]
	}

	return fl
}

type FilePoolStatus struct {
	Connections []FileConnStatus
}

type downloadPeer interface {
	ID() peerId
	height() uint64
	fileAddress() string
}

type downloadPeerSet interface {
	pickDownloadPeers(height uint64) (m map[peerId]downloadPeer)
}

type connPoolImpl struct {
	mu    sync.Mutex
	peers downloadPeerSet
	mi    map[peerId]int // value is the index of `connPoolImpl.l`
	l     connections    // connections sort by speed, from fast to slow
}

func newPool(peers downloadPeerSet) *connPoolImpl {
	return &connPoolImpl{
		mi:    make(map[peerId]int),
		peers: peers,
	}
}

func (fp *connPoolImpl) status() FilePoolStatus {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	cs := make([]FileConnStatus, len(fp.l))

	for i := 0; i < len(fp.l); i++ {
		cs[i] = fp.l[i].status()
	}

	return FilePoolStatus{
		Connections: cs,
	}
}

// delete filePeer and connection
func (fp *connPoolImpl) delPeer(id peerId) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delPeerLocked(id)
}

func (fp *connPoolImpl) delPeerLocked(id peerId) {
	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

func (fp *connPoolImpl) addConn(c syncConnection) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, ok := fp.mi[c.ID()]; ok {
		return errFileConnExist
	}

	fp.l = append(fp.l, c)
	fp.mi[c.ID()] = len(fp.l) - 1
	return nil
}

func (fp *connPoolImpl) catch(c syncConnection) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delPeerLocked(c.ID())
}

// sort list, and update index to map
func (fp *connPoolImpl) sort() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sortLocked()
}

func (fp *connPoolImpl) sortLocked() {
	sort.Sort(fp.l)
	for i, c := range fp.l {
		fp.mi[c.ID()] = i
	}
}

// choose chain fast fileConn, or create chain new conn randomly
func (fp *connPoolImpl) chooseSource(from, to uint64) (downloadPeer, syncConnection, error) {
	peerMap := fp.peers.pickDownloadPeers(to)

	if len(peerMap) == 0 {
		return nil, nil, errNoSuitablePeer
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sortLocked()
	for _, c := range fp.l {
		delete(peerMap, c.ID())
	}

	var createNew bool
	if len(peerMap) > 0 {
		createNew = rand.Intn(10) > 5
	}
	for i, c := range fp.l {
		if c.isBusy() || c.height() < to {
			continue
		}

		if len(fp.l)+1 > 3*(i+1) {
			// very fast
			return nil, c, nil
		}

		// if createNew is true, there must has new available peers
		if createNew {
			for _, p := range peerMap {
				return p, nil, nil
			}
		}
	}

	return nil, nil, nil
}

func (fp *connPoolImpl) reset() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.mi = make(map[peerId]int)

	for _, c := range fp.l {
		_ = c.Close()
	}

	fp.l = nil
}

type downloadTask struct {
	from, to uint64
	ch       chan error
	ctx      context.Context
}
type downloadTasks []downloadTask

func (l downloadTasks) Len() int {
	return len(l)
}

func (l downloadTasks) Less(i, j int) bool {
	return l[i].from < l[j].from
}

func (l downloadTasks) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type connPool interface {
	addConn(c syncConnection) error
	catch(c syncConnection)
	chooseSource(from, to uint64) (downloadPeer, syncConnection, error)
	reset()
}

type downloader struct {
	mu      sync.Mutex
	queue   downloadTasks // wait to download
	pool    connPool
	factory syncConnInitiator
	dialing map[string]struct{}
	dialer  *net2.Dialer

	term    chan struct{}
	wg      sync.WaitGroup
	running int32

	log log15.Logger
}

func newDownloader(pool connPool, factory syncConnInitiator) *downloader {
	return &downloader{
		pool:    pool,
		factory: factory,
		dialing: make(map[string]struct{}),
		dialer:  &net2.Dialer{Timeout: 5 * time.Second},
		log:     log15.New("module", "net/fileClient"),
	}
}

func (fc *downloader) download(ctx context.Context, from, to uint64) <-chan error {
	ch := make(chan error, 1)

	wait, err := fc.downloadChunk(from, to)
	if wait {
		fc.wait(downloadTask{
			from: from,
			to:   to,
			ch:   ch,
			ctx:  ctx,
		})
	} else {
		ch <- err
	}

	return ch
}

func (fc *downloader) runTask(t downloadTask) {
	select {
	case <-t.ctx.Done():
		t.ch <- nil
		return
	default:

	}

	cont, err := fc.downloadChunk(t.from, t.to)
	if cont {
		fc.wait(t)
	} else {
		t.ch <- err
	}
}

func (fc *downloader) downloadChunk(from, to uint64) (wait bool, err error) {
	var p downloadPeer
	var c syncConnection
	if p, c, err = fc.pool.chooseSource(from, to); err != nil {
		// no peers
		return false, err
	} else if c != nil {
		if err = fc.doJob(c, from, to); err == nil {
			// downloaded
			return false, nil
		}
	} else if p != nil {
		if c, err = fc.createConn(p); err == nil {
			if err = fc.doJob(c, from, to); err == nil {
				// downloaded
				return false, nil
			}
		}
	}

	return true, err
}

func (fc *downloader) wait(t downloadTask) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.queue = append(fc.queue, t)
	sort.Sort(fc.queue)
}

func (fc *downloader) doJob(c syncConnection, from, to uint64) error {
	start := time.Now()

	fc.log.Info(fmt.Sprintf("begin download chunk %d-%d from %s", from, to, c.RemoteAddr()))

	if err := c.download(from, to); err != nil {
		fc.pool.catch(c)
		fc.log.Error(fmt.Sprintf("download chunk %d-%d from %s error: %v", from, to, c.RemoteAddr(), err))

		return err
	}

	fc.log.Info(fmt.Sprintf("download chunk %d-%d from %s elapse %s", from, to, c.RemoteAddr(), time.Now().Sub(start)))

	return nil
}

func (fc *downloader) start() {
	if atomic.CompareAndSwapInt32(&fc.running, 0, 1) {
		fc.term = make(chan struct{})

		fc.wg.Add(1)
		go fc.loop()
	}
}

func (fc *downloader) stop() {
	if atomic.CompareAndSwapInt32(&fc.running, 1, 0) {
		close(fc.term)
		fc.wg.Wait()
	}
}

func (fc *downloader) loop() {
	defer fc.wg.Done()

	ticker := time.NewTicker(500 * time.Microsecond)
	defer ticker.Stop()

Loop:
	for {
		select {
		case <-fc.term:
			break Loop
		case <-ticker.C:
			var t downloadTask
			fc.mu.Lock()
			if len(fc.queue) > 0 {
				t = fc.queue[0]
				fc.queue = fc.queue[1:]
			}
			fc.mu.Unlock()

			go fc.runTask(t)
		}
	}

	fc.pool.reset()
}

func (fc *downloader) createConn(p downloadPeer) (c syncConnection, err error) {
	addr := p.fileAddress()

	fc.mu.Lock()
	if _, ok := fc.dialing[addr]; ok {
		fc.mu.Unlock()
		return nil, errPeerDialing
	}
	fc.dialing[addr] = struct{}{}
	fc.mu.Unlock()

	tcp, err := fc.dialer.Dial("tcp", addr)

	fc.mu.Lock()
	delete(fc.dialing, addr)
	fc.mu.Unlock()

	if err != nil {
		return nil, err
	}

	c, err = fc.factory.initiate(tcp, p)

	err = fc.pool.addConn(c)

	return
}
