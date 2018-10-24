package wallet

import (
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/wallet/entropystore"
)

type Manager struct {
	entropyStoreManager *entropystore.Manager
	config              *Config
}

func New(config *Config) *Manager {
	if config == nil {
		return nil
	}
	return &Manager{
		config: config,
	}
}

func (m Manager) GetEntropyStoreManager() *entropystore.Manager {
	return m.entropyStoreManager
}

func (m *Manager) RecoverEntropyStoreFromMnemonic(mnemonic string, password string, switchToIt bool) (em *entropystore.Manager, err error) {
	sm, e := entropystore.StoreNewEntropy(m.config.DataDir, mnemonic, password, entropystore.DefaultMaxIndex)
	if e != nil {
		return nil, e
	}
	if switchToIt {
		m.switchEntropyStore(sm)
	}
	return sm, e
}

func (m *Manager) switchEntropyStore(sm *entropystore.Manager) error {
	if sm == nil {
		return errors.New("nil entropy manager")
	}
	if m.entropyStoreManager != nil {
		m.entropyStoreManager.Lock()
	}
	m.entropyStoreManager = sm
	return nil
}

func (m *Manager) SwitchEntropyStore(absFilename string) error {
	if m.entropyStoreManager != nil {
		m.entropyStoreManager.Lock()
	}
	mayValid, addr, e := entropystore.IsMayValidEntropystoreFile(absFilename)
	if e != nil {
		return e
	}
	if !mayValid {
		return errors.New("not valid entropy store file")
	}
	m.entropyStoreManager = entropystore.NewManager(absFilename, *addr, entropystore.DefaultMaxIndex)
	return nil
}

func (m *Manager) NewMnemonicAndEntropyStore(password string, switchToIt bool) (mnemonic string, em *entropystore.Manager, err error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", nil, nil
	}
	mnemonic, err = bip39.NewMnemonic(entropy)
	if err != nil {
		return "", nil, nil
	}

	em, e := m.RecoverEntropyStoreFromMnemonic(mnemonic, password, switchToIt)
	if e != nil {
		return "", nil, e
	}
	return mnemonic, em, nil
}
