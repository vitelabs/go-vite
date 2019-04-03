package net

import (
	"context"
	"errors"
	"fmt"
	net2 "net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
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

type fileConns []*syncConn

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

	peers *peerSet

	mi map[vnode.NodeID]int // manage fileConns, corresponding value is the index of `filePeerPool.l`
	l  fileConns            // fileConns sort by speed, from fast to slow
}

func newPool() *filePeerPool {
	return &filePeerPool{
		mi: make(map[vnode.NodeID]int),
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

// delete filePeer and connection
func (fp *filePeerPool) delPeer(id vnode.NodeID) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delPeerLocked(id)
}

func (fp *filePeerPool) delPeerLocked(id vnode.NodeID) {
	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

func (fp *filePeerPool) addConn(c *syncConn) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, ok := fp.mi[c.id]; ok {
		return errFileConnExist
	}

	fp.l = append(fp.l, c)
	fp.mi[c.id] = len(fp.l) - 1
	return nil
}

func (fp *filePeerPool) catch(id vnode.NodeID) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delPeerLocked(id)
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

// choose chain fast fileConn, or create chain new conn randomly
func (fp *filePeerPool) chooseSource(from, to uint64) (*filePeer, *syncConn, error) {
	peers := fp.peers.pick(to)

	if len(peers) == 0 {
		return nil, nil, errors.New("no suitable peers")
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	var id vnode.NodeID
	var c *syncConn
	for _, p := range peers {
		id = p.ID()
		if index, ok := fp.mi[id]; ok {
			c = fp.l[index]
			if c.isBusy() {
				continue
			}
			return nil, c, nil
		}
	}

	return nil, nil, nil
}

func (fp *filePeerPool) reset() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.mi = make(map[vnode.NodeID]int)

	for _, c := range fp.l {
		_ = c.close()
	}

	fp.l = nil
}

type asyncFileTask struct {
	from, to uint64
	ch       chan error
	ctx      context.Context
}
type asyncFileTasks []asyncFileTask

func (l asyncFileTasks) Len() int {
	return len(l)
}

func (l asyncFileTasks) Less(i, j int) bool {
	return l[i].from < l[j].from
}

func (l asyncFileTasks) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type peerId = vnode.NodeID

type fileClient struct {
	fqueue asyncFileTasks // wait to download

	cacher syncCacher

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

func newFileClient(cacher syncCacher, rec blockReceiver, peers *peerSet) *fileClient {
	return &fileClient{
		cacher:  cacher,
		peers:   peers,
		pool:    newPool(),
		dialer:  &net2.Dialer{Timeout: 5 * time.Second},
		dialing: make(map[string]struct{}),
		rec:     rec,
		log:     log15.New("module", "net/fileClient"),
	}
}

func (fc *fileClient) download(ctx context.Context, from, to uint64) <-chan error {
	ch := make(chan error, 1)

	wait, err := fc.downloadChunk(from, to)
	if wait {
		fc.wait(asyncFileTask{
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

func (fc *fileClient) runTask(t asyncFileTask) {
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

func (fc *fileClient) downloadChunk(from, to uint64) (wait bool, err error) {
	var p *filePeer
	var c *syncConn
	if p, c, err = fc.pool.chooseSource(from, to); err != nil {
		fc.log.Error(fmt.Sprintf("no suitable peers to download chunk %d-%d", from, to))
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

func (fc *fileClient) wait(t asyncFileTask) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.fqueue = append(fc.fqueue, t)
	sort.Sort(fc.fqueue)
}

func (fc *fileClient) doJob(c *syncConn, from, to uint64) error {
	start := time.Now()

	if err := c.download(from, to); err != nil {
		fc.pool.catch(c.id)
		fc.log.Error(fmt.Sprintf("download chunk %d-%d from %s error: %v", from, to, c.RemoteAddr(), err))

		return err
	}

	fc.log.Info(fmt.Sprintf("download chunk %d-%d from %s elapse %s", from, to, c.RemoteAddr(), time.Now().Sub(start)))

	return nil
}

func (fc *fileClient) fatalPeer(id vnode.NodeID, err error) {
	fc.pool.delPeer(id)
	//if p := fc.peers.get(id); p != nil {
	//	p.catch(err)
	//}
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

			go fc.runTask(t)
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
func (fc *fileClient) createConn(p *filePeer) (c *syncConn, err error) {
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

	c = newFileConn(tcp, p.id, fc.cacher, fc.log)

	err = fc.pool.addConn(c)
	if err != nil {
		// already exist chain file connection
		_ = c.close()
	}

	return
}
