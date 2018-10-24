package wallet

import (
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Manager struct {
	entropyStoreManager *entropystore.Manager
	config              *Config
	unlockChangedIndex  int
	unlockChangedLis    map[int]func(event entropystore.UnlockEvent)
	mutex               sync.Mutex
}

func New(config *Config) *Manager {
	if config == nil {
		return nil
	}
	return &Manager{
		config:           config,
		unlockChangedLis: make(map[int]func(event entropystore.UnlockEvent)),
	}
}

func (m Manager) AddLockEventListener(lis func(event entropystore.UnlockEvent)) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.unlockChangedIndex++
	m.unlockChangedLis[m.unlockChangedIndex] = lis
	return m.unlockChangedIndex
}

func (m Manager) RemoveUnlockChangeChannel(id int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.unlockChangedLis, id)
}

func (m Manager) ListEntropyFiles() ([]string, error) {

	files, err := ioutil.ReadDir(m.config.DataDir)
	if err != nil {
		return nil, err
	}

	filenames := make([]string, 0)
	for _, file := range files {
		if file.IsDir() || file.Mode()&os.ModeType != 0 {
			continue
		}
		fn := file.Name()
		if strings.HasPrefix(fn, ".") || strings.HasSuffix(fn, "~") {
			continue
		}
		absFilePath := filepath.Join(m.config.DataDir, file.Name())
		b, _, e := entropystore.IsMayValidEntropystoreFile(absFilePath)
		if e != nil || !b {
			continue
		}
		filenames = append(filenames, absFilePath)
	}

	return filenames, nil
}

func (m Manager) GetEntropyStoreManager() (*entropystore.Manager, error) {
	if m.entropyStoreManager == nil {
		return nil, walleterrors.ErrEmptyStore
	}
	return m.entropyStoreManager, nil
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

func (m Manager) GetDataDir() string {
	return m.config.DataDir
}

func (m Manager) Start() {

}

func (m Manager) Stop() {

}
