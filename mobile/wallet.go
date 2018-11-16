package mobile

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet"
	"path/filepath"
)

type DerivationResult struct {
	Path    string
	Address Address
}

type NewEntropyResult struct {
	Mnemonic     string
	EntropyStore string
}

type Wallet struct {
	wallet *wallet.Manager
}

func NewWallet(dataDir string, maxSearchIndex int, useLightScrypt bool) *Wallet {
	return &Wallet{
		wallet: wallet.New(&wallet.Config{
			DataDir:        dataDir,
			MaxSearchIndex: uint32(maxSearchIndex),
			UseLightScrypt: useLightScrypt,
		}),
	}
}

func (w Wallet) ListAllEntropyFiles() []string {
	return w.wallet.ListAllEntropyFiles()
}

func (w *Wallet) Unlock(entropyStore, passphrase string) error {
	return w.wallet.Unlock(entropyStore, passphrase)
}

func (w *Wallet) IsUnlocked(entropyStore string) bool {
	return w.wallet.IsUnlocked(entropyStore)
}

func (w *Wallet) Lock(entropyStore string) error {
	return w.wallet.Lock(entropyStore)
}

func (w *Wallet) AddEntropyStore(entropyStore string) error {
	return w.wallet.AddEntropyStore(entropyStore)
}

func (w *Wallet) RemoveEntropyStore(entropyStore string) {
	w.wallet.RemoveEntropyStore(entropyStore)
}

func (w *Wallet) RecoverEntropyStoreFromMnemonic(mnemonic, newPassphrase, language, extensionWord string) (entropyStore *string, err error) {
	em, e := w.wallet.RecoverEntropyStoreFromMnemonic(mnemonic, language, newPassphrase, &extensionWord)
	if e != nil {
		return nil, e
	}
	f := em.GetPrimaryAddr().String()
	return &f, nil
}

func (w *Wallet) NewMnemonicAndEntropyStore(passphrase, language, extensionWord string, mnemonicSize int) (result *NewEntropyResult, err error) {
	mnemonic, em, e := w.wallet.NewMnemonicAndEntropyStore(passphrase, passphrase, &extensionWord, &mnemonicSize)
	if e != nil {
		return nil, e
	}

	return &NewEntropyResult{
		Mnemonic:     mnemonic,
		EntropyStore: em.GetPrimaryAddr().String(),
	}, nil
}

func (w *Wallet) DeriveByFullPath(entropyStore, fullpath, extensionWord string) (dResult *DerivationResult, err error) {
	manager, err := w.wallet.GetEntropyStoreManager(entropyStore)
	if err != nil {
		return nil, err
	}

	s, key, err := manager.DeriveForFullPath(fullpath, &extensionWord)
	if err != nil {
		return nil, err
	}
	addr, err := key.Address()
	if err != nil {
		return nil, err
	}
	return &DerivationResult{
		Path: s,
		Address: Address{
			address: *addr,
		},
	}, nil
}

func (w *Wallet) DeriveByIndex(entropyStore string, index int, extensionWord string) (dResult *DerivationResult, err error) {
	manager, err := w.wallet.GetEntropyStoreManager(entropyStore)
	if err != nil {
		return nil, err
	}

	s, key, err := manager.DeriveForIndexPath(uint32(index), &extensionWord)
	if err != nil {
		return nil, err
	}
	addr, err := key.Address()
	if err != nil {
		return nil, err
	}
	return &DerivationResult{
		Path: s,
		Address: Address{
			address: *addr,
		},
	}, nil
}

func (w Wallet) GetDataDir() string {
	return w.wallet.GetDataDir()
}

func (w *Wallet) Start() {
	w.wallet.Start()
}

func (w *Wallet) Stop() {
	w.wallet.Stop()
}

func EntropyStoreToAddress(entropyStore string) (*Address, error) {
	addrStr := entropyStore
	if filepath.IsAbs(entropyStore) {
		addrStr = filepath.Base(entropyStore)
	}
	address, err := types.HexToAddress(addrStr)
	if err != nil {
		return nil, err
	}
	return &Address{
		address: address,
	}, nil
}
