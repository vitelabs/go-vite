package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"time"
)

// all query include from block
type Chain interface {
	// the second return value mean chunk befor/after file
	GetSubLedgerByHeight(start, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64)
	GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error)

	// query chunk
	GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	Compressor() *compress.Compressor
}

type Verifier interface {
	VerifyforP2P(block *ledger.AccountBlock) bool
}

type Config struct {
	Port     uint16
	Chain    Chain
	Verifier Verifier
}

const DefaultPort uint16 = 8484

type Net struct {
	*Config
	peers       *peerSet
	syncer      *syncer
	fetcher     *fetcher
	receiver    *receiver
	broadcaster *broadcaster
	term        chan struct{}
	log         log15.Logger
	Protocols   []*p2p.Protocol // mount to p2p.Server
	wg          sync.WaitGroup
	fs          *fileServer
	fc          *fileClient
	handlers    map[cmd]MsgHandler
}

// auto from
func New(cfg *Config) (*Net, error) {
	port := cfg.Port
	if port == 0 {
		port = DefaultPort
	}
	fs, err := newFileServer(port, cfg.Chain)
	if err != nil {
		return nil, err
	}

	fc := newFileClient(cfg.Chain)

	peers := NewPeerSet()
	pool := newRequestPool()

	broadcaster := newBroadcaster(peers)
	filter := newFilter()
	receiver := newReceiver(cfg.Verifier, broadcaster, filter)
	syncer := newSyncer(cfg.Chain, peers, pool, receiver, fc)
	fetcher := newFetcher(filter, peers, receiver, pool)

	syncer.feed.Sub(receiver.listen) // subscribe sync status
	syncer.feed.Sub(fetcher.listen)  // subscribe sync status

	n := &Net{
		Config:      cfg,
		peers:       peers,
		syncer:      syncer,
		fetcher:     fetcher,
		receiver:    receiver,
		broadcaster: broadcaster,
		fs:          fs,
		fc:          fc,
		term:        make(chan struct{}),
		handlers:    make(map[cmd]MsgHandler),
		log:         log15.New("module", "vite/net"),
	}

	pool.ctx = &context{
		syncer: syncer,
		peers:  peers,
		pool:   pool,
		fc:     fc,
	}

	n.AddHandler(_statusHandler(statusHandler))
	n.AddHandler(&getSubLedgerHandler{cfg.Chain})
	n.AddHandler(&getSnapshotBlocksHandler{cfg.Chain})
	n.AddHandler(&getAccountBlocksHandler{cfg.Chain})
	n.AddHandler(&getChunkHandler{cfg.Chain})
	n.AddHandler(pool)     // receive all response except NewSnapshotBlockCode
	n.AddHandler(receiver) // receive newBlocks

	n.Protocols = append(n.Protocols, &p2p.Protocol{
		Name: CmdSetName,
		ID:   CmdSet,
		Handle: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			// will be called by p2p.Peer.runProtocols use goroutine
			peer := newPeer(p, rw, CmdSet)
			return n.HandlePeer(peer)
		},
	})

	go n.fs.start()

	go n.fc.start()

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
		n.syncer.stop()
		n.fs.stop()
		n.fc.stop()
	}
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

	n.log.Info(fmt.Sprintf("handshake with %s", p))
	err := p.Handshake(&message.HandShake{
		CmdSet:  p.CmdSet,
		Height:  current.Height,
		Port:    n.Port,
		Current: current.Hash,
		Genesis: genesis.Hash,
	})

	if err != nil {
		n.log.Error(fmt.Sprintf("handshake with %s error: %v", p, err))
		return err
	}
	n.log.Info(fmt.Sprintf("handshake with %s done", p))

	return n.startPeer(p)
}

func (n *Net) startPeer(p *Peer) error {
	n.peers.Add(p)
	defer n.peers.Del(p)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	n.log.Info(fmt.Sprintf("startPeer %s", p))
	go n.syncer.start()

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
		n.log.Error(fmt.Sprintf("read message from %s error: %v", p, err))
		return
	}
	defer msg.Discard()

	code := cmd(msg.Cmd)
	n.log.Info(fmt.Sprintf("receive %s from %s", code, p))

	if code == HandshakeCode {
		n.log.Error(fmt.Sprintf("handshake twice with %s", p))
		return errHandshakeTwice
	}

	handler := n.handlers[code]
	if handler != nil {
		return handler.Handle(msg, p)
	}

	n.log.Error(fmt.Sprintf("missing handler for message %s", code))

	return fmt.Errorf("unknown message cmd %d", msg.Cmd)
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
	n.broadcaster.BroadcastSnapshotBlock(block)
}

func (n *Net) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	n.broadcaster.BroadcastAccountBlock(addr, block)
}

func (n *Net) BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock) {
	n.broadcaster.BroadcastAccountBlocks(addr, blocks)
}

func (n *Net) FetchSnapshotBlocks(start types.Hash, count uint64) {
	n.fetcher.FetchSnapshotBlocks(start, count)
}

func (n *Net) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	n.fetcher.FetchAccountBlocks(start, count, address)
}

func (n *Net) SubscribeAccountBlock(fn AccountblockCallback) (subId int) {
	return n.receiver.aFeed.Sub(fn)
}

func (n *Net) UnsubscribeAccountBlock(subId int) {
	n.receiver.aFeed.Unsub(subId)
}

func (n *Net) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	return n.receiver.sFeed.Sub(fn)
}

func (n *Net) UnsubscribeSnapshotBlock(subId int) {
	n.receiver.sFeed.Unsub(subId)
}

func (n *Net) SubscribeSyncStatus(fn func(SyncState)) (subId int) {
	return n.syncer.feed.Sub(fn)
}

func (n *Net) UnsubscribeSyncStatus(subId int) {
	n.syncer.feed.Unsub(subId)
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
