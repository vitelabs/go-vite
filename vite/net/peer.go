package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/block"
	"io/ioutil"
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

const filterCap = 100000

// @section Peer for protocol handle, not p2p Peer.
var errPeerTermed = errors.New("peer has been terminated")

type Peer struct {
	*p2p.Peer
	ts                p2p.MsgReadWriter
	ID                string
	head              types.Hash
	height            uint64
	Version           uint64
	Lock              sync.RWMutex
	KnownBlocks       *block.CuckooSet
	Log               log15.Logger
	term              chan struct{}
	sendSnapshotBlock chan *ledger.SnapshotBlock  // sending new snapshotblock
	sendAccountBlocks chan []*ledger.AccountBlock // sending new accountblocks
	Speed             int                         // response performance
	reqs              *request
	reqErr            chan error // pending reqs got error (eg. timeout)
}

func newPeer(p *p2p.Peer, ts p2p.MsgReadWriter, version uint64) *Peer {
	return &Peer{
		Peer:              p,
		ts:                ts,
		ID:                p.ID().Brief(),
		Version:           version,
		KnownBlocks:       block.NewCuckooSet(filterCap),
		Log:               log15.New("module", "net/peer"),
		term:              make(chan struct{}),
		sendSnapshotBlock: make(chan *ledger.SnapshotBlock),
		sendAccountBlocks: make(chan []*ledger.AccountBlock),
		Speed:             0,
		reqs:              &request{},
		reqErr:            make(chan error, 1),
	}
}

func (p *Peer) delRequest(req *request) {

}

func (p *Peer) Handshake(netId uint64, height uint64, current, genesis types.Hash) error {
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

func (p *Peer) SetHead(head types.Hash, height uint64) {
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

//func (p *Peer) Broadcast() {
//	for {
//		select {
//		case abs := <-p.sendAccountBlocks:
//			p.SendAccountBlocks(abs)
//		case b := <-p.sendSnapshotBlock:
//			p.SendNewSnapshotBlock(b)
//		case <-p.term:
//			return
//		}
//	}
//}

func (p *Peer) SeeBlock(hash types.Hash) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.KnownBlocks.Add(hash)
}

func (p *Peer) execute(r *req) error {
	p.jobLock.Lock()
	defer p.jobLock.Unlock()

	select {
	case <-p.term:
		return errPeerTermed
	default:
		p.jobs = append(p.jobs, r)
		return r.peer.Send(r.param.Cmd, r.param.Params)
	}
}

func (p *Peer) delete(r *req) {
	p.jobLock.Lock()
	defer p.jobLock.Unlock()

	i := 0
	for j, req := range p.jobs {
		if !req.Equal(r) {
			p.jobs[i] = p.jobs[j]
			i++
		}
	}

	p.jobs = p.jobs[:i]
}

func (p *Peer) receive(cmd Cmd, data interface{}) {
	p.jobLock.RLock()
	defer p.jobLock.RUnlock()

	for _, job := range p.jobs {
		go func(r *req) {
			if r.callback != nil {
				done, err := r.callback(cmd, data)
				r.errch <- err
				r.done <- done

				if err != nil || done {
					p.delete(r)
				}
			}
		}(job)
	}
}

// response
func (p *Peer) SendSnapshotBlockHeaders(headers) error {
	return p.Send(SnapshotBlockHeadersCode, headers)
}

func (p *Peer) SendSnapshotBlockBodies(bodies) error {
	return p.Send(SnapshotBlockBodiesCode, bodies)
}

func (p *Peer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock) (err error) {
	err = p.Send(SnapshotBlocksCode, bs)

	if err != nil {
		return
	}

	for _, b := range bs {
		p.SeeBlock(b.Hash)
	}

	return
}

func (p *Peer) SendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error) {
	err = p.Send(NewSnapshotBlockCode, b)

	if err != nil {
		return
	}

	p.SeeBlock(b.Hash)

	return
}

func (p *Peer) SendAccountBlocks(abs []*ledger.AccountBlock) (err error) {
	err = p.Send(AccountBlocksCode, abs)

	if err != nil {
		return
	}

	for _, b := range abs {
		p.SeeBlock(b.Hash)
	}

	return nil
}

func (p *Peer) SendSubLedger() error {
	return p.Send(SubLedgerCode)
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

		p.jobLock.Lock()
		defer p.jobLock.Unlock()
		// put jobs back to reqPool
		for _, job := range p.jobs {
			job.errch <- errPeerTermed
		}

		p.jobs = nil
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

func (p *Peer) Receive(cmd Cmd, payload interface{}) {
	p.jobLock.RLock()
	defer p.jobLock.RUnlock()
	for _, job := range p.jobs {
		if job.callback != nil {
			done, err := job.callback(cmd, payload)
			job.done <- done
			job.errch <- err
		}
	}
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
		peerHeight := peer.Head().Height
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
		if p.Head().Height > height {
			peers = append(peers, p)
		}
	}

	return
}

func (m *peerSet) PickIdle(height uint64) (peers []*Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for _, p := range m.peers {
		if p.Head().Height > height && len(p.jobs) == 0 {
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
		if !peer.KnownBlocks.Has(hash) {
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
