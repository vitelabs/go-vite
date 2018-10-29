package wallet

import (
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Manager struct {
	config              *Config
	unlockChangedIndex  int
	entropyStoreManager map[string]*entropystore.Manager // key is the entropyStore`s abs path
	unlockChangedLis    map[int]func(event entropystore.UnlockEvent)
	mutex               sync.Mutex

	log log15.Logger
}

func New(config *Config) *Manager {
	if config == nil {
		return nil
	}

	if config.MaxSearchIndex == 0 {
		config.MaxSearchIndex = entropystore.DefaultMaxIndex
	}

	return &Manager{
		config:              config,
		unlockChangedLis:    make(map[int]func(event entropystore.UnlockEvent)),
		entropyStoreManager: make(map[string]*entropystore.Manager),

		log: log15.New("module", "wallet"),
	}
}

func (m Manager) ListAllEntropyFiles() []string {
	files := make([]string, 0)
	for filename, _ := range m.entropyStoreManager {
		files = append(files, filename)
	}
	return files
}

func (m *Manager) Unlock(entropyStore, password string) error {
	manager, e := m.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return e
	}

	return manager.Unlock(password)
}

func (m *Manager) IsUnlocked(entropyStore string) bool {
	manager, e := m.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return false
	}
	return manager.IsUnlocked()
}

func (m *Manager) Lock(entropyStore string) error {
	manager, e := m.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return e
	}
	manager.Lock()
	return nil
}

func (m Manager) GlobalCheckAddrUnlock(targetAdr types.Address) bool {
	_, _, _, err := m.GlobalFindAddr(targetAdr)
	return err == nil
}

func (m *Manager) RefreshCache() {
	for filename, _ := range m.entropyStoreManager {
		_, e := os.Stat(filename)
		if e != nil {
			manager, ok := m.entropyStoreManager[filename]
			if ok {
				manager.Lock()
				delete(m.entropyStoreManager, filename)
			}
		}
	}
}

func (m Manager) GlobalFindAddr(targetAdr types.Address) (path string, key *derivation.Key, index uint32, err error) {
	for path, em := range m.entropyStoreManager {
		if em.IsUnlocked() {
			key, index, err = em.FindAddr(targetAdr)
			if err == walleterrors.ErrAddressNotFound {
				continue
			}
			if err != nil {
				return "", nil, 0, err
			}
			return path, key, index, nil
		}

	}
	return "", nil, 0, walleterrors.ErrAddressNotFound
}

func (m Manager) GlobalFindAddrWithPassphrase(targetAdr types.Address, pass string) (path string, key *derivation.Key, index uint32, err error) {
	for path, em := range m.entropyStoreManager {
		key, index, err = em.FindAddrWithPassword(pass, targetAdr)
		if err == walleterrors.ErrAddressNotFound {
			continue
		}
		if err != nil {
			return "", nil, 0, err
		}
		return path, key, index, nil

	}
	return "", nil, 0, walleterrors.ErrAddressNotFound
}

func (m Manager) ListEntropyFilesInStandardDir() ([]string, error) {

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

func (m *Manager) GetEntropyStoreManager(entropyStore string) (*entropystore.Manager, error) {
	absPath := entropyStore
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(m.config.DataDir, entropyStore)
	}
	if manager, ok := m.entropyStoreManager[absPath]; ok {
		return manager, nil
	}
	return nil, walleterrors.ErrStoreNotFound
}

// if your entropyStore file is not in the standard dir you can add it so we can index it
func (m *Manager) AddEntropyStore(entropyStore string) error {
	absPath := entropyStore
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(m.config.DataDir, entropyStore)
	}

	mayValid, addr, e := entropystore.IsMayValidEntropystoreFile(absPath)
	if e != nil {
		return e
	}
	if !mayValid {
		return errors.New("not valid entropy store file")
	}
	if _, ok := m.entropyStoreManager[absPath]; ok {
		return nil
	}
	m.entropyStoreManager[absPath] = entropystore.NewManager(absPath, *addr, m.config.MaxSearchIndex)
	m.entropyStoreManager[absPath].SetLockEventListener(func(event entropystore.UnlockEvent) {
		for _, lis := range m.unlockChangedLis {
			if lis != nil {
				lis(event)
			}
		}
	})
	return nil
}

func (m *Manager) RemoveEntropyStore(entropyStore string) {
	absPath := entropyStore
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(m.config.DataDir, entropyStore)
	}

	manager, ok := m.entropyStoreManager[absPath]
	if ok {
		manager.Lock()
		delete(m.entropyStoreManager, entropyStore)
	}
}

func (m *Manager) RecoverEntropyStoreFromMnemonic(mnemonic string, password string) (em *entropystore.Manager, err error) {
	sm, e := entropystore.StoreNewEntropy(m.config.DataDir, mnemonic, password, entropystore.DefaultMaxIndex)
	if e != nil {
		return nil, e
	}
	m.entropyStoreManager[sm.GetEntropyStoreFile()] = sm
	sm.SetLockEventListener(func(event entropystore.UnlockEvent) {
		for _, lis := range m.unlockChangedLis {
			if lis != nil {
				lis(event)
			}
		}
	})
	return sm, nil
}

func (m *Manager) NewMnemonicAndEntropyStore(password string) (mnemonic string, em *entropystore.Manager, err error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", nil, nil
	}
	mnemonic, err = bip39.NewMnemonic(entropy)
	if err != nil {
		return "", nil, nil
	}

	em, e := m.RecoverEntropyStoreFromMnemonic(mnemonic, password)
	if e != nil {
		return "", nil, e
	}
	return mnemonic, em, nil
}

func (m Manager) GetDataDir() string {
	return m.config.DataDir
}

func (m *Manager) Start() {
	m.entropyStoreManager = make(map[string]*entropystore.Manager)
	files, e := m.ListEntropyFilesInStandardDir()
	if e != nil {
		m.log.Error("wallet start err", "err", e)
	}
	for _, entropyStore := range files {
		if e = m.AddEntropyStore(entropyStore); e != nil {
			m.log.Error("wallet start AddEntropyStore", "err", e)
		}
	}
}

func (m *Manager) Stop() {
	for _, em := range m.entropyStoreManager {
		em.Lock()
		em.RemoveUnlockChangeChannel()
	}
	m.entropyStoreManager = nil
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
