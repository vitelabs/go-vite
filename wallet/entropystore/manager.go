package entropystore

import (
	"fmt"
	"github.com/tyler-smith/go-bip39"
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
	event       string        // "Unlocked Locked"
}

func (ue UnlockEvent) String() string {
	return ue.PrimaryAddr.String() + " " + ue.event
}

func (ue UnlockEvent) Unlocked() bool {
	return ue.event == UnLocked
}

type Manager struct {
	primaryAddr    types.Address
	ks             CryptoStore
	maxSearchIndex uint32

	unlockedSeed    []byte
	unlockedEntropy []byte
	mutex           sync.Mutex

	unlockChangedLis   map[int]func(event UnlockEvent)
	unlockChangedIndex int
	log                log15.Logger
}

func NewManager(entropyStoreFilename string, primaryAddr types.Address, maxSearchIndex uint32) *Manager {
	return &Manager{
		primaryAddr:        primaryAddr,
		ks:                 CryptoStore{entropyStoreFilename},
		unlockChangedLis:   make(map[int]func(event UnlockEvent)),
		maxSearchIndex:     maxSearchIndex,
		unlockChangedIndex: 100,

		log: log15.New("module", "wallet/keystore/Manager"),
	}
}

func (km *Manager) IsAddrUnlocked(addr types.Address) bool {
	if !km.IsUnlocked() {
		return false
	}
	_, _, e := FindAddrFromSeed(km.unlockedSeed, addr, km.maxSearchIndex)
	if e != nil {
		return false
	}
	return true
}

func (km *Manager) IsUnlocked() bool {
	return km.unlockedSeed != nil
}

func (km *Manager) ListAddress(maxIndex uint32) ([]types.Address, error) {
	if km.unlockedSeed == nil {
		return nil, walleterrors.ErrLocked
	}
	addr := make([]types.Address, maxIndex)
	for i := uint32(0); i < maxIndex; i++ {
		_, key, e := km.DeriveForIndexPath(i)
		if e != nil {
			return nil, e
		}
		address, e := key.Address()
		if e != nil {
			return nil, e
		}
		addr[i] = *address
	}

	return addr, nil
}

func (km *Manager) Unlock(password string) error {
	seed, entropy, e := km.ks.ExtractSeed(password)
	if e != nil {
		return e
	}
	km.unlockedSeed = seed
	km.unlockedEntropy = entropy

	pAddr, e := derivation.GetPrimaryAddress(seed)
	if e != nil {
		return e
	}
	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{PrimaryAddr: *pAddr, event: UnLocked})
	}
	return nil
}

func (km *Manager) Lock() {

	km.unlockedSeed = nil

	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{PrimaryAddr: km.primaryAddr, event: Locked})
	}
}

func (km *Manager) FindAddrWithPassword(password string, addr types.Address) (key *derivation.Key, index uint32, e error) {
	seed, _, err := km.ks.ExtractSeed(password)
	if err != nil {
		return nil, 0, err
	}
	return FindAddrFromSeed(seed, addr, km.maxSearchIndex)
}

func (km *Manager) FindAddr(addr types.Address) (key *derivation.Key, index uint32, e error) {
	if !km.IsUnlocked() {
		return nil, 0, walleterrors.ErrLocked
	}

	return FindAddrFromSeed(km.unlockedSeed, addr, km.maxSearchIndex)
}

func (km *Manager) SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error) {
	key, _, e := FindAddrFromSeed(data, a, km.maxSearchIndex)
	if e != nil {
		return nil, nil, walleterrors.ErrLocked
	}
	return key.SignData(data)
}

func (km *Manager) SignDataWithPassphrase(addr types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error) {
	seed, _, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return nil, nil, err
	}
	key, _, e := FindAddrFromSeed(seed, addr, km.maxSearchIndex)
	if e != nil {
		return nil, nil, e
	}

	return key.SignData(data)
}

func (km *Manager) DeriveForFullPath(path string) (fpath string, key *derivation.Key, err error) {
	if km.unlockedSeed == nil {
		return "", nil, walleterrors.ErrLocked
	}

	key, e := derivation.DeriveForPath(path, km.unlockedSeed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPath(index uint32) (path string, key *derivation.Key, err error) {
	return km.DeriveForFullPath(fmt.Sprintf(derivation.ViteAccountPathFormat, index))
}

func (km *Manager) DeriveForFullPathWithPassphrase(path, passphrase string) (fpath string, key *derivation.Key, err error) {
	seed, _, err := km.ks.ExtractSeed(passphrase)
	if err != nil {
		return "", nil, err
	}

	key, e := derivation.DeriveForPath(path, seed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPathWithPassphrase(index uint32, passphrase string) (path string, key *derivation.Key, err error) {
	return km.DeriveForFullPathWithPassphrase(fmt.Sprintf(derivation.ViteAccountPathFormat, index), passphrase)
}

func (km Manager) GetPrimaryAddr() (primaryAddr types.Address) {
	return km.primaryAddr
}

func (km Manager) GetEntropyStoreFile() string {
	return km.ks.EntropyStoreFilename
}

func StoreNewEntropy(storeDir string, mnemonic string, pwd string, maxSearchIndex uint32) (*Manager, error) {
	entropy, e := bip39.EntropyFromMnemonic(mnemonic)
	if e != nil {
		return nil, e
	}

	primaryAddress, e := MnemonicToPrimaryAddr(mnemonic)

	filename := FullKeyFileName(storeDir, *primaryAddress)
	ss := CryptoStore{filename}
	e = ss.StoreEntropy(entropy, *primaryAddress, pwd)
	if e != nil {
		return nil, e
	}
	return NewManager(filename, *primaryAddress, maxSearchIndex), nil
}

func MnemonicToPrimaryAddr(mnemonic string) (primaryAddress *types.Address, e error) {
	seed := bip39.NewSeed(mnemonic, "")
	primaryAddress, e = derivation.GetPrimaryAddress(seed)
	if e != nil {
		return nil, e
	}
	return primaryAddress, nil
}

// it is very fast(in my mac 2.8GHZ intel cpu 1Million search cost 728ms) so we dont need cache the relation
func FindAddrFromSeed(seed []byte, addr types.Address, maxSearchIndex uint32) (key *derivation.Key, index uint32, e error) {
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
