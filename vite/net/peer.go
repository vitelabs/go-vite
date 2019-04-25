package net

import (
	"errors"
	"sort"
	"sync"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
)

var errPeerExisted = errors.New("peer has existed")
var errPeerNotExist = errors.New("peer not exist")

type peerId = vnode.NodeID

type peerConn struct {
	id  []byte
	add bool // add or remove
}

// Peer for protocol handle, not p2p Peer.
type Peer interface {
	p2p.Peer
	setHead(head types.Hash, height uint64)
	setPeers(ps []peerConn, patch bool)
	peers() map[peerId]struct{}
	seeBlock(hash types.Hash) bool
	height() uint64
	head() types.Hash
	fileAddress() string
	catch(err error)
	send(c code, id p2p.MsgId, data p2p.Serializable) error
	sendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId p2p.MsgId) error
	sendAccountBlocks(bs []*ledger.AccountBlock, msgId p2p.MsgId) error
	info() PeerInfo
}

// PeerState is p2p.Peer.State
type PeerState struct {
	// Head is hash of the highest snapshot block
	Head types.Hash
	// Height is the snapshot chain height
	Height uint64
	// FileAddress is the network address can be connected to download ledger file
	FileAddress string
}

// PeerInfo is for api
type PeerInfo struct {
	ID     string   `json:"id"`
	Addr   string   `json:"addr"`
	Head   string   `json:"head"`
	Height uint64   `json:"height"`
	Peers  []string `json:"peers"`
}

type peer struct {
	p2p.Peer

	m  map[peerId]struct{}
	m2 map[peerId]struct{} // MUST NOT write m2, only read, for cross peers

	knownBlocks blockFilter
	errChan     chan error
	once        sync.Once

	log log15.Logger
}

func (p *peer) info() PeerInfo {
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
		ID:     p.Peer.ID().String(),
		Addr:   p.Peer.Address().String(),
		Head:   p.head().String(),
		Height: p.height(),
		Peers:  ps,
	}
}

func (p *peer) head() types.Hash {
	st := p.State().(PeerState)
	return st.Head
}

func (p *peer) height() uint64 {
	st := p.State().(PeerState)
	return st.Height
}

func newPeer(p p2p.Peer, log log15.Logger) Peer {
	return &peer{
		Peer:        p,
		knownBlocks: newBlockFilter(filterCap),
		m:           make(map[peerId]struct{}),
		log:         log,
		errChan:     make(chan error, 1),
	}
}

func (p *peer) catch(err error) {
	p.once.Do(func() {
		p.errChan <- err
	})
}

func (p *peer) fileAddress() string {
	st := p.State().(PeerState)
	return st.FileAddress
}

func (p *peer) setHead(head types.Hash, height uint64) {
	st := p.State().(PeerState)
	st.Head = head
	st.Height = height

	p.SetState(st)
}

func (p *peer) setPeers(ps []peerConn, patch bool) {
	var id vnode.NodeID
	var err error

	if patch {
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

func (p *peer) peers() map[peerId]struct{} {
	return p.m2
}

func (p *peer) seeBlock(hash types.Hash) (existed bool) {
	return p.knownBlocks.lookAndRecord(hash[:])
}

func (p *peer) send(c code, id p2p.MsgId, data p2p.Serializable) error {
	buf, err := data.Serialize()
	if err != nil {
		return err
	}

	var msg = p2p.Msg{
		Code:    p2p.Code(c),
		Id:      id,
		Payload: buf,
	}

	return p.WriteMsg(msg)
}

func (p *peer) sendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId p2p.MsgId) (err error) {
	ms := &message.SnapshotBlocks{
		Blocks: bs,
	}

	return p.send(SnapshotBlocksCode, msgId, ms)
}

func (p *peer) sendAccountBlocks(bs []*ledger.AccountBlock, msgId p2p.MsgId) (err error) {
	ms := &message.AccountBlocks{
		Blocks: bs,
	}

	return p.send(AccountBlocksCode, msgId, ms)
}

func (p *peer) sendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error) {
	ms := &message.NewSnapshotBlock{
		Block: b,
		TTL:   10,
	}

	return p.send(NewSnapshotBlockCode, 0, ms)
}

