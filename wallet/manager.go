package wallet

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
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

func (m *Manager) Unlock(entropyStore, passphrase string) error {
	manager, e := m.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return e
	}

	return manager.Unlock(passphrase)
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

	mayValid, json, e := entropystore.IsMayValidEntropystoreFile(absPath)
	if e != nil {
		return e
	}
	if !mayValid {
		return errors.New("not valid entropy store file")
	}
	if _, ok := m.entropyStoreManager[absPath]; ok {
		return nil
	}
	m.entropyStoreManager[absPath] = entropystore.NewManager(absPath, *json.PrimaryAddress, &entropystore.Config{
		MaxSearchIndex: m.config.MaxSearchIndex,
		UseLightScrypt: m.config.UseLightScrypt,
	})
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
func (m *Manager) ExtractMnemonic(entropyStore, passphrase string) (string, error) {
	absPath := entropyStore
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(m.config.DataDir, entropyStore)
	}

	manager, ok := m.entropyStoreManager[absPath]
	if ok {
		return manager.ExtractMnemonic(passphrase)
	}
	return "", walleterrors.ErrStoreNotFound
}

func (m *Manager) RecoverEntropyStoreFromMnemonic(mnemonic, language, passphrase string, extensionWord *string) (em *entropystore.Manager, err error) {

	entropyprofile, e := entropystore.MnemonicToEntropy(mnemonic, language, extensionWord != nil, extensionWord)
	if e != nil {
		return nil, e
	}

	sm, e := entropystore.StoreNewEntropy(m.config.DataDir, passphrase, entropyprofile, &entropystore.Config{
		MaxSearchIndex: m.config.MaxSearchIndex,
		UseLightScrypt: m.config.UseLightScrypt,
	})
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

func (m *Manager) NewMnemonicAndEntropyStore(language, passphrase string, extensionWord *string, mnemonicSize *int) (mnemonic string, em *entropystore.Manager, err error) {

	mnemonic, e := entropystore.NewMnemonic(language, mnemonicSize)
	if e != nil {
		return "", nil, e
	}

	manager, e := m.RecoverEntropyStoreFromMnemonic(mnemonic, language, passphrase, extensionWord)
	if e != nil {
		return "", nil, e
	}

	return mnemonic, manager, nil
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

func (m *Manager) DeriveKey(entropyPath string, addr *types.Address, index uint32, extensionWord *string) (*derivation.Key, error) {
	manager, err := m.GetEntropyStoreManager(entropyPath)
	if err != nil {
		return nil, err
	}
	_, key, err := manager.DeriveForIndexPath(index, extensionWord)
	if err != nil {
		return nil, err
	}
	if addr != nil {
		address, err := key.Address()
		if err != nil {
			return nil, err
		}
		if *address != *addr {
			return nil, errors.New("address do not match.")
		}
	}

	return key, nil
}

func (m *Manager) MatchAddress(entropyPath string, addr types.Address, index uint32, extensionWord *string) error {
	_, e := m.DeriveKey(entropyPath, &addr, index, extensionWord)
	return e
}
