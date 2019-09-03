package interfaces

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"math/big"
)

type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}
