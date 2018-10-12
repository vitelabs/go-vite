package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
)

type Pow struct {
}

func GetPowNonce(difficulty string, data types.Hash) []byte {
	return pow.GetPowNonce(nil, data)[:]
}
