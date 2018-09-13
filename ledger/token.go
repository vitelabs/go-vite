package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Token struct {
	TokenId   types.TokenTypeId
	TokenName string

	Decimals    uint8
	TotalSupply *big.Int
}

func ViteTokenId() *types.TokenTypeId {
	return &types.TokenTypeId{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
}
