package statistics

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

//todo
type AccountInfo struct {
}

type OnroadAccountInfo struct {
	AccountAddress      types.Address
	TotalNumber         uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*TokenBalanceInfo
}

type TokenBalanceInfo struct {
	TotalAmount big.Int
	Number      uint64
}
