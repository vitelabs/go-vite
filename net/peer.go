package net

import (
	"encoding/binary"
	"errors"
	"fmt"
	_net "net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/bloom"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net/netool"
	"github.com/vitelabs/go-vite/net/vnode"
)

var errPeerExisted = errors.New("peer has existed")
var errPeerNotExist = errors.New("peer not exist")
var errPeerAlreadyRunning = errors.New("peer is already running")
var errPeerNotRunning = errors.New("peer is not running")
var errPeerCannotWrite = errors.New("peer is not writable")

func extractAddress(sender *_net.TCPAddr, fileAddressBytes []byte, defaultPort int) (address string) {
	var fromIP = sender.IP

	if len(fileAddressBytes) == 2 {
		port := binary.BigEndian.Uint16(fileAddressBytes)
		return fromIP.String() + ":" + strconv.Itoa(int(port))
	}

	if len(fileAddressBytes) > 2 {
		var ep = new(vnode.EndPoint)

		if err := ep.Deserialize(fileAddressBytes); err == nil {
			if ep.Typ.Is(vnode.HostIP) {
				err = netool.CheckRelayIP(fromIP, ep.Host)
				if err != nil {
					ep.Host = fromIP
				}

				address = ep.String()
				return
			}

			epStr := ep.String()
			_, err = _net.ResolveTCPAddr("tcp", epStr)
			if err == nil {
				address = ep.String()
				return
			}
		}
	}

	return fromIP.String() + ":" + strconv.Itoa(defaultPort)
}

type peerId = vnode.NodeID

type peerConn struct {
	id  []byte
	add bool // add or remove
}

// PeerInfo is for api
type PeerInfo struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	Version    int64    `json:"version"`
	Height     uint64   `json:"height"`
	Address    string   `json:"address"`
	Flag       PeerFlag `json:"flag"`
	Superior   bool     `json:"superior"`
	Reliable   bool     `json:"reliable"`
	CreateAt   string   `json:"createAt"`
	ReadQueue  int      `json:"readQueue"`
	WriteQueue int      `json:"writeQueue"`
	Peers      []string `json:"peers"`
}

type PeerFlag byte

const (
	PeerFlagInbound  PeerFlag = 0
	PeerFlagOutbound PeerFlag = 1
	PeerFlagStatic   PeerFlag = 1 << 1
)

func (f PeerFlag) is(f2 PeerFlag) bool {
	return (f & f2) > 0
}

type PeerManager interface {
	UpdatePeer(p *Peer, newSuperior bool)
}

type Peer struct {
	codec Codec

	Id            peerId
	Name          string
	Height        uint64
	Head          types.Hash
	Version       int64
	publicAddress string
	fileAddress   string

	CreateAt int64

	Flag     PeerFlag
	Superior bool

	reliable int32 // whether the same chain

	busy  int32
	busyT int64

	running    int32
	writable   int32 // set to 0 when write error in writeLoop, or close actively
	writing    int32
	readQueue  chan Msg // will be closed when read error in readLoop
	writeQueue chan Msg // will be closed in method Close

	errChan chan error
	wg      sync.WaitGroup

	manager PeerManager
	handler msgHandler

	knownBlocks *bloom.Filter

	m  map[peerId]struct{}
	m2 map[peerId]struct{} // MUST NOT write m2, only read, for cross peers

	once sync.Once

	log log15.Logger
}

func (p *Peer) isReliable() bool {
	return atomic.LoadInt32(&p.reliable) == 1
}

func (p *Peer) setReliable(bool2 bool) {
	var v int32
	if bool2 {
		v = 1
	}
	atomic.StoreInt32(&p.reliable, v)
	if v > 0 {
		p.log.Info("set reliable true")
	} else {
		p.log.Info("set reliable false")
	}
}

// WriteMsg will put msg into queue, then write asynchronously
func (p *Peer) WriteMsg(msg Msg) (err error) {
	if !p.canWritable() {
		return errPeerCannotWrite
	}

	// 5s
	if atomic.LoadInt32(&p.busy) == 1 && time.Now().Unix()-p.busyT < 5 {
		return nil
	}

	p.write()
	defer p.writeDone()

	select {
	case p.writeQueue <- msg:
		atomic.StoreInt32(&p.busy, 0)
	default:
		atomic.StoreInt32(&p.busy, 1)
		p.busyT = time.Now().Unix()
	}

	return nil
}

func (p *Peer) write() {
	atomic.AddInt32(&p.writing, 1)
}

func (p *Peer) writeDone() {
	atomic.AddInt32(&p.writing, -1)
}

