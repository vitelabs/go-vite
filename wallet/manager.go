package wallet

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"github.com/vitelabs/go-vite/log"
	"fmt"
)

type Manager struct {
	keystoreManager *keystore.Manager
}

func (m Manager) NewAddress(pwd []string, reply *string) error {
	log.Info("NewAddress")
	if len(pwd) != 1 {
		return fmt.Errorf("password len error")
	}
	key, err := m.keystoreManager.StoreNewKey(pwd[0])
	if err != nil {
		return err
	}
	*reply = key.Address.Hex()
	return nil
}

func (m Manager) ListAddress(v interface{}, reply *string) error {
	log.Info("ListAddress")
	*reply = types.Addresses(m.keystoreManager.Addresses()).String()
	return nil
}

func NewManager(walletdir string) *Manager {
	return &Manager{
		keystoreManager: keystore.NewManager(walletdir),
	}
}

func (m *Manager) Init() {
	m.keystoreManager.Init()
}
