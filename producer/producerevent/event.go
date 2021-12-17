package producerevent

import (
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
)

type AccountEvent interface {
}

type AccountStartEvent struct {
	AccountEvent
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time
}
