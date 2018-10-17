package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	net2 "net"
	"sort"
	"strconv"
	"sync"
)

const filterCap = 100000

// @section Peer for protocol handle, not p2p Peer.
//var errPeerTermed = errors.New("peer has been terminated")

type Peer struct {
	*p2p.Peer
	mrw         p2p.MsgReadWriter
	ID          string
	head        types.Hash // hash of the top snapshotblock in snapshotchain
	height      uint64     // height of the snapshotchain
	filePort    uint16     // fileServer port, for request file
	CmdSet      uint64     // which cmdSet it belongs
	KnownBlocks *cuckoofilter.CuckooFilter
	log         log15.Logger
	errch       chan error
}

func newPeer(p *p2p.Peer, mrw p2p.MsgReadWriter, cmdSet uint64) *Peer {
	return &Peer{
		Peer:        p,
		mrw:         mrw,
		ID:          p.ID().String(),
		CmdSet:      cmdSet,
		KnownBlocks: cuckoofilter.NewCuckooFilter(filterCap),
		log:         log15.New("module", "net/peer"),
		errch:       make(chan error, 1),
	}
}

func (p *Peer) FileAddress() *net2.TCPAddr {
	return &net2.TCPAddr{
		IP:   p.IP(),
		Port: int(p.filePort),
	}
}

func (p *Peer) Handshake(our *message.HandShake) error {
	errch := make(chan error, 1)
	common.Go(func() {
		errch <- p.Send(HandshakeCode, 0, our)
	})

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
	if p.filePort == 0 {
		p.filePort = DefaultPort
	}

	return nil
}

func (p *Peer) ReadHandshake() (their *message.HandShake, err error) {
	msg, err := p.mrw.ReadMsg()

	if err != nil {
		return
	}

	if msg.Cmd != uint32(HandshakeCode) {
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
	p.log.Info("update status", "ID", p.ID, "height", p.height, "head", p.head)
}

func (p *Peer) SeeBlock(hash types.Hash) {
	p.KnownBlocks.InsertUnique(hash[:])
}

// send

func (p *Peer) SendSubLedger(bs []*ledger.SnapshotBlock, abs []*ledger.AccountBlock, msgId uint64) (err error) {
	err = p.Send(SubLedgerCode, msgId, &message.SubLedger{
		SBlocks: bs,
		ABlocks: abs,
	})

	if err != nil {
		return
	}

	for _, block := range bs {
		p.SeeBlock(block.Hash)
	}

	for _, block := range abs {
		p.SeeBlock(block.Hash)
	}

	return
}

func (p *Peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error) {
	err = p.Send(SnapshotBlocksCode, msgId, &message.SnapshotBlocks{bs})

	if err != nil {
		return
	}

	for _, b := range bs {
		p.SeeBlock(b.Hash)
	}

	return
}

func (p *Peer) SendAccountBlocks(bs []*ledger.AccountBlock, msgId uint64) (err error) {
	err = p.Send(AccountBlocksCode, msgId, &message.AccountBlocks{bs})

	if err != nil {
		return
	}

	for _, b := range bs {
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

func (p *Peer) SendNewAccountBlock(b *ledger.AccountBlock) (err error) {
	err = p.Send(NewAccountBlockCode, 0, b)

	if err != nil {
		return
	}

	p.SeeBlock(b.Hash)

	return
}

func (p *Peer) Send(code cmd, msgId uint64, payload p2p.Serializable) error {
	if msg, err := p2p.PackMsg(p.CmdSet, uint32(code), msgId, payload); err != nil {
		p.log.Error(fmt.Sprintf("pack message %s to %s error: %v", code, p, err))
		return err
	} else if err := p.mrw.WriteMsg(msg); err != nil {
		p.log.Error(fmt.Sprintf("send message %s to %s error: %v", code, p, err))
		return err
	}

	return nil
}

type PeerInfo struct {
	ID     string `json:"id"`
	Addr   string `json:"addr"`
	Head   string `json:"head"`
	Height uint64 `json:"height"`
}

func (p *PeerInfo) String() string {
	return p.ID + "@" + p.Addr + "/" + strconv.FormatUint(p.Height, 10)
}

func (p *Peer) Info() *PeerInfo {
	return &PeerInfo{
		ID:     p.ID,
		Addr:   p.RemoteAddr().String(),
		Head:   p.head.String(),
		Height: p.height,
	}
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

func newPeerSet() *peerSet {
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
		if p.height >= height {
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
