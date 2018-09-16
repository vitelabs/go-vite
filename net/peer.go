package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"io/ioutil"
	"math/big"
	"sync"
)

type Set interface {
	Has(interface{}) bool
	Add(interface{})
	Del(interface{})
	Count() uint
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

const filterCap = 100000

// @section Peer for protocol handle, not p2p Peer.
type Peer struct {
	*p2p.Peer
	ts                p2p.MsgReadWriter
	ID                string
	head              types.Hash
	height            *big.Int
	Version           uint64
	Lock              sync.RWMutex
	sending           chan struct{} // use this channel to ensure that only one goroutine send msg simultaneously.
	KnownBlocks       Set
	Log               log15.Logger
	term              chan struct{}
	sendSnapshotBlock chan *ledger.SnapshotBlock  // sending new snapshotblock
	sendAccountBlocks chan []*ledger.AccountBlock // sending new accountblocks
	Speed             int                         // response performance
}

func newPeer(p *p2p.Peer, ts p2p.MsgReadWriter, version uint64) *Peer {
	return &Peer{
		Peer:              p,
		ts:                ts,
		ID:                p.ID().Brief(),
		Version:           version,
		height:            new(big.Int),
		sending:           make(chan struct{}, 1),
		KnownBlocks:       NewCuckooSet(filterCap),
		Log:               log15.New("module", "net/peer"),
		term:              make(chan struct{}),
		sendSnapshotBlock: make(chan *ledger.SnapshotBlock),
		sendAccountBlocks: make(chan []*ledger.AccountBlock),
		Speed:             0,
	}
}

func (p *Peer) Handshake(netId uint64, height *big.Int, current, genesis types.Hash) error {
	errch := make(chan error, 1)
	go func() {
		errch <- p.Send(HandshakeCode, &HandShakeMsg{
			Version:      p.Version,
			NetID:        netId,
			Height:       height,
			CurrentBlock: current,
			GenesisBlock: genesis,
		})
	}()

	their, err := p.ReadHandshake()
	if err != nil {
		return err
	}

	if err = <-errch; err != nil {
		return err
	}

	if their.Version != p.Version {
		return fmt.Errorf("different protocol version, our %d, their %d\n", p.Version, their.Version)
	}

	if their.GenesisBlock != genesis {
		return errors.New("different genesis block")
	}

	if their.NetID != netId {
		return fmt.Errorf("different network, our %d, their %d\n", netId, their.NetID)
	}

	p.SetHead(their.CurrentBlock, their.Height)

	return nil
}

func (p *Peer) ReadHandshake() (their *HandShakeMsg, err error) {
	msg, err := p.ts.ReadMsg()
	if err != nil {
		return
	}
	if msg.Cmd != uint64(HandshakeCode) {
		err = fmt.Errorf("should be HandshakeCode %d, got %d\n", HandshakeCode, msg.Cmd)
		return
	}
	their = new(HandShakeMsg)

	data, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return
	}

	err = their.Deserialize(data)

	return
}

func (p *Peer) SetHead(head types.Hash, height *big.Int) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	p.head = head
	p.height = height
	p.Log.Info("update status", "ID", p.ID, "height", p.height, "head", p.Head)
}

func (p *Peer) Head() *BlockID {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	return &BlockID{
		Hash:   p.head,
		Height: p.height,
	}
}

func (p *Peer) Broadcast() {
	for {
		select {
		case abs := <-p.sendAccountBlocks:
			p.SendAccountBlocks(abs)
		case b := <-p.sendSnapshotBlock:
			p.SendNewSnapshotBlock(b)
		case <-p.term:
			return
		}
	}
}

// response
func (p *Peer) SendSnapshotBlockHeaders(headers) error {
	// todo

	return p.Send(SnapshotBlockHeadersCode, headers)
}

func (p *Peer) SendSnapshotBlockBodies(bodies) error {
	return p.Send(SnapshotBlockBodiesCode, bodies)
}

func (p *Peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock) error {
	return p.Send(SnapshotBlocksCode, bs)
}

func (p *Peer) SendNewSnapshotBlock(b *ledger.SnapshotBlock) error {
	return p.Send(NewSnapshotBlockCode, b)
}

func (p *Peer) SendAccountBlocks(abs []*ledger.AccountBlock) error {
	return p.Send(AccountBlocksCode, abs)
}

func (p *Peer) SendSubLedger() error {
	return p.Send(SubLedgerCode, abs)
}

// request
func (p *Peer) RequestSnapshotHeaders(s *Segment) error {
	return p.Send(GetSnapshotBlockHeadersCode, s)
}

func (p *Peer) RequestSnapshotBodies(hashes []types.Hash) error {
	return p.Send(GetSnapshotBlockBodiesCode, hashes)
}

func (p *Peer) RequestSnapshotBlocks(s *Segment) error {
	return p.Send(GetSnapshotBlocksCode, s)
}

func (p *Peer) RequestSnapshotBlocksByHash(hashes []types.Hash) error {
	return p.Send(GetSnapshotBlocksByHashCode, hashes)
}

func (p *Peer) RequestAccountBlocks(as AccountSegment) error {
	return p.Send(GetAccountBlocksCode, as)
}

func (p *Peer) RequestAccountBlocksByHash(hashes []types.Hash) error {
	return p.Send(GetAccountBlocksByHashCode, hashes)
}

func (p *Peer) RequestSubLedger(s *Segment) error {
	return p.Send(GetSubLedgerCode, s)
}

func (p *Peer) Send(code Cmd, msg p2p.Serializable) error {
	return p2p.Send(p.ts, uint64(CmdSetID), uint64(code), msg)
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
	var peerHeight *big.Int
	for _, peer := range m.peers {
		peerHeight = peer.Head().Height
		if peerHeight != nil && peerHeight.Cmp(maxHeight) > 0 {
			maxHeight = peer.Head().Height
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

func (m *peerSet) Pick(height *big.Int) (peers []*Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, p := range m.peers {
		if p.Head().Height.Cmp(height) > 0 {
			peers = append(peers, p)
		}
	}
	return
}

func (m *peerSet) Info() (info []*PeerInfo) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, peer := range m.peers {
		info = append(info, peer.Info())
	}

	return
}
