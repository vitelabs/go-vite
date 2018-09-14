package producer

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

type AccountEvent struct {
}

type AccountStartEvent struct {
	AccountEvent
	Gid     uint32
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp      uint64     // add to block
	SnapshotHash   types.Hash // add to block
	SnapshotHeight uint64     // add to block
}

type Producer interface {
	SetAccountEventFunc(func(AccountEvent))
}
