package wallet

import "github.com/vitelabs/go-vite/common"

type Wallet interface {
	Status() (string, error)

	Close() error

	Open(passphrase string) error

	ListAddress() []common.Address

	// use cached priv to sign data if the priv of address binding has`t evet
	SignData(a common.Address, data []byte) ([]byte, error)

	SignDataWithPassphrase(a common.Address, passphrase, data []byte) ([]byte, error)
}
