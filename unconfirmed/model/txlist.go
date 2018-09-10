package model

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Tx struct {
	Hash   *types.Hash
	amount *big.Int
}
