package keystore

import (
	"github.com/vitelabs/go-vite/common/types"
)

// keystorewallet is a simple soft wallet that  only owns one address
type keystoreWallet struct {
	address types.Address
	keyPoll *Manager // Keystore where the account originates from
}

func (ksw *keystoreWallet) Status() (string, error) {
	panic("implement me")
}

// keystore wallet is a soft wallet we dont need to Close it
func (ksw *keystoreWallet) Close() error {
	return nil
}

// keystore wallet is a soft wallet we dont need to Open it
func (ksw *keystoreWallet) Open(passphrase string) error {
	return nil
}

func (ksw *keystoreWallet) ListAddress() []types.Address {
	panic("implement me")
}

func (ksw *keystoreWallet) SignData(a types.Address, data []byte) ([]byte, error) {
	panic("implement me")
}

func (ksw *keystoreWallet) SignDataWithPassphrase(a types.Address, passphrase, data []byte) ([]byte, error) {
	panic("implement me")
}
