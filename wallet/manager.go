package wallet

import (
	"github.com/vitelabs/go-vite/wallet/keystore"
)

type Manager struct {
	KeystoreManager *keystore.Manager
}

func NewManager(walletdir string) *Manager {
	km := keystore.NewManager(walletdir)
	km.Init()
	return &Manager{
		KeystoreManager: km,
	}
}
