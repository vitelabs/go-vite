package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
)

type Pow struct {
}

func (p Pow) GetPowNonce(difficulty string, data types.Hash) []byte {
	log.Info("GetPowNonce")
	//b := pow.GetPowNonce(nil, data)
	b, err := pow.GetPowNonceFromRemote(nil, data)
	if err != nil {
		log.Error("GetPowNonceFromRemote", "error", err)
	}
	log.Info("GetPowNonce End")
	return b[:]
}
