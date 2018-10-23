package wallet

import (
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/wallet/seedstore"
)

type Manager struct {
	seedStoreManager *seedstore.Manager
	config           *Config
}

func New(config *Config) *Manager {
	if config == nil {
		return nil
	}
	return &Manager{
		config: config,
	}
}

func (m Manager) GetSeedStoreManager() *seedstore.Manager {
	return m.seedStoreManager
}

func (m *Manager) RecoverSeedStoreFromMnemonic(mnemonic string, seedStorePassword string, switchToIt bool, seedPassword *string) (seedStoreFile string, err error) {
	rawSeedPassword := ""
	if seedPassword != nil {
		rawSeedPassword = *seedPassword
	}
	seed := bip39.NewSeed(mnemonic, rawSeedPassword)

	sm, e := seedstore.StoreNewSeed(m.config.DataDir, seed, seedStorePassword)
	if e != nil {
		return "", nil
	}
	if switchToIt {
		m.seedStoreManager = sm
	}
	return sm.SeedStoreFile(), e
}

func (m *Manager) SwitchSeedStore(fullSeedStoreFile string) error {
	m.seedStoreManager = seedstore.NewManager(fullSeedStoreFile)
	return nil
}

// seedStorePassword :  represents the keystore file`s password
// seedPassword:  in bip 39 when generate a seed from a mnemonic you can pass a password or not
func (m *Manager) NewMnemonicAndSeedStore(seedStorePassword string, switchToIt bool, seedPassword *string) (mnemonic, seedStoreFile string, err error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", "", nil
	}
	mnemonic, err = bip39.NewMnemonic(entropy)
	if err != nil {
		return "", "", nil
	}

	file, e := m.RecoverSeedStoreFromMnemonic(mnemonic, seedStorePassword, switchToIt, seedPassword)
	if e != nil {
		return "", "", e
	}
	return mnemonic, file, nil
}
