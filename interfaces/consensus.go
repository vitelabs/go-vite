package interfaces

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}

type TimeIndex interface {
	Index2Time(index uint64) (time.Time, time.Time)
	Time2Index(t time.Time) uint64
}