func (p *peer) sendNewAccountBlock(b *ledger.AccountBlock) (err error) {
	ms := &message.NewAccountBlock{
		Block: b,
		TTL:   10,
	}

	return p.send(NewAccountBlockCode, 0, ms)
}

type peerEventCode byte

const (
	addPeer peerEventCode = iota + 1
	delPeer
)

type peerEvent struct {
	code  peerEventCode
	peer  Peer
	count int
}

type peerSet struct {
	m   map[peerId]Peer
	prw sync.RWMutex

	subs []chan<- peerEvent
}

// pickDownloadPeers implement downloadPeerSet
func (m *peerSet) pickDownloadPeers(height uint64) (m2 map[peerId]downloadPeer) {
	m2 = make(map[peerId]downloadPeer)

	m.prw.RLock()
	defer m.prw.RUnlock()

	for id, p := range m.m {
		if p.height() >= height {
			m2[id] = p
		}
	}

	return
}

func newPeerSet() *peerSet {
	return &peerSet{
		m: make(map[peerId]Peer),
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
		c <- e
	}
}

// bestPeer is the tallest peer
func (m *peerSet) bestPeer() (best Peer) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	var maxHeight uint64
	for _, p := range m.m {
		peerHeight := p.height()
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
func (m *peerSet) syncPeer() Peer {
	l := m.sortPeers()
	if len(l) == 0 {
		return nil
	}

	sort.Sort(l)
	mid := len(l) / 3

	return l[mid]
}

// add will return error if another peer with the same id has in the set
func (m *peerSet) add(peer Peer) error {
	m.prw.Lock()
	defer m.prw.Unlock()

	id := peer.ID()

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
func (m *peerSet) remove(id peerId) (p Peer, err error) {
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
func (m *peerSet) pick(height uint64) (l []Peer) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	for _, p := range m.m {
		if p.height() >= height {
			l = append(l, p)
		}
	}

	return
}

// peers return all peers sort from low to high
func (m *peerSet) sortPeers() (l peers) {
	m.prw.RLock()

	l = make(peers, len(m.m))
	i := 0
	for _, p := range m.m {
		l[i] = p
		i++
	}
	m.prw.RUnlock()

	sort.Sort(l)

	return
}

// broadcastPeers return all peers unsorted.
// for broadcaster, use interface `[]broadcastPeer` other than `[]Peer` let easier to mock.
func (m *peerSet) broadcastPeers() (l []broadcastPeer) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	l = make([]broadcastPeer, len(m.m))

	i := 0
	for _, p := range m.m {
		l[i] = p
		i++
	}

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
func (m *peerSet) get(id peerId) Peer {
	m.prw.RLock()
	defer m.prw.RUnlock()

	return m.m[id]
}

func (m *peerSet) count() int {
	m.prw.RLock()
	defer m.prw.RUnlock()

	return len(m.m)
}

func (m *peerSet) info() []PeerInfo {
	m.prw.RLock()
	defer m.prw.RUnlock()

	infos := make([]PeerInfo, len(m.m))

	var i int
	for _, p := range m.m {
		infos[i] = p.info()
		i++
	}

	return infos
}

// peers can be sort by height, from high to low
type peers []Peer

func (s peers) Len() int {
	return len(s)
}

func (s peers) Less(i, j int) bool {
	return s[i].height() > s[j].height()
}

func (s peers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s peers) delete(id peerId) peers {
	for i, p := range s {
		if p.ID() == id {
			last := len(s) - 1
			if i != last {
				copy(s[i:], s[i+1:])
			}
			return s[:last]
		}
	}

	return s
}
