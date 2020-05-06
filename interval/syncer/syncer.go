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
	FetchAccount(address string, hash common.HashHeight, prevCnt uint64)
	FetchSnapshot(hash common.HashHeight, prevCnt uint64)
	Fetch(request face.FetchRequest)
}

type Sender interface {
	// when new block create
	BroadcastAccountBlocks(string, []*common.AccountStateBlock) error
	BroadcastSnapshotBlocks([]*common.SnapshotBlock) error

	// when fetch block message be arrived
	SendAccountBlocks(string, []*common.AccountStateBlock, p2p.Peer) error
	SendSnapshotBlocks([]*common.SnapshotBlock, p2p.Peer) error

	SendAccountHashes(string, []common.HashHeight, p2p.Peer) error
	SendSnapshotHashes([]common.HashHeight, p2p.Peer) error

	RequestAccountHash(string, common.HashHeight, uint64) error
	RequestSnapshotHash(common.HashHeight, uint64) error
	RequestAccountBlocks(string, []common.HashHeight) error
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

func (self *syncer) DefaultHandler() MsgHandler {
	return self.receiver
}

func NewSyncer(net p2p.P2P, bus EventBus.Bus) Syncer {
	self := &syncer{bus: bus}
	self.sender = &sender{net: net}
	self.p2p = net
	self.fetcher = &fetcher{sender: self.sender, retryPolicy: &defaultRetryPolicy{fetchedHashs: make(map[string]*RetryStatus)}, addressRetry: &addressRetryPolicy{}}
	return self
}
func (self *syncer) Init(reader face.ChainReader, writer face.PoolWriter) {
	rw := &chainRw{ChainReader: reader, PoolWriter: writer}
	self.state = newState(rw, self.fetcher, self.sender, self.p2p, self.bus)
	self.receiver = newReceiver(self.fetcher, rw, self.sender, self.state)
	self.p2p.SetHandlerFn(self.DefaultHandler().Handle)
	self.p2p.SetHandShaker(self.state)
}

func (self *syncer) Start() {
	self.state.start()
}
func (self *syncer) Stop() {
	self.state.stop()
}

func (self *syncer) Fetcher() Fetcher {
	return self.fetcher
}

func (self *syncer) Sender() Sender {
	return self.sender
}

func (self *syncer) Handlers() Handlers {
	return self.receiver
}

func (self *syncer) Done() bool {
	return self.state.syncDone()
}
