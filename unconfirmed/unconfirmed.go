package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// db struct
type unconfirmedBlock struct {
	gid 	string
	address types.Address
	hash    types.Hash
}

// memery struct
type TokenInfo struct {
	TokenId     *types.TokenTypeId
	TotalAmount *big.Int
}

type UnconfirmedMeta struct {
	TotalNumber   *big.Int
	TokenInfoList []*TokenInfo
}
