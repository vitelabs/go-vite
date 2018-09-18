package producer

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

type AccountEvent interface {
}

type AccountStartEvent struct {
	AccountEvent
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp      time.Time  // add to block
	SnapshotHash   types.Hash // add to block
	SnapshotHeight uint64     // add to block
}

type Producer interface {
	SetAccountEventFunc(func(AccountEvent))
}
