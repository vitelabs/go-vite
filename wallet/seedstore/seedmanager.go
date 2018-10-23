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

	defaultMaxIndex = uint32(100)
)

type UnlockEvent struct {
	PrimaryAddr types.Address // represent which seed
	Address     types.Address // represent which addr in specific addr
	event       string        // "Unlocked Locked "
}

func (ue UnlockEvent) String() string {
	return "primary" + ue.PrimaryAddr.String() + " " + ue.Address.Hex() + " " + ue.event
}

func (ue UnlockEvent) Unlocked() bool {
	return ue.event == UnLocked
}

// Manage keys from various wallet in here we will cache account
// Manager is a keystore wallet and an interface
type Manager struct {
	ks SeedStorePassphrase

	unlockedAddr   map[types.Address]*derivation.Key
	unlockedSeed   []byte
	indexToAddress map[uint32]types.Address

	mutex sync.RWMutex

	unlockChangedLis   map[int]func(event UnlockEvent)
	unlockChangedIndex int
	log                log15.Logger
}

func NewManager(seedStoreFilename string) *Manager {
	return &Manager{
		ks:                 SeedStorePassphrase{seedStoreFilename},
		unlockedAddr:       make(map[types.Address]*derivation.Key),
		indexToAddress:     make(map[uint32]types.Address),
		unlockChangedLis:   make(map[int]func(event UnlockEvent)),
		unlockChangedIndex: 100,

		log: log15.New("module", "wallet/keystore/Manager"),
	}
}

func (km Manager) SeedStoreFile() string {
	return km.ks.SeedStoreFilename
}

func (km *Manager) IsAddrUnlocked(addr types.Address) bool {
	km.mutex.Lock()
	_, exist := km.unlockedAddr[addr]
	km.mutex.Unlock()
	return exist
}

func (km *Manager) IsUnlocked() bool {
	return km.unlockedSeed != nil
}

func (km *Manager) UnlockSeed(password string) error {
	seed, e := km.ks.ExtractSeed(password)
	if e != nil {
		return e
	}
	km.unlockedSeed = seed
	return nil
}

func (km *Manager) LockSeed() error {
	km.unlockedSeed = nil

	for k, v := range km.unlockedAddr {

	}

	km.mutex.Lock()
	km.unlockedAddr = make(map[types.Address]*derivation.Key)
	km.mutex.Unlock()

	return nil
}

func (km *Manager) UnlockAddress(addr types.Address, password string, maxSearchIndex *uint32) error {
	km.mutex.RLock()
	_, exist := km.unlockedAddr[addr]
	if exist {
		km.mutex.RUnlock()
		return nil
	}
	km.mutex.RUnlock()

	seed, err := km.ks.ExtractSeed(password)
	if err != nil {
		return err
	}

	key, index, e := FindAddrFromSeed(seed, addr, maxSearchIndex)
	if e != nil {
		return e
	}
	km.mutex.Lock()
	km.unlockedAddr[addr] = key
	km.indexToAddress[index] = addr
	km.mutex.Unlock()

	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{Address: addr, event: UnLocked})
	}

	return walleterrors.ErrNotFind
}

func (km *Manager) FindAddrInUnlock(addr types.Address, maxSearchIndex *uint32) (*derivation.Key, uint32, error) {
	seed, err := km.ks.ExtractSeed(password)
	if err != nil {
		return nil, 0, err
	}
	return FindAddrFromSeed(seed, addr, maxSearchIndex)
}

func (km *Manager) FindAddr(password string, addr types.Address, maxSearchIndex *uint32) (*derivation.Key, uint32, error) {
	seed, err := km.ks.ExtractSeed(password)
	if err != nil {
		return nil, 0, err
	}
	return FindAddrFromSeed(seed, addr, maxSearchIndex)
}

func (km *Manager) UnlockAddressWithIndex(index uint32, passphrase string) error {
	km.mutex.RLock()
	if addr, exist := km.indexToAddress[index]; exist {
		if _, exist = km.unlockedAddr[addr]; exist {
			km.mutex.RUnlock()
			return nil
		}
	}
	km.mutex.RUnlock()

	seed, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return err
	}

	key, e := derivation.DeriveWithIndex(index, seed)
	if e != nil {
		return e

	}
	addr, e := key.Address()
	if e != nil {
		return e

	}
	km.mutex.Lock()
	km.indexToAddress[index] = *addr
	km.unlockedAddr[*addr] = key
	km.mutex.Unlock()

	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{Address: *addr, event: UnLocked})
	}
	return nil
}

func (km *Manager) LockAddress(addr types.Address) error {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	if _, found := km.unlockedAddr[addr]; found {
		delete(km.unlockedAddr, addr)
		return nil
	}
	return nil
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

func (km *Manager) SignDataWithPassphrase(addr types.Address, passphrase string, data []byte, maxSearchIndex *uint32) (signedData, pubkey []byte, err error) {
	seed, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return nil, nil, err
	}
	key, _, e := FindAddrFromSeed(seed, addr, maxSearchIndex)
	if e != nil {
		return nil, nil, e
	}

	return key.SignData(data)
}

func (km *Manager) DeriveForFullPath(path, password string) {

}

func (km *Manager) DeriveForIndexPath(index uint32, password string) (string, *derivation.Key, error) {
	seed, err := km.ks.ExtractSeed(password)
	if err != nil {
		return "", nil, err
	}
	path := fmt.Sprintf(derivation.ViteAccountPathFormat, index)

	key, e := derivation.DeriveForPath(path, seed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func StoreNewSeed(seedDir string, seed []byte, pwd string) (*Manager, error) {
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
	return NewManager(filename), nil
}

func FindAddrFromSeed(seed []byte, addr types.Address, maxSearchIndex *uint32) (*derivation.Key, uint32, error) {
	realSearchIndex := defaultMaxIndex
	if maxSearchIndex != nil {
		realSearchIndex = *maxSearchIndex
	}
	for i := uint32(0); i < realSearchIndex; i++ {
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
