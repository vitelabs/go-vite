package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/verifier"
)

type SignManager interface {
	SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error)
	SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error)
}

type Verifier interface {
	Verifier() *verifier.AccountVerifier
}