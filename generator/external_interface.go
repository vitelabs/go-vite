package generator

import "github.com/vitelabs/go-vite/common/types"

type SignManager interface {
	SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error)
	SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error)
}