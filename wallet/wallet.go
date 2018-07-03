package wallet

import "github.com/vitelabs/go-vite/common/types"

type Wallet interface {
	Status() (string, error)

	Close() error

	Open(passphrase string) error

	ListAddress() []types.Address

	// use cached priv to sign data if the priv of address binding has`t evet
	SignData(a types.Address, data []byte) ([]byte, error)

	SignDataWithPassphrase(a types.Address, passphrase, data []byte) ([]byte, error)
}
