package net

import (
	"fmt"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"time"
)

// all query include start block
type Chain interface {
	// query common Hash
	GetAbHashList(start, count, step uint64, forward bool) ([]*ledger.HashHeight, error)

	// the second return value mean chunk befor/after file
	GetSubLedgerByHeight(start, count uint64, forward bool) ([]string, [][2]uint64)
	GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]string, [][2]uint64, error)

	// query chunk
	GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	Compressor() *compress.Compressor
}

type Config struct {
	Port  uint16
	Chain Chain
}

type Net struct {
	*Config
	peers        *peerSet
	snapshotFeed *snapshotBlockFeed
	accountFeed  *accountBlockFeed
	term         chan struct{}
	blockRecord  *cuckoofilter.CuckooFilter // record blocks has retrieved from network
	log          log15.Logger
	Protocols    []*p2p.Protocol // mount to p2p.Server
	wg           sync.WaitGroup
	fileServer   *FileServer
	newBlocks    []*ledger.SnapshotBlock // before syncDone, cache newBlocks
	syncer       *syncer
	handlers     map[cmd]MsgHandler
}

func New(cfg *Config) (*Net, error) {
	n := &Net{
		Config:       cfg,
		snapshotFeed: new(snapshotBlockFeed),
		accountFeed:  new(accountBlockFeed),
		term:         make(chan struct{}),
		blockRecord:  cuckoofilter.NewCuckooFilter(10000),
		log:          log15.New("module", "vite/net"),
	}

	peerSet := NewPeerSet()
	pool := newRequestPool(peerSet)
	sync := newSyncer(cfg.Chain, peerSet, pool, n)

	fileServer, err := newFileServer(cfg.Port, cfg.Chain, peerSet, sync)
	if err != nil {
		return nil, err
	}

	n.peers = peerSet
	n.fileServer = fileServer
	n.syncer = sync

	n.AddHandler(_statusHandler(statusHandler))
	n.AddHandler(&forkHandler{cfg.Chain})
	n.AddHandler(&getSubLedgerHandler{cfg.Chain})
	n.AddHandler(&getSnapshotBlocksHandler{cfg.Chain})
	n.AddHandler(&getAccountBlocksHandler{cfg.Chain})
	n.AddHandler(&getChunkHandler{cfg.Chain})
	n.AddHandler(&blocksHandler{sync})
	// todo
	n.AddHandler(&newSnapshotBlockHandler{})
	n.AddHandler(fileServer)

	//
	n.Protocols = make([]*p2p.Protocol, len(cmdSets))
	for i, cmdset := range cmdSets {
		n.Protocols[i] = &p2p.Protocol{
			Name: CmdSetName,
			ID:   cmdset,
			Handle: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				// will be called by p2p.Peer.runProtocols use goroutine
				peer := newPeer(p, rw, cmdset)
				return n.HandlePeer(peer)
			},
		}
	}

	return n, nil
}

func (n *Net) AddHandler(handler MsgHandler) {
	cmds := handler.Cmds()
	for _, cmd := range cmds {
		n.handlers[cmd] = handler
	}
}

func (n *Net) Stop() {
	select {
	case <-n.term:
	default:
		close(n.term)
	}

	n.fileServer.stop()
}

func (n *Net) Syncing() bool {
	return n.syncer.state == Syncing
}

func (n *Net) SyncState() SyncState {
	return n.syncer.state
}

// will be called by p2p.Server, run as goroutine
func (n *Net) HandlePeer(p *Peer) error {
	current := n.Chain.GetLatestSnapshotBlock()
	genesis := n.Chain.GetGenesisSnapshotBlock()

	err := p.Handshake(&message.HandShake{
		CmdSet:  p.CmdSet,
		Height:  current.Height,
		Port:    n.Port,
		Current: current.Hash,
		Genesis: genesis.Hash,
	})

	if err != nil {
		return err
	}

	return n.startPeer(p)
}

