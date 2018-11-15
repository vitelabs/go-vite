package producerevent

import (
	"github.com/vitelabs/go-vite/common/types"
	"time"
)

type AccountEvent interface {
}

type AccountStartEvent struct {
	AccountEvent
	Gid types.Gid

	EntropyStorePath string
	Bip44Index       uint32
	Address          types.Address

	Stime time.Time
	Etime time.Time

	Timestamp      time.Time  // add to block
	SnapshotHash   types.Hash // add to block
	SnapshotHeight uint64     // add to block
}
