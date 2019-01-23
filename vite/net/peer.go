package net

import (
	"fmt"
	net2 "net"
	"sort"
	"strconv"
	"sync"
	"errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

const filterCap = 10000

var errDiffGesis = errors.New("different genesis block")

// Peer for protocol handle, not p2p Peer.
type Peer interface {
	RemoteAddr() *net2.TCPAddr
	FileAddress() *net2.TCPAddr
	SetHead(head types.Hash, height uint64)
	SeeBlock(hash types.Hash)
	SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error)
	SendAccountBlocks(bs []*ledger.AccountBlock, msgId uint64) (err error)
	SendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error)
	SendNewAccountBlock(b *ledger.AccountBlock) (err error)
	Send(code ViteCmd, msgId uint64, payload p2p.Serializable) (err error)
	SendMsg(msg *p2p.Msg) (err error)
	Report(err error)
	ID() string
	Height() uint64
	Head() types.Hash
	Disconnect(reason p2p.DiscReason)
}

type peer struct {
	*p2p.Peer
	mrw      *p2p.ProtoFrame
	id       string
	head     types.Hash // hash of the top snapshotblock in snapshotchain
	height   uint64     // height of the snapshotchain
	filePort uint16     // fileServer port, for request file
	CmdSet   p2p.CmdSet // which cmdSet it belongs
	knownBlocks blockFilter
	errChan chan error
	once    sync.Once

	log log15.Logger
}

func (p *peer) Head() types.Hash {
	return p.head
}

func (p *peer) Height() uint64 {
	return p.height
}

func (p *peer) ID() string {
	return p.id
}

func newPeer(p *p2p.Peer, mrw *p2p.ProtoFrame, cmdSet p2p.CmdSet) *peer {
	return &peer{
		Peer:        p,
		mrw:         mrw,
		id:          p.ID().String(),
		CmdSet:      cmdSet,
		knownBlocks: newBlockFilter(filterCap),
		log:         log15.New("module", "net/peer"),
		errChan:     make(chan error, 1),
	}
}

func (p *peer) Report(err error) {
	p.once.Do(func() {
		p.errChan <- err
	})
}

func (p *peer) FileAddress() *net2.TCPAddr {
	return &net2.TCPAddr{
		IP:   p.IP(),
		Port: int(p.filePort),
	}
}

func (p *peer) Handshake(our *message.HandShake) error {
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

	if their.Genesis != our.Genesis {
		return errDiffGesis
	}

	p.SetHead(their.Current, their.Height)
	p.filePort = their.Port
	if p.filePort == 0 {
		p.filePort = DefaultPort
	}

	return nil
}

func (p *peer) ReadHandshake() (their *message.HandShake, err error) {
	msg, err := p.mrw.ReadMsg()

	if err != nil {
		return
	}

	if msg.Cmd != p2p.Cmd(HandshakeCode) {
		err = fmt.Errorf("should be HandshakeCode %d, got %d\n", HandshakeCode, msg.Cmd)
		return
	}

	their = new(message.HandShake)

	err = their.Deserialize(msg.Payload)

	return
}

func (p *peer) SetHead(head types.Hash, height uint64) {
	p.head = head
	p.height = height

	p.log.Debug(fmt.Sprintf("update peers %s to status %s/%d", p.RemoteAddr(), head, height))
}

func (p *peer) SeeBlock(hash types.Hash) {
	p.knownBlocks.lookAndRecord(hash[:])
}

// send

func (p *peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error) {
	return p.Send(SnapshotBlocksCode, msgId, &message.SnapshotBlocks{bs})
}

func (p *peer) SendAccountBlocks(bs []*ledger.AccountBlock, msgId uint64) (err error) {
	return p.Send(AccountBlocksCode, msgId, &message.AccountBlocks{bs})
}

func (p *peer) SendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error) {
	p.SeeBlock(b.Hash)
	return p.Send(NewSnapshotBlockCode, 0, b)
}

func (p *peer) SendNewAccountBlock(b *ledger.AccountBlock) (err error) {
	p.SeeBlock(b.Hash)
	return p.Send(NewAccountBlockCode, 0, b)
}

