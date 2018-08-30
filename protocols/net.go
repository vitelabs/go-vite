package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type Net struct {
	start        time.Time
	peers        *peerSet
	snapshotFeed *snapshotBlockFeed
	accountFeed  *accountBlockFeed
}

func New() *Net {
	return &Net{
		peers:        NewPeerSet(),
		snapshotFeed: NewSnapshotBlockFeed(),
		accountFeed:  NewAccountBlockFeed(),
	}
}

func (n *Net) Start() {
	n.start = time.Now()
}

func (n *Net) Stop() {

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

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	return &NetStatus{}
}

type NetStatus struct {
	Peers      []*PeerInfo
	SyncStatus int
	Uptime     time.Duration
	Running    bool
}
