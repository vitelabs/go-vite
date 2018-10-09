package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"net"
	"sort"
	"sync"
)

const filterCap = 100000

// @section Peer for protocol handle, not p2p Peer.
var errPeerTermed = errors.New("peer has been terminated")

type Peer struct {
	*p2p.Peer
	mrw         p2p.MsgReadWriter
	ID          string
	head        types.Hash // hash of the top snapshotblock in snapshotchain
	height      uint64     // height of the snapshotchain
	filePort    uint16     // fileServer port, for request file
	CmdSet      uint64     // which cmdSet it belongs
	KnownBlocks *cuckoofilter.CuckooFilter
	Log         log15.Logger
	term        chan struct{}
	errch       chan error
}

func newPeer(p *p2p.Peer, mrw p2p.MsgReadWriter, cmdSet uint64) *Peer {
	return &Peer{
		Peer:        p,
		mrw:         mrw,
		ID:          p.ID().Brief(),
		CmdSet:      cmdSet,
		KnownBlocks: cuckoofilter.NewCuckooFilter(filterCap),
		Log:         log15.New("module", "net/peer"),
		term:        make(chan struct{}),
		errch:       make(chan error, 1),
	}
}

func (p *Peer) FileAddress() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   p.IP(),
		Port: int(p.filePort),
	}
}

func (p *Peer) Handshake(our *message.HandShake) error {
	errch := make(chan error, 1)
	go func() {
		errch <- p.Send(HandshakeCode, 0, our)
	}()

	their, err := p.ReadHandshake()
	if err != nil {
		return err
	}

	if err = <-errch; err != nil {
		return err
	}

	if their.CmdSet != p.CmdSet {
		return fmt.Errorf("different protocol, our %d, their %d\n", p.CmdSet, their.CmdSet)
	}

	if their.Genesis != our.Genesis {
		return errors.New("different genesis block")
	}

	p.SetHead(their.Current, their.Height)
	p.filePort = their.Port

	return nil
}

func (p *Peer) ReadHandshake() (their *message.HandShake, err error) {
	msg, err := p.mrw.ReadMsg()

	if err != nil {
		return
	}

	if msg.Cmd != uint64(HandshakeCode) {
		err = fmt.Errorf("should be HandshakeCode %d, got %d\n", HandshakeCode, msg.Cmd)
		return
	}

	their = new(message.HandShake)

	err = their.Deserialize(msg.Payload)

	return
}

func (p *Peer) SetHead(head types.Hash, height uint64) {
	p.head = head
	p.height = height
	p.Log.Info("update status", "ID", p.ID, "height", p.height, "head", p.head)
}

func (p *Peer) SeeBlock(hash types.Hash) {
	p.KnownBlocks.InsertUnique(hash[:])
}

// response
func (p *Peer) SendSubLedger(s *message.SubLedger, msgId uint64) (err error) {
	err = p.Send(SubLedgerCode, msgId, s)

	if err != nil {
		return
	}

	for _, blocks := range s.ABlocks {
		for _, block := range blocks {
			p.SeeBlock(block.Hash)
		}
	}

	for _, b := range s.SBlocks {
		p.SeeBlock(b.Hash)
	}

	return
}

func (p *Peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error) {
	err = p.Send(SnapshotBlocksCode, msgId, &message.SnapshotBlocks{
		Blocks: bs,
	})

	if err != nil {
		return
	}

	for _, b := range bs {
		p.SeeBlock(b.Hash)
	}

	return
}

func (p *Peer) SendAccountBlocks(s *message.AccountBlocks, msgId uint64) (err error) {
	err = p.Send(AccountBlocksCode, msgId, s)

	if err != nil {
		return
	}

	for _, b := range s.Blocks {
		p.SeeBlock(b.Hash)
	}

	return
}

func (p *Peer) SendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error) {
	err = p.Send(NewSnapshotBlockCode, 0, b)

	if err != nil {
		return
	}

	p.SeeBlock(b.Hash)

	return
}

func (p *Peer) Send(code cmd, msgId uint64, payload p2p.Serializable) error {
	data, err := payload.Serialize()
	if err != nil {
		return err
	}

	return p.mrw.WriteMsg(&p2p.Msg{
		CmdSetID: p.CmdSet,
		Cmd:      uint64(code),
		Id:       msgId,
		Size:     uint64(len(data)),
		Payload:  data,
	})
}

type PeerInfo struct {
	Addr   string
	Flag   int
	Head   string
	Height uint64
}

func (p *Peer) Info() *PeerInfo {
	// todo
	return &PeerInfo{}
}

// @section PeerSet
var errSetHasPeer = errors.New("peer is existed")

type peerEventCode byte

const (
	addPeer peerEventCode = iota + 1
	delPeer
)

type peerEvent struct {
	code  peerEventCode
	peer  *Peer
	count int
	err   error
}

type peerSet struct {
	peers map[string]*Peer
	rw    sync.RWMutex
	subs  []chan<- *peerEvent
}

func NewPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*Peer),
	}
}

func (m *peerSet) Sub(c chan<- *peerEvent) {
	m.rw.Lock()
	defer m.rw.Unlock()

	m.subs = append(m.subs, c)
}

func (m *peerSet) Unsub(c chan<- *peerEvent) {
	m.rw.Lock()
	defer m.rw.Unlock()

	var i, j int
	for i, j = 0, 0; i < len(m.subs); i++ {
		if m.subs[i] != c {
			m.subs[j] = m.subs[i]
			j++
		}
	}
	m.subs = m.subs[:j]
}

func (m *peerSet) Notify(e *peerEvent) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, c := range m.subs {
		select {
		case c <- e:
		default:
		}
	}
}

// the tallest peer
func (m *peerSet) BestPeer() (best *Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	var maxHeight uint64
	for _, peer := range m.peers {
		peerHeight := peer.height
		if peerHeight > maxHeight {
			maxHeight = peerHeight
			best = peer
		}
	}

	return
}

func (m *peerSet) Has(id string) bool {
	_, ok := m.peers[id]
	return ok
}

func (m *peerSet) Add(peer *Peer) error {
	m.rw.Lock()
	defer m.rw.Unlock()

	if _, ok := m.peers[peer.ID]; ok {
		return errSetHasPeer
	}

	m.peers[peer.ID] = peer
	m.Notify(&peerEvent{
		code:  addPeer,
		peer:  peer,
		count: len(m.peers),
	})
	return nil
}

func (m *peerSet) Del(peer *Peer) {
	m.rw.Lock()
	defer m.rw.Unlock()

	delete(m.peers, peer.ID)
	m.Notify(&peerEvent{
		code:  delPeer,
		peer:  peer,
		count: len(m.peers),
	})
}

func (m *peerSet) Count() int {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return len(m.peers)
}

// pick peers whose height taller than the target height
// has sorted from low to high
func (m *peerSet) Pick(height uint64) (peers []*Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, p := range m.peers {
		if p.height > height {
			peers = append(peers, p)
		}
	}

	sort.Sort(Peers(peers))

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

func (m *peerSet) UnknownBlock(hash types.Hash) (peers []*Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, peer := range m.peers {
		if !peer.KnownBlocks.Lookup(hash[:]) {
			peers = append(peers, peer)
		}
	}

	return
}

// @implementation sort.Interface
type Peers []*Peer

func (s Peers) Len() int {
	return len(s)
}

func (s Peers) Less(i, j int) bool {
	return s[i].height < s[j].height
}

func (s Peers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
