package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"net"
	"sync"
	"time"
)

type Net struct {
	start        time.Time
	peers        *peerSet
	snapshotFeed *snapshotBlockFeed
	accountFeed  *accountBlockFeed

	stop chan struct{}

	pool *reqPool

	// for sync
	FromHeight   *big.Int
	TargetHeight *big.Int
	SyncState
	StateLock     sync.RWMutex
	SyncStartHook func(*big.Int, *big.Int)
	SyncDoneHook  func(*big.Int, *big.Int)
	SyncErrHook   func(*big.Int, *big.Int)

	SnapshotChain BlockChain
}

func New() *Net {
	peers := NewPeerSet()
	n := &Net{
		peers:        peers,
		snapshotFeed: NewSnapshotBlockFeed(),
		accountFeed:  NewAccountBlockFeed(),
	}

	return n
}

func (n *Net) Start() {
	n.start = time.Now()
}

func (n *Net) Stop() {
	select {
	case <-n.stop:
	default:
		close(n.stop)
	}
}

func (this *Net) Syncing() bool {
	this.StateLock.RLock()
	defer this.StateLock.RUnlock()
	return this.SyncState == syncing
}

func (n *Net) ReceiveConn(conn net.Conn) {
	select {
	case <-n.stop:
	default:
	}

}

func (n *Net) HandlePeer(p *Peer) {

}

func (n *Net) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock, propagate bool) {

}

func (n *Net) BroadcastAccountBlocks(blocks []*ledger.AccountBlock, propagate bool) {

}

type snap struct {
	From    types.Hash
	To      types.Hash
	Count   uint64
	Forward bool
	Step    int
}

func (n *Net) FetchSnapshotBlocks(s *snap) {

}

type ac map[types.Address]*snap

func (n *Net) FetchAccountBlocks(a ac) {

}

func (n *Net) SubscribeAccountBlock() *snapshotBlockSub {
	return n.snapshotFeed.Subscribe()
}

func (n *Net) SubscribeSnapshotBlock() *accountBlockSub {
	return n.accountFeed.Subscribe()
}

func (n *Net) SubscribeSyncStatus(func(SyncState)) (subId int) {

}

func (n *Net) UnsubscribeSyncStatus(subId int) {

}

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	running := true
	select {
	case <-n.stop:
		running = false
	default:
	}

	return &NetStatus{
		Peers:   n.peers.Info(),
		Running: running,
		Uptime:  time.Now().Sub(n.start),
	}
}

type NetStatus struct {
	Peers      []*PeerInfo
	SyncStatus SyncState
	Uptime     time.Duration
	Running    bool
}
