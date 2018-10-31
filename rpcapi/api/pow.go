package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
)

type Pow struct {
}

func (p Pow) GetPowNonce(difficulty string, data types.Hash) []byte {
	log.Info("GetPowNonce")
	b, _ := pow.GetPowNonce(nil, data)
	log.Info("GetPowNonce End")
	return b[:]
}
