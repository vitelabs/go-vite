package syncer

import (
	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type BlockHash struct {
	Height int
	Hash   string
}

type Syncer interface {
	Fetcher() Fetcher
	Sender() Sender
	Handlers() Handlers
	DefaultHandler() MsgHandler
	Init(reader face.ChainReader, writer face.PoolWriter)
	Start()
	Stop()
	Done() bool
}
type chainRw struct {
	face.ChainReader
	face.PoolWriter
}

//type snapshotChainReader interface {
//	getBlocksByHeightHash(hashH common.HashHeight) *common.SnapshotBlock
//}
//type accountChainReader interface {
//	getBlocksByHeightHash(address string, hashH common.HashHeight) *common.AccountStateBlock
//}

type Fetcher interface {
	FetchAccount(address common.Address, hash common.HashHeight, prevCnt uint64)
	FetchSnapshot(hash common.HashHeight, prevCnt uint64)
	Fetch(request face.FetchRequest)
}

type Sender interface {
	// when new block create
	BroadcastAccountBlocks(common.Address, []*common.AccountStateBlock) error
	BroadcastSnapshotBlocks([]*common.SnapshotBlock) error

	// when fetch block message be arrived
	SendAccountBlocks(common.Address, []*common.AccountStateBlock, p2p.Peer) error
	SendSnapshotBlocks([]*common.SnapshotBlock, p2p.Peer) error

	SendAccountHashes(common.Address, []common.HashHeight, p2p.Peer) error
	SendSnapshotHashes([]common.HashHeight, p2p.Peer) error

	RequestAccountHash(common.Address, common.HashHeight, uint64) error
	RequestSnapshotHash(common.HashHeight, uint64) error
	RequestAccountBlocks(common.Address, []common.HashHeight) error
	RequestSnapshotBlocks([]common.HashHeight) error
}
type MsgHandler interface {
	Handle(common.NetMsgType, []byte, p2p.Peer)
	Types() []common.NetMsgType
	Id() string
}

type Handlers interface {
	RegisterHandler(MsgHandler)
	UnRegisterHandler(MsgHandler)
}

type syncer struct {
	sender   *sender
	fetcher  *fetcher
	receiver *receiver
	p2p      p2p.P2P
	state    *state

	bus EventBus.Bus
}

func (s *syncer) DefaultHandler() MsgHandler {
	return s.receiver
}

func NewSyncer(net p2p.P2P, bus EventBus.Bus) Syncer {
	self := &syncer{bus: bus}
	self.sender = &sender{net: net}
	self.p2p = net
	self.fetcher = &fetcher{sender: self.sender, retryPolicy: &defaultRetryPolicy{fetchedHashs: make(map[string]*RetryStatus)}, addressRetry: &addressRetryPolicy{}}
	return self
}
func (s *syncer) Init(reader face.ChainReader, writer face.PoolWriter) {
	rw := &chainRw{ChainReader: reader, PoolWriter: writer}
	s.state = newState(rw, s.fetcher, s.sender, s.p2p, s.bus)
	s.receiver = newReceiver(s.fetcher, rw, s.sender, s.state)
	s.p2p.SetHandlerFn(s.DefaultHandler().Handle)
	s.p2p.SetHandShaker(s.state)
}

func (s *syncer) Start() {
	s.state.start()
}
func (s *syncer) Stop() {
	s.state.stop()
}

func (s *syncer) Fetcher() Fetcher {
	return s.fetcher
}

func (s *syncer) Sender() Sender {
	return s.sender
}

func (s *syncer) Handlers() Handlers {
	return s.receiver
}

func (s *syncer) Done() bool {
	return s.state.syncDone()
}
