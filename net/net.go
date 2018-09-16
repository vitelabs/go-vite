package net

import (
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"sync"
	"time"
)

type Config struct {
	NetID uint64
}

type Net struct {
	*Config
	start         time.Time
	peers         *peerSet
	snapshotFeed  *snapshotBlockFeed
	accountFeed   *accountBlockFeed
	term          chan struct{}
	pool          *reqPool
	FromHeight    *big.Int
	TargetHeight  *big.Int
	syncState     SyncState
	slock         sync.RWMutex // use for syncState change
	SyncStartHook func(*big.Int, *big.Int)
	SyncDoneHook  func(*big.Int, *big.Int)
	SyncErrHook   func(*big.Int, *big.Int)
	stateFeed     *SyncStateFeed
	SnapshotChain BlockChain
	blockRecord   *cuckoofilter.CuckooFilter // record blocks has retrieved from network
}

func New(cfg *Config) *Net {
	n := &Net{
		Config:       cfg,
		peers:        NewPeerSet(),
		snapshotFeed: new(snapshotBlockFeed),
		accountFeed:  new(accountBlockFeed),
		stateFeed:    new(SyncStateFeed),
		term:         make(chan struct{}),
		pool:         new(reqPool),
		blockRecord:  cuckoofilter.NewCuckooFilter(10000),
	}

	return n
}

func (n *Net) Start() {
	n.start = time.Now()
}

func (n *Net) Stop() {
	select {
	case <-n.term:
	default:
		close(n.term)
	}
}

func (n *Net) Syncing() bool {
	n.slock.RLock()
	defer n.slock.RUnlock()
	return n.syncState == Syncing
}

func (n *Net) SetSyncState(st SyncState) {
	n.slock.Lock()
	defer n.slock.Unlock()
	n.syncState = st
	n.stateFeed.Notify(st)
}

func (n *Net) ReceiveConn(conn net.Conn) {
	select {
	case <-n.term:
	default:
	}

}

func (n *Net) HandlePeer(p *Peer) {
	head, err := n.SnapshotChain.GetLatestSnapshotBlock()
	if err != nil {
		log.Fatal("cannot get current block", err)
	}

	genesis, err := n.SnapshotChain.GetGenesesBlock()
	if err != nil {
		log.Fatal("cannot get genesis block", err)
	}

	err = p.Handshake(n.NetID, head.Height, head.Hash, genesis.Hash)
	if err != nil {
		return
	}
}

func (n *Net) HandleMsg(p *Peer) (err error) {
	msg, err := p.ts.ReadMsg()
	if err != nil {
		return
	}
	defer msg.Discard()

	Code := Cmd(msg.Cmd)
	if Code == HandshakeCode {
		return errHandshakeTwice
	}

	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return
	}

	switch Cmd(msg.Cmd) {
	case StatusCode:
		status := new(BlockID)
		err = status.Deserialize(payload)
		if err != nil {
			return
		}
		p.SetHead(status.Hash, status.Height)
	case GetSubLedgerCode:
		seg := new(Segment)
		err = seg.Deserialize(payload)
		if err != nil {
			return
		}

	case GetSnapshotBlockHeadersCode:
	case GetSnapshotBlockBodiesCode:
	case GetSnapshotBlocksCode:
	case GetSnapshotBlocksByHashCode:
	case GetAccountBlocksCode:
	case GetAccountBlocksByHashCode:
	case SubLedgerCode:
	case SnapshotBlockHeadersCode:
	case SnapshotBlockBodiesCode:
	case SnapshotBlocksCode:
	case AccountBlocksCode:
	case NewSnapshotBlockCode:
	case ExceptionCode:
	default:

	}

	return nil
}

func (n *Net) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock, propagate bool) {
	for _, block := range blocks {
		go func(b *ledger.SnapshotBlock) {
			peers := n.peers.UnknownBlock(block.Hash)

			for _, peer := range peers {
				go peer.SendNewSnapshotBlock(block)
			}
		}(block)
	}
}

func (n *Net) BroadcastAccountBlocks(blocks []*ledger.AccountBlock, propagate bool) {
	for _, block := range blocks {
		peers := n.peers.UnknownBlock(block.Hash)
		for _, peer := range peers {
			go func(peers []*Peer, b *ledger.AccountBlock) {
				peer.SendAccountBlocks([]*ledger.AccountBlock{block})
			}(peers, block)
		}
	}
}

func (n *Net) FetchSnapshotBlocks(s *Segment) {

}

func (n *Net) FetchSnapshotBlocksByHash(hashes []types.Hash) {

}

func (n *Net) FetchAccountBlocks(as AccountSegment) {

}

func (n *Net) FetchAccountBlocksByHash(hashes []types.Hash) {

}

func (n *Net) SubscribeAccountBlock(fn func(block *ledger.AccountBlock)) (subId int) {
	return n.accountFeed.Sub(fn)
}

func (n *Net) UnsubscribeAccountBlock(subId int) {
	n.accountFeed.Unsub(subId)
}

func (n *Net) receiveAccountBlock(block *ledger.AccountBlock) {
	n.accountFeed.Notify(block)
}

func (n *Net) SubscribeSnapshotBlock(fn func(block *ledger.SnapshotBlock)) (subId int) {
	return n.snapshotFeed.Sub(fn)
}

func (n *Net) UnsubscribeSnapshotBlock(subId int) {
	n.snapshotFeed.Unsub(subId)
}

func (n *Net) receiveSnapshotBlock(block *ledger.SnapshotBlock) {
	n.snapshotFeed.Notify(block)
}

func (n *Net) SubscribeSyncStatus(fn func(SyncState)) (subId int) {
	return n.stateFeed.Sub(fn)
}

func (n *Net) UnsubscribeSyncStatus(subId int) {
	n.stateFeed.Unsub(subId)
}

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	running := true
	select {
	case <-n.term:
		running = false
	default:
	}

	n.slock.RLock()
	st := n.syncState
	n.slock.RUnlock()

	return &NetStatus{
		Peers:     n.peers.Info(),
		Running:   running,
		Uptime:    time.Now().Sub(n.start),
		SyncState: st,
	}
}

type NetStatus struct {
	Peers     []*PeerInfo
	SyncState SyncState
	Uptime    time.Duration
	Running   bool
}
