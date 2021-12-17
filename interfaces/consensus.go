package interfaces

import (
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}
