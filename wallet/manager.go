package wallet

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"path/filepath"
)

type Manager struct {
	KeystoreManager *keystore.Manager
	DateDir         string
}

func NewManagerAndInit(walletdir string) *Manager {

	if walletdir == "" {
		walletdir = filepath.Join(common.DefaultDataDir(), "wallet")
	}
	walletdir = filepath.Join(walletdir, "wallet")
	km := keystore.NewManager(walletdir)

	km.Init()

	return &Manager{
		KeystoreManager: km,
		DateDir:         walletdir,
	}
}
