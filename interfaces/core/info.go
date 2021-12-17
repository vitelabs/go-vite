package core

import (
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
)

type AccountInfo struct {
	AccountAddress      types.Address
	TotalNumber         uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*TokenBalanceInfo
}

type TokenBalanceInfo struct {
	TotalAmount big.Int
	Number      uint64
}
