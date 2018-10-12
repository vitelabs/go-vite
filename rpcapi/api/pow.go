package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
)

type Pow struct {
}

func (p Pow) GetPowNonce(difficulty string, data types.Hash) []byte {
	b := pow.GetPowNonce(nil, data)
	return b[:]
}
