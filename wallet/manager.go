package wallet

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"path/filepath"
)

type Manager struct {
	KeystoreManager *keystore.Manager
	config          *Config
}

func New(config *Config) *Manager {
	if config == nil {
		config = &Config{DataDir: filepath.Join(common.DefaultDataDir(), "wallet")}
	}
	km := keystore.NewManager(config.DataDir)
	km.Init()
	return &Manager{
		KeystoreManager: km,
		config:          config,
	}
}

func (m *Manager) Start() {

}
func (m *Manager) Stop() {

}
