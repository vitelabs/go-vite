package seedstore

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"sync"
)

const (
	Locked   = "Locked"
	UnLocked = "Unlocked"

	DefaultMaxIndex = uint32(100)
)

type UnlockEvent struct {
	PrimaryAddr types.Address // represent which seed we use the seed`s PrimaryAddress represents the seed
	event       string        // "Unlocked Locked "
}

func (ue UnlockEvent) String() string {
	return ue.PrimaryAddr.String() + " " + ue.event
}

func (ue UnlockEvent) Unlocked() bool {
	return ue.event == UnLocked
}

type Manager struct {
	ks             SeedStorePassphrase
	maxSearchIndex uint32

	unlockedAddr map[types.Address]*derivation.Key
	unlockedSeed []byte

	mutex sync.RWMutex

	unlockChangedLis   map[int]func(event UnlockEvent)
	unlockChangedIndex int
	log                log15.Logger
}

func NewManager(seedStoreFilename string, maxSearchIndex uint32) *Manager {
	return &Manager{
		ks:                 SeedStorePassphrase{seedStoreFilename},
		unlockedAddr:       make(map[types.Address]*derivation.Key),
		maxSearchIndex:     maxSearchIndex,
		unlockChangedLis:   make(map[int]func(event UnlockEvent)),
		unlockChangedIndex: 100,

		log: log15.New("module", "wallet/keystore/Manager"),
	}
}

func (km Manager) SeedStoreFile() string {
	return km.ks.SeedStoreFilename
}

func (km *Manager) IsAddrUnlocked(addr types.Address) bool {
	if !km.IsSeedUnlocked() {
		return false
	}

	km.mutex.RLock()
	_, exist := km.unlockedAddr[addr]
	if exist {
		km.mutex.RUnlock()
		return true
	}
	km.mutex.RUnlock()

	key, _, e := FindAddrFromSeed(km.unlockedSeed, addr, km.maxSearchIndex)
	if e != nil {
		return false
	}

	km.mutex.Lock()
	km.unlockedAddr[addr] = key
	km.mutex.Unlock()

	return true
}

func (km *Manager) IsSeedUnlocked() bool {
	return km.unlockedSeed != nil
}

func (km *Manager) ListAddress(maxIndex uint32) ([]*types.Address, error) {
	if km.unlockedSeed == nil {
		return nil, walleterrors.ErrLocked
	}
	addr := make([]*types.Address, maxIndex)
	for i := uint32(0); i < maxIndex; i++ {
		_, key, e := km.DeriveForIndexPath(i)
		if e != nil {
			return nil, e
		}
		address, e := key.Address()
		if e != nil {
			return nil, e
		}
		addr[i] = address
	}

	return addr, nil
}

func (km *Manager) UnlockSeed(password string) error {
	seed, e := km.ks.ExtractSeed(password)
	if e != nil {
		return e
	}
	km.unlockedSeed = seed

	pAddr, e := derivation.GetPrimaryAddress(seed)
	if e != nil {
		return e
	}
	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{PrimaryAddr: *pAddr, event: UnLocked})
	}
	return nil
}

func (km *Manager) LockSeed() error {
	pAddr, e := derivation.GetPrimaryAddress(km.unlockedSeed)
	if e != nil {
		return e
	}
	km.unlockedSeed = nil

	km.mutex.Lock()
	km.unlockedAddr = make(map[types.Address]*derivation.Key)
	km.mutex.Unlock()

	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{PrimaryAddr: *pAddr, event: Locked})
	}
	return nil
}

func (km *Manager) FindAddr(password string, addr types.Address) (*derivation.Key, uint32, error) {
	seed, err := km.ks.ExtractSeed(password)
	if err != nil {
		return nil, 0, err
	}
	return FindAddrFromSeed(seed, addr, km.maxSearchIndex)
}

func (km *Manager) SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error) {
	km.mutex.RLock()
	key, found := km.unlockedAddr[a]
	km.mutex.RUnlock()
	if !found {
		return nil, nil, walleterrors.ErrLocked
	}
	return key.SignData(data)
}

func (km *Manager) SignDataWithPassphrase(addr types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error) {
	seed, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return nil, nil, err
	}
	key, _, e := FindAddrFromSeed(seed, addr, km.maxSearchIndex)
	if e != nil {
		return nil, nil, e
	}

	return key.SignData(data)
}

func (km *Manager) DeriveForFullPath(path string) (string, *derivation.Key, error) {
	if km.unlockedSeed == nil {
		return "", nil, walleterrors.ErrLocked
	}

	key, e := derivation.DeriveForPath(path, km.unlockedSeed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPath(index uint32) (string, *derivation.Key, error) {
	return km.DeriveForFullPath(fmt.Sprintf(derivation.ViteAccountPathFormat, index))
}

func (km *Manager) DeriveForFullPathWithPassphrase(path, passphrase string) (string, *derivation.Key, error) {
	seed, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return "", nil, err
	}

	key, e := derivation.DeriveForPath(path, seed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPathWithPassphrase(index uint32, passphrase string) (string, *derivation.Key, error) {
	return km.DeriveForFullPathWithPassphrase(fmt.Sprintf(derivation.ViteAccountPathFormat, index), passphrase)
}

func StoreNewSeed(seedDir string, seed []byte, pwd string, maxSearchIndex uint32) (*Manager, error) {
	primaryAddress, e := derivation.GetPrimaryAddress(seed)
	if e != nil {
		return nil, e
	}

	filename := FullKeyFileName(seedDir, *primaryAddress)
	ss := SeedStorePassphrase{filename}
	e = ss.StoreSeed(seed, pwd)
	if e != nil {
		return nil, e
	}
	return NewManager(filename, maxSearchIndex), nil
}

func FindAddrFromSeed(seed []byte, addr types.Address, maxSearchIndex uint32) (*derivation.Key, uint32, error) {
	for i := uint32(0); i < maxSearchIndex; i++ {
		key, e := derivation.DeriveWithIndex(i, seed)
		if e != nil {
			return nil, 0, e
		}
		genAddr, e := key.Address()
		if addr == *genAddr {
			return key, i, nil
		}
	}
	return nil, 0, walleterrors.ErrNotFind
}

func (km *Manager) AddLockEventListener(lis func(event UnlockEvent)) int {
	km.mutex.Lock()
	defer km.mutex.Unlock()

	km.unlockChangedIndex++
	km.unlockChangedLis[km.unlockChangedIndex] = lis

	return km.unlockChangedIndex
}

func (km *Manager) RemoveUnlockChangeChannel(id int) {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	delete(km.unlockChangedLis, id)
}