func (p *Peer) Info() PeerInfo {
	var ps []string

	if total := len(p.m2); total > 0 {
		ps = make([]string, total)
		var i int
		// m2 maybe replaced concurrently, more peers than total.
		for id := range p.m2 {
			ps[i] = id.String()
			i++
			if i == total {
				break
			}
		}
		ps = ps[:i]
	}

	return PeerInfo{
		Id:         p.Id.String(),
		Name:       p.Name,
		Version:    p.Version,
		Height:     p.Height,
		Address:    p.codec.Address().String(),
		Flag:       p.Flag,
		Superior:   p.Superior,
		Reliable:   atomic.LoadInt32(&p.reliable) == 1,
		CreateAt:   time.Unix(p.CreateAt, 0).Format("2006-01-02 15:04:05"),
		ReadQueue:  len(p.readQueue),
		WriteQueue: len(p.writeQueue),
		Peers:      ps,
	}
}

func newPeer(c Codec, their *HandshakeMsg, publicAddress, fileAddress string, superior bool, flag PeerFlag, manager PeerManager, handler msgHandler) *Peer {
	c.SetReadTimeout(readMsgTimeout)
	c.SetWriteTimeout(writeMsgTimeout)

	peer := &Peer{
		codec:         c,
		Id:            their.ID,
		Name:          their.Name,
		Height:        their.Height,
		Head:          their.Head,
		Version:       their.Version,
		publicAddress: publicAddress,
		fileAddress:   fileAddress,
		CreateAt:      their.Timestamp,
		Flag:          flag,
		Superior:      superior,
		reliable:      0,
		running:       0,
		writable:      1,
		writing:       0,
		readQueue:     make(chan Msg, 10),
		writeQueue:    make(chan Msg, 1000),
		errChan:       make(chan error, 4), // read write handle catch
		wg:            sync.WaitGroup{},
		manager:       manager,
		handler:       handler,
		knownBlocks:   bloom.New(filterCap, rt),
		m:             make(map[peerId]struct{}),
		m2:            nil,
		once:          sync.Once{},
	}
	peer.log = netLog.New("peer", peer.String())

	return peer
}

func (p *Peer) run() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.goLoop(p.readLoop, p.errChan)
		p.goLoop(p.writeLoop, p.errChan)
		p.goLoop(p.handleLoop, p.errChan)

		err = <-p.errChan
		return
	}

	return errPeerAlreadyRunning
}

func (p *Peer) goLoop(fn func() error, ch chan<- error) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := fn()
		ch <- err
	}()
}

func (p *Peer) readLoop() (err error) {
	defer close(p.readQueue)

	var msg Msg

	for {
		msg, err = p.codec.ReadMsg()
		if err != nil {
			p.stopWrite(fmt.Errorf("failed to read message: %v", err))
			return
		}

		msg.ReceivedAt = time.Now().Unix()
		msg.Sender = p

		switch msg.Code {
		case CodeDisconnect:
			if len(msg.Payload) > 0 {
				err = PeerError(msg.Payload[0])
			} else {
				err = PeerUnknownReason
			}
			return
		case CodeControlFlow:
		// todo

		default:
			p.readQueue <- msg
		}
	}
}

func (p *Peer) writeLoop() (err error) {
	var msg Msg
	for msg = range p.writeQueue {
		if err = p.codec.WriteMsg(msg); err != nil {
			p.stopWrite(fmt.Errorf("failed to write msg %d %d bytes: %v", msg.Code, len(msg.Payload), err))
			return
		}
	}

	return nil
}

func (p *Peer) handleLoop() (err error) {
	var msg Msg
	for msg = range p.readQueue {
		err = p.handler.handle(msg)
		if err != nil {
			p.log.Warn(fmt.Sprintf("failed to handle msg %d: %v", msg.Code, err))
			return
		}
	}

	return nil
}

func (p *Peer) canWritable() bool {
	return atomic.LoadInt32(&p.writable) == 1
}

func (p *Peer) stopWrite(err error) {
	atomic.StoreInt32(&p.writable, 0)
	p.log.Warn(fmt.Sprintf("stop write: %v", err))
}

