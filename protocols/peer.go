package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"sync"
)

type Set interface {
	Has(interface{}) bool
	Add(interface{})
	Del(interface{})
	Count() int
}

// used for mark request type
type reqFlag int

const (
	snapshotHeadersFlag reqFlag = iota
	accountblocksFlag
	snapshotBlocksFlag
	blocksFlag
)

type reqInfo struct {
	id   uint64
	flag reqFlag
	size uint64
}

func (r *reqInfo) Replay() {

}

// @section Peer for protocol handle, not p2p Peer.
type Peer struct {
	ts      Transport
	ID      string
	Head    types.Hash
	Height  *big.Int
	Version int
	Lock    sync.RWMutex

	// use this channel to ensure that only one goroutine send msg simultaneously.
	sending chan struct{}

	KnownSnapshotBlocks Set
	KnownAccountBlocks  Set

	Log log15.Logger

	term chan struct{}

	// wait for sending
	sendSnapshotBlock chan *ledger.SnapshotBlock
	sendAccountBlock  chan *ledger.AccountBlock

	outstandingReqs []*reqInfo

	// response performance
	Speed int
}

func newPeer() *Peer {
	return &Peer{
		sending: make(chan struct{}, 1),
	}
}

func (p *Peer) HandShake() error {
	return nil
}

func (p *Peer) Update(head types.Hash, height *big.Int) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	p.Head = head
	p.Height = height
	p.Log.Info("update status", "ID", p.ID, "height", p.Height, "head", p.Head)
}

func (p *Peer) Altitude() (head types.Hash, height *big.Int) {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	return p.Head, p.Height
}

func (p *Peer) Broadcast() {

}

// response
func (p *Peer) SendSnapshotBlockHeaders() {

}

func (p *Peer) SendSnapshotBlockBodies() {

}

func (p *Peer) SendSnapshotBlocks() {

}

func (p *Peer) SendAccountBlocks() {

}

func (p *Peer) SendSubLedger() {

}

// request
func (p *Peer) RequestSnapshotHeaders() {

}

func (p *Peer) RequestSnapshotBodies() {

}

func (p *Peer) RequestSnapshotBlocks() {

}

func (p *Peer) RequestAccountBlocks() {

}

func (p *Peer) RequestSubLedger() {

}

func (p *Peer) Destroy() {
	select {
	case <-p.term:
	default:
		close(p.term)
	}
}

type PeerInfo struct {
	Addr   string
	Flag   int
	Head   string
	Height uint64
}

func (p *Peer) Info() *PeerInfo {
	return &PeerInfo{}
}

// @section PeerSet
type peerSet struct {
	peers map[string]*Peer
	rw    sync.RWMutex
}

func NewPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*Peer),
	}
}

func (m *peerSet) BestPeer() (best *Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	maxHeight := new(big.Int)
	for _, peer := range m.peers {
		cmp := peer.Height.Cmp(maxHeight)
		if cmp > 0 {
			maxHeight = peer.Height
			best = peer
		}
	}

	return
}

func (m *peerSet) Has(id string) bool {
	_, ok := m.peers[id]
	return ok
}

func (m *peerSet) Add(peer *Peer) {
	m.rw.Lock()
	m.peers[peer.ID] = peer
	m.rw.Unlock()
}

func (m *peerSet) Del(peer *Peer) {
	m.rw.Lock()
	delete(m.peers, peer.ID)
	m.rw.Unlock()
}

func (m *peerSet) Count() int {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return len(m.peers)
}