func (n *Net) startPeer(p *Peer) error {
	n.peers.Add(p)
	defer n.peers.Del(p)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	go n.syncer.sync()

	for {
		select {
		case <-n.term:
			return p2p.DiscQuitting
		case err := <-p.errch:
			return err
		case <-ticker.C:
			current := n.Chain.GetLatestSnapshotBlock()
			p.Send(StatusCode, 0, &ledger.HashHeight{
				Hash:   current.Hash,
				Height: current.Height,
			})
		default:
			if err := n.handleMsg(p); err != nil {
				return err
			}
		}
	}

	close(p.term)
	return nil
}

func (n *Net) handleMsg(p *Peer) (err error) {
	msg, err := p.mrw.ReadMsg()
	if err != nil {
		return
	}
	defer msg.Discard()

	code := cmd(msg.Cmd)
	if code == HandshakeCode {
		return errHandshakeTwice
	}

	handler := n.handlers[code]
	if handler != nil {
		return handler.Handle(msg, p)
	}

	return fmt.Errorf("unknown message cmd %d", msg.Cmd)
}

func (n *Net) receiveNewBlocks(block *ledger.SnapshotBlock) {
	if n.Syncing() {
		n.newBlocks = append(n.newBlocks, block)
	} else {
		// todo
	}
}

type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock)
	BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock)
}

type Fetcher interface {
	FetchSnapshotBlocks(start types.Hash, count uint64)
	FetchAccountBlocks(start types.Hash, count uint64, address types.Address)
}

type SnapshotBlockCallback func(block *ledger.SnapshotBlock)
type AccountblockCallback func(addr types.Address, block *ledger.AccountBlock)
type SyncStateCallback func(SyncState)

type Subscriber interface {
	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeAccountBlock(fn AccountblockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSyncStatus(subId int)
}

func (n *Net) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	peers := n.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		go peer.SendNewSnapshotBlock(block)
	}
}

func (n *Net) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	peers := n.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		go func(peer *Peer) {
			peer.SendAccountBlocks(addr, []*ledger.AccountBlock{block}, 0)
		}(peer)
	}
}

func (n *Net) BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		n.BroadcastAccountBlock(addr, block)
	}
}

func (n *Net) FetchSnapshotBlocks(start types.Hash, count uint64) {
	// if the sync data has not downloaded, then ignore fetch request
}

func (n *Net) FetchAccountBlocks(start types.Hash, count uint64, address types.Address) {
	// if the sync data has not downloaded, then ignore fetch request
}

func (n *Net) SubscribeAccountBlock(fn AccountblockCallback) (subId int) {
	return n.accountFeed.Sub(fn)
}

func (n *Net) UnsubscribeAccountBlock(subId int) {
	n.accountFeed.Unsub(subId)
}

func (n *Net) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	return n.snapshotFeed.Sub(fn)
}

func (n *Net) UnsubscribeSnapshotBlock(subId int) {
	n.snapshotFeed.Unsub(subId)
}

func (n *Net) SubscribeSyncStatus(fn func(SyncState)) (subId int) {
	return n.syncer.feed.Sub(fn)
}

func (n *Net) UnsubscribeSyncStatus(subId int) {
	n.syncer.feed.Unsub(subId)
}

// @implementation of BlockReceiver
func (n *Net) receiveSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		// todo verify
		if !n.blockRecord.Lookup(block.Hash[:]) {
			n.blockRecord.InsertUnique(block.Hash[:])
			n.snapshotFeed.Notify(block)
		}
	}
}

func (n *Net) receiveAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock) {
	for _, ablocks := range mblocks {
		for _, ablock := range ablocks {
			// todo verify
			if !n.blockRecord.Lookup(ablock.Hash[:]) {
				n.blockRecord.InsertUnique(ablock.Hash[:])
				n.accountFeed.Notify(ablock)
			}
		}
	}
}

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	running := true
	select {
	case <-n.term:
		running = false
	default:
	}

	return &NetStatus{
		Peers:     n.peers.Info(),
		Running:   running,
		SyncState: n.syncer.state,
	}
}

type NetStatus struct {
	Peers     []*PeerInfo
	SyncState SyncState
	Running   bool
}