func (p *Peer) Close(err error) (err2 error) {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		p.log.Warn(fmt.Sprintf("close: %v", err))

		if pe, ok := err.(PeerError); ok {
			_ = p.WriteMsg(Msg{
				Code:    CodeDisconnect,
				Payload: []byte{byte(pe)},
			})
		}

		time.Sleep(100 * time.Millisecond)
		p.stopWrite(fmt.Errorf("close peer: %v", err))

		// ensure nobody is writing
		for {
			if atomic.LoadInt32(&p.writing) == 0 {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		close(p.writeQueue)

		if err3 := p.codec.Close(); err3 != nil {
			err2 = err3
		}

		p.wg.Wait()
	}

	return errPeerNotRunning
}

func (p *Peer) Disconnect(err error) {
	p.log.Warn(fmt.Sprintf("disconnect peer: %v", err))
	_ = Disconnect(p.codec, err)
}

func (p *Peer) String() string {
	return p.Id.Brief() + "@" + p.codec.Address().String()
}

func (p *Peer) SetState(head types.Hash, height uint64) {
	p.Head, p.Height = head, height
}

func (p *Peer) SetSuperior(superior bool) error {
	if atomic.LoadInt32(&p.running) == 0 {
		return errPeerNotRunning
	}

	p.manager.UpdatePeer(p, superior)

	return nil
}

func (p *Peer) catch(err error) {
	p.once.Do(func() {
		p.errChan <- err
	})
}

func (p *Peer) setPeers(ps []peerConn, patch bool) {
	var id vnode.NodeID
	var err error

	if false == patch {
		p.m = make(map[peerId]struct{})
	}

	for _, c := range ps {
		if c.add {
			id, err = vnode.Bytes2NodeID(c.id)
			if err != nil {
				continue
			}
			p.m[id] = struct{}{}
		} else {
			delete(p.m, id)
		}
	}

	// make copy
	m2 := make(map[peerId]struct{}, len(p.m))
	for id = range p.m {
		m2[id] = struct{}{}
	}

	p.m2 = m2
}

func (p *Peer) peers() map[peerId]struct{} {
	return p.m2
}

func (p *Peer) send(c Code, id MsgId, data Serializable) error {
	buf, err := data.Serialize()
	if err != nil {
		return err
	}

	var msg = Msg{
		Code:    c,
		Id:      id,
		Payload: buf,
	}

	return p.WriteMsg(msg)
}

func (p *Peer) sendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId MsgId) (err error) {
	ms := &SnapshotBlocks{
		Blocks: bs,
	}

	return p.send(CodeSnapshotBlocks, msgId, ms)
}

func (p *Peer) sendAccountBlocks(bs []*ledger.AccountBlock, msgId MsgId) (err error) {
	ms := &AccountBlocks{
		Blocks: bs,
	}

	return p.send(CodeAccountBlocks, msgId, ms)
}

type peerEventCode byte

const (
	addPeer peerEventCode = iota + 1
	delPeer
)

type peerEvent struct {
	code  peerEventCode
	peer  *Peer
	count int
}

type peerSet struct {
	m   map[peerId]*Peer
	prw sync.RWMutex

	subs []chan<- peerEvent
}

func (m *peerSet) reliable() (l peers) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if atomic.LoadInt32(&p.reliable) == 1 {
			l = append(l, p)
		}
	}

	return
}

func (m *peerSet) UpdatePeer(p *Peer, newSuperior bool) {
	// todo
}

func (m *peerSet) has(id peerId) bool {
	m.prw.RLock()
	defer m.prw.RUnlock()

	_, ok := m.m[id]
	return ok
}

// pickDownloadPeers implement downloadPeerSet
func (m *peerSet) pickDownloadPeers(height uint64) (m2 map[peerId]*Peer) {
	m2 = make(map[peerId]*Peer)

	m.prw.RLock()
	defer m.prw.RUnlock()

	for id, p := range m.m {
		if p.Height >= height {
			m2[id] = p
		}
	}

	return
}

func newPeerSet() *peerSet {
	return &peerSet{
		m: make(map[peerId]*Peer),
	}
}

func (m *peerSet) sub(ch chan<- peerEvent) {
	m.subs = append(m.subs, ch)
}

func (m *peerSet) unSub(ch chan<- peerEvent) {
	for i, c := range m.subs {
		if c == ch {
			if i != len(m.subs)-1 {
				copy(m.subs[i:], m.subs[i+1:])
			}
			m.subs = m.subs[:len(m.subs)-1]
			return
		}
	}
}

func (m *peerSet) notify(e peerEvent) {
	for _, c := range m.subs {
		select {
		case c <- e:
		default:
		}
	}
}

// bestPeer is the tallest peer
func (m *peerSet) bestPeer() (best *Peer) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	var maxHeight uint64
	for _, p := range m.m {
		peerHeight := p.Height
		if peerHeight > maxHeight {
			maxHeight = peerHeight
			best = p
		}
	}

	return
}

