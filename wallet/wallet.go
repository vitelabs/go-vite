package wallet

import "github.com/vitelabs/go-vite/common/types"

type Wallet interface {
	Status() (string, error)

	Close() error

	Open(passphrase string) error

	Addresses() []types.Address

	// use cached priv to sign data if the priv of address binding has`t evet
	SignData(a types.Address, data []byte) (signedData []byte, pubkey []byte, err error)

	SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData []byte, pubkey []byte, err error)
}

type Provider interface {
	Wallets() []Wallet
	Subscribe(event chan<- Event)
}

type EventType int

const (
	WalletArrived EventType = iota

	WalletOpened

	WalletUpdate

	WalletDropped
)

type Event struct {
	Wallet Wallet
	Kind   EventType
}
