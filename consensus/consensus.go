package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Verifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error)
}

type Event struct {
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp      time.Time  // add to block
	SnapshotHash   types.Hash // add to block
	SnapshotHeight uint64     // add to block
}

type Subscriber interface {
	Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event))
	UnSubscribe(gid types.Gid, id string)
}

type Consensus interface {
	Verifier
	Subscriber
}
