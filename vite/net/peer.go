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
	peers() map[vnode.NodeID]struct{}
	seeBlock(hash types.Hash)
	hasBlock(hash types.Hash) bool
	height() uint64
	head() types.Hash

	fileAddress() string

	catch(err error)

	send(c code, id p2p.MsgId, data p2p.Serializable) error
	sendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId p2p.MsgId) error
	sendAccountBlocks(bs []*ledger.AccountBlock, msgId p2p.MsgId) error
	sendNewSnapshotBlock(b *ledger.SnapshotBlock) error
	sendNewAccountBlock(b *ledger.AccountBlock) error
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

type peer struct {
	p2p.Peer

	peerMap sync.Map // [vnode.NodeID]struct{}

	knownBlocks blockFilter
	errChan     chan error
	once        sync.Once

	log log15.Logger
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
		p.peerMap = sync.Map{}
	}

	for _, c := range ps {
		if c.add {
			id, err = vnode.Bytes2NodeID(c.id)
			if err != nil {
				continue
			}
			p.peerMap.Store(id, struct{}{})
		} else {
			p.peerMap.Delete(id)
		}
	}
}

func (p *peer) peers() map[vnode.NodeID]struct{} {
	m := make(map[vnode.NodeID]struct{})

	p.peerMap.Range(func(key, value interface{}) bool {
		id := key.(vnode.NodeID)
		m[id] = struct{}{}
		return true
	})

	return m
}

func (p *peer) seeBlock(hash types.Hash) {
	p.knownBlocks.lookAndRecord(hash[:])
}

func (p *peer) hasBlock(hash types.Hash) bool {
	return p.knownBlocks.has(hash[:])
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
	p.seeBlock(b.Hash)

	ms := &message.NewSnapshotBlock{
		Block: b,
		TTL:   10,
	}

	return p.send(NewSnapshotBlockCode, 0, ms)
}

func (p *peer) sendNewAccountBlock(b *ledger.AccountBlock) (err error) {
	p.seeBlock(b.Hash)

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

// syncPeer choose the middle peer from all peers sorted by height from low to high, eg:
// [1, 2, 4, 4, 5, 5, 10] represent all peers,
// middle peer is `peerList[ len(peerList)/2 ]` at height 4.
// choose middle peer but not the highest peer is to defend fake height attack. because
// it`s more hard to fake.
func (m *peerSet) syncPeer() Peer {
	l := m.peers()
	if len(l) == 0 {
		return nil
	}

	sort.Sort(l)
	mid := len(l) / 2

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

// unknownBlock return unsorted peers never received the specific block.
func (m *peerSet) unknownBlock(hash types.Hash) (l []broadcastPeer) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	l = make([]broadcastPeer, len(m.m))

	i := 0
	for _, p := range m.m {
		if seen := p.hasBlock(hash); !seen {
			l[i] = p
			i++
		}
	}

	return l[:i]
}

// peers return all peers unsorted.
func (m *peerSet) peers() (l peers) {
	m.prw.RLock()
	defer m.prw.RUnlock()

	l = make(peers, len(m.m))

	i := 0
	for _, p := range m.m {
		l[i] = p
		i++
	}

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

	mp := make(map[peerId]struct{}, len(m.m))

	for id := range m.m {
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

// peers can be sort by height, from low to high
type peers []Peer

func (s peers) Len() int {
	return len(s)
}

func (s peers) Less(i, j int) bool {
	return s[i].height() < s[j].height()
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