// syncPeer choose the middle peer from all peers sorted by height from high to low, eg:
// peerList [10, 8, 7, 7, 7, 5, 4] represent all 7 peers,
// middle peer is `peerList[ len(peerList)/3 ]` at height 7.
// choose middle peer but not the highest peer is to defend fake height attack. because
// it`s more hard to fake.
func (m *peerSet) syncPeer() *Peer {
	l := m.sortPeers(true)
	if len(l) == 0 {
		return nil
	}

	sort.Sort(l)
	mid := len(l) / 3

	return l[mid]
}

// add will return error if another peer with the same id has in the set
func (m *peerSet) add(peer *Peer) error {
	m.prw.Lock()
	defer m.prw.Unlock()

	id := peer.Id

	if _, ok := m.m[id]; ok {
		return errPeerExisted
	}

	m.m[id] = peer

	go m.notify(peerEvent{
		code:  addPeer,
		peer:  peer,
		count: len(m.m),
	})
	return nil
}

// remove and return the specific peer, err is not nil if cannot find the peer
func (m *peerSet) remove(id peerId) (p *Peer, err error) {
	m.prw.Lock()

	p, ok := m.m[id]
	if ok {
		delete(m.m, id)
		var count = len(m.m)
		m.prw.Unlock()

		go m.notify(peerEvent{
			code:  delPeer,
			peer:  p,
			count: count,
		})
	} else {
		m.prw.Unlock()
		err = errPeerNotExist
	}

	return
}

// pick peers satisfy p.height() >= height, unsorted
func (m *peerSet) pick(height uint64) (ps peers) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if p.Height >= height {
			ps = append(ps, p)
		}
	}

	return
}

func (m *peerSet) pickReliable(height uint64) (ps peers) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if p.Height >= height && atomic.LoadInt32(&p.reliable) == 1 {
			ps = append(ps, p)
		}
	}

	return
}

// peers return all peers sort from low to high
func (m *peerSet) peers() (l peers) {
	m.prw.RLock()

	l = make(peers, len(m.m))
	i := 0
	for _, p := range m.m {
		l[i] = p
		i++
	}
	m.prw.RUnlock()

	return
}

// peers return all peers sort from low to high
func (m *peerSet) sortPeers(reliable bool) (l peers) {
	m.prw.RLock()

	l = make(peers, len(m.m))
	i := 0
	for _, p := range m.m {
		if reliable {
			if atomic.LoadInt32(&p.reliable) == 1 {
				l[i] = p
				i++
			}
		} else {
			l[i] = p
			i++
		}
	}

	m.prw.RUnlock()

	l = l[:i]
	sort.Sort(l)

	return
}

// idMap generate a map of peers ID, to heartbeat
func (m *peerSet) idMap() map[peerId]struct{} {
	m.prw.RLock()
	defer m.prw.RUnlock()

	var count = len(m.m)
	if count > maxNeighbors {
		count = maxNeighbors
	}

	mp := make(map[peerId]struct{}, count)

	var i int
	for id := range m.m {
		i++
		if i > count {
			break
		}
		mp[id] = struct{}{}
	}

	return mp
}

// get the specific peer
func (m *peerSet) get(id peerId) *Peer {
	m.prw.RLock()
	defer m.prw.RUnlock()

	return m.m[id]
}

func (m *peerSet) count() int {
	m.prw.RLock()
	defer m.prw.RUnlock()

	return len(m.m)
}

func (m *peerSet) countWithoutSBP() (n int) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if p.Superior {
			continue
		}
		n++
	}

	return
}

func (m *peerSet) inboundWithoutSBP() (n int) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if p.Superior {
			continue
		}
		if p.Flag.is(PeerFlagInbound) {
			n++
		}
	}

	return
}

func (m *peerSet) info() []PeerInfo {
	m.prw.RLock()
	defer m.prw.RUnlock()

	infos := make([]PeerInfo, len(m.m))

	var i int
	for _, p := range m.m {
		infos[i] = p.Info()
		i++
	}

	return infos
}

// peers can be sort by height, from high to low
type peers []*Peer

func (s peers) Len() int {
	return len(s)
}

func (s peers) Less(i, j int) bool {
	return s[i].Height > s[j].Height
}

func (s peers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s peers) delete(id peerId) peers {
	for i, p := range s {
		if p.Id == id {
			last := len(s) - 1
			if i != last {
				copy(s[i:], s[i+1:])
			}
			return s[:last]
		}
	}

	return s
}
