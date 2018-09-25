package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/list"
	"sync"
	"sync/atomic"
)

const filterCap = 100000

// @section Peer for protocol handle, not p2p Peer.
var errPeerTermed = errors.New("peer has been terminated")

type Peer struct {
	*p2p.Peer
	mrw                p2p.MsgReadWriter
	ID                string
	head              types.Hash
	height            uint64
	CmdSet           uint64
	msgId uint64	// atomic, auto_increment, use for mark unique message
	KnownBlocks       *cuckoofilter.CuckooFilter
	Log               log15.Logger
	term              chan struct{}
	reqs              *list.List
	reqErr            chan error // pending reqs got error (eg. timeout)
}

func newPeer(p *p2p.Peer, mrw p2p.MsgReadWriter, cmdSet uint64) *Peer {
	return &Peer{
		Peer:              p,
		mrw:                mrw,
		ID:                p.ID().Brief(),
		CmdSet:           cmdSet,
		KnownBlocks:       cuckoofilter.NewCuckooFilter(filterCap),
		Log:               log15.New("module", "net/peer"),
		term:              make(chan struct{}),
		reqs:              list.New(),
		reqErr:            make(chan error, 1),
	}
}

func (p *Peer) Handshake(our *HandShakeMsg) error {
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

	if their.GenesisBlock != our.GenesisBlock {
		return errors.New("different genesis block")
	}

	if their.NetID != our.NetID {
		return fmt.Errorf("different network, our %d, their %d\n", our.NetID, their.NetID)
	}

	p.SetHead(their.CurrentBlock, their.Height)

	return nil
}

func (p *Peer) ReadHandshake() (their *HandShakeMsg, err error) {
	msg, err := p.mrw.ReadMsg()

	if err != nil {
		return
	}

	if msg.Cmd != uint64(HandshakeCode) {
		err = fmt.Errorf("should be HandshakeCode %d, got %d\n", HandshakeCode, msg.Cmd)
		return
	}

	their = new(HandShakeMsg)

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

func (p *Peer) doRequest(r *request) error {
	select {
	case <-p.term:
		return errPeerTermed
	default:
		p.reqs.Append(r)
		return r.peer.Send(r.cmd, r.id, r.param)
	}
}

func (p *Peer) delRequest(r *request) {
	p.reqs.Traverse(func(prev, current *list.Element) {
		if current.Value == r {
			p.reqs.Remove(prev, current)
		}
	})
}

//func (p *Peer) receive(msg *p2p.Msg) {
	//for _, job := range p.jobs {
	//	go func(r *req) {
	//		if r.callback != nil {
	//			done, err := r.callback(cmd, data)
	//			r.errch <- err
	//			r.done <- done
	//
	//			if err != nil || done {
	//				p.delete(r)
	//			}
	//		}
	//	}(job)
	//}
//}

// response
func (p *Peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error) {
	err = p.Send(SnapshotBlocksCode, msgId, bs)

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

func (p *Peer) SendAccountBlocks(abs []*ledger.AccountBlock, msgId uint64) (err error) {
	err = p.Send(AccountBlocksCode, msgId, abs)

	if err != nil {
		return
	}

	for _, b := range abs {
		p.SeeBlock(b.Hash)
	}

	return nil
}

func (p *Peer) SendFileList(fs *FileList, msgId uint64) error {
	return p.Send(FileListCode, msgId, fs)
}
func (p *Peer) SendSubLedger(s *SubLedger, msgId uint64) error {
	return p.Send(SubLedgerCode, msgId, s)
}

// request
func (p *Peer) RequestSnapshotBlocks(s *Segment) error {
	atomic.AddUint64(&p.msgId, 1)
	return p.Send(GetSnapshotBlocksCode, p.msgId, s)
}

func (p *Peer) RequestSnapshotBlocksByHash(hashes []types.Hash) error {
	atomic.AddUint64(&p.msgId, 1)
	return p.Send(GetSnapshotBlocksByHashCode, p.msgId, hashes)
}

func (p *Peer) RequestAccountBlocks(as AccountSegment) error {
	atomic.AddUint64(&p.msgId, 1)
	return p.Send(GetAccountBlocksCode, p.msgId, as)
}

func (p *Peer) RequestAccountBlocksByHash(hashes []types.Hash) error {
	atomic.AddUint64(&p.msgId, 1)
	return p.Send(GetAccountBlocksByHashCode, p.msgId, hashes)
}

func (p *Peer) RequestSubLedger(s *Segment) error {
	atomic.AddUint64(&p.msgId, 1)
	return p.Send(GetSubLedgerCode, p.msgId, s)
}

func (p *Peer) Send(code cmd, msgId uint64, payload p2p.Serializable) error {
	data, err := payload.Serialize()
	if err != nil {
		return err
	}

	return p.mrw.WriteMsg(&p2p.Msg{
		CmdSetID:   p.CmdSet,
		Cmd:        uint64(code),
		Id:         msgId,
		Size:       uint64(len(data)),
		Payload:    data,
	})
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
var errSetHasClosed = errors.New("peer set has closed")
var errSetHasPeer = errors.New("peer is existed")

type peerSet struct {
	peers  map[string]*Peer
	rw     sync.RWMutex
	closed bool
}

func NewPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*Peer),
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

	if m.closed {
		return errSetHasClosed
	}
	if _, ok := m.peers[peer.ID]; ok {
		return errSetHasPeer
	}

	m.peers[peer.ID] = peer
	return nil
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

// pick peers whose height taller than the target height
func (m *peerSet) Pick(height uint64) (peers []*Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, p := range m.peers {
		if p.height > height {
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

func (m *peerSet) close() {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, peer := range m.peers {
		peer.Disconnect(p2p.DiscQuitting)
	}

	m.closed = true
}