func (p *peer) Send(code ViteCmd, msgId uint64, payload p2p.Serializable) (err error) {
	var msg *p2p.Msg

	if msg, err = p2p.PackMsg(p.CmdSet, p2p.Cmd(code), msgId, payload); err != nil {
		p.log.Error(fmt.Sprintf("pack message %s to %s error: %v", code, p.RemoteAddr(), err))
		return err
	} else if err = p.mrw.WriteMsg(msg); err != nil {
		p.log.Error(fmt.Sprintf("send message %s to %s error: %v", code, p.RemoteAddr(), err))
		return err
	}

	p.log.Info(fmt.Sprintf("send message %s to %s", code, p.RemoteAddr()))

	return nil
}

func (p *peer) SendMsg(msg *p2p.Msg) (err error) {
	return p.mrw.WriteMsg(msg)
}

type PeerInfo struct {
	ID      string `json:"id"`
	Addr    string `json:"addr"`
	Head    string `json:"head"`
	Height  uint64 `json:"height"`
	Created string `json:"created"`
}

func (p *PeerInfo) String() string {
	return p.ID + "@" + p.Addr + "/" + strconv.FormatUint(p.Height, 10)
}

func (p *peer) Info() *PeerInfo {
	return &PeerInfo{
		ID:      p.id,
		Addr:    p.RemoteAddr().String(),
		Head:    p.head.String(),
		Height:  p.height,
		Created: p.Created.Format("2006-01-02 15:04:05"),
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
	peer  *peer
	count int
	err   error
}

type peerSet struct {
	peers map[string]*peer
	rw    sync.RWMutex
	subs  []chan<- *peerEvent
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

func (m *peerSet) Sub(c chan<- *peerEvent) {
	m.rw.Lock()
	defer m.rw.Unlock()

	m.subs = append(m.subs, c)
}

func (m *peerSet) UnSub(c chan<- *peerEvent) {
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
func (m *peerSet) BestPeer() (best Peer) {
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

func (m *peerSet) SyncPeer() Peer {
	s := m.Peers()
	if len(s) == 0 {
		return nil
	}

	sort.Sort(s)
	mid := len(s) / 2

	return s[mid]
}

func (m *peerSet) Add(peer *peer) error {
	m.rw.Lock()
	defer m.rw.Unlock()

	if _, ok := m.peers[peer.id]; ok {
		return errSetHasPeer
	}

	m.peers[peer.id] = peer
	m.Notify(&peerEvent{
		code:  addPeer,
		peer:  peer,
		count: len(m.peers),
	})
	return nil
}

func (m *peerSet) Del(peer *peer) {
	m.rw.Lock()
	defer m.rw.Unlock()

	delete(m.peers, peer.id)
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

// Pick peers whose height taller than the target height
// has sorted from low to high
func (m *peerSet) Pick(height uint64) (l []Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, p := range m.peers {
		if p.height >= height {
			l = append(l, p)
		}
	}

	return
}

func (m *peerSet) Info() (info []*PeerInfo) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	info = make([]*PeerInfo, len(m.peers))

	i := 0
	for _, peer := range m.peers {
		info[i] = peer.Info()
		i++
	}

	return
}

func (m *peerSet) UnknownBlock(hash types.Hash) (peers []Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	peers = make([]Peer, len(m.peers))

	i := 0
	for _, p := range m.peers {
		if seen := p.knownBlocks.has(hash[:]); !seen {
			peers[i] = p
			i++
		}
	}

	return peers[:i]
}

func (m *peerSet) Peers() (peers Peers) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	peers = make(Peers, len(m.peers))

	i := 0
	for _, peer := range m.peers {
		peers[i] = peer
		i++
	}

	return
}

func (m *peerSet) Has(id string) bool {
	m.rw.Lock()
	defer m.rw.Unlock()

	_, ok := m.peers[id]
	return ok
}

func (m *peerSet) Get(id string) Peer {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return m.peers[id]
}

// @implementation sort.Interface
type peers []Peer

func (s peers) Len() int {
	return len(s)
}

func (s peers) Less(i, j int) bool {
	return s[i].Height() < s[j].Height()
}

func (s peers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s peers) delete(id string) peers {
	for i, p := range s {
		if p.ID() == id {
			lastIndex := len(s) - 1
			if i != lastIndex {
				copy(s[i:], s[i+1:])
			}
			return s[:lastIndex]
		}
	}

	return s
}

type Peers []Peer

func (l Peers) Len() int {
	return len(l)
}

func (l Peers) Less(i, j int) bool {
	return l[i].Height() < l[j].Height()
}

func (l Peers) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
