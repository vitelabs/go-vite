package entropystore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
)

const (
	Locked   = "Locked"
	UnLocked = "Unlocked"

	DefaultMaxIndex = uint32(100)
)

type UnlockEvent struct {
	EntropyStoreFile string
	PrimaryAddr      types.Address // represent which seed we use the seed`s PrimaryAddress represents the seed
	event            string        // "Unlocked Locked"
}

func (ue UnlockEvent) String() string {
	return ue.EntropyStoreFile + " " + ue.PrimaryAddr.String() + " " + ue.event
}

func (ue UnlockEvent) Unlocked() bool {
	return ue.event == UnLocked
}

type Manager struct {
	primaryAddr    types.Address
	ks             *CryptoStore
	maxSearchIndex uint32

	unlockedEntropy *EntropyProfile
	seed            []byte

	unlockChangedLis func(event UnlockEvent)

	log log15.Logger
}

func StoreNewEntropy(storeDir, passphrase string, entropyprofile *EntropyProfile, config *Config) (*Manager, error) {

	filename := FullKeyFileName(storeDir, *entropyprofile.PrimaryAddress)
	ss := NewCryptoStore(filename, config.UseLightScrypt)
	e := ss.StoreEntropy(*entropyprofile, passphrase)
	if e != nil {
		return nil, e
	}
	return NewManager(filename, *entropyprofile.PrimaryAddress, config), nil
}

func NewManager(entropyStoreFilename string, primaryAddr types.Address, config *Config) *Manager {
	return &Manager{
		primaryAddr:    primaryAddr,
		ks:             NewCryptoStore(entropyStoreFilename, config.UseLightScrypt),
		maxSearchIndex: config.MaxSearchIndex,

		log: log15.New("module", "wallet/keystore/Manager"),
	}
}

func (km *Manager) IsAddrUnlocked(addr types.Address, extensionWord *string) bool {
	_, _, e := FindAddrFromEntropy(*km.unlockedEntropy, addr, extensionWord, km.maxSearchIndex)
	if e != nil {
		return false
	}
	return true
}

func (km *Manager) FindAddr(addr types.Address, extensionWord *string) (key *derivation.Key, index uint32, e error) {
	if !km.IsUnlocked() {
		return nil, 0, walleterrors.ErrLocked
	}
	return FindAddrFromEntropy(*km.unlockedEntropy, addr, extensionWord, km.maxSearchIndex)
}

func (km *Manager) IsUnlocked() bool {
	return km.unlockedEntropy != nil
}

func (km *Manager) ListAddress(from, to uint32, extensionWord *string) ([]types.Address, error) {
	if from > to {
		return nil, errors.New("from > to")
	}
	if km.unlockedEntropy == nil {
		return nil, walleterrors.ErrLocked
	}
	addr := make([]types.Address, to-from)
	addrIndex := 0
	for i := from; i < to; i++ {
		_, key, e := km.DeriveForIndexPath(i, extensionWord)
		if e != nil {
			return nil, e
		}
		address, e := key.Address()
		if e != nil {
			return nil, e
		}
		addr[addrIndex] = *address
		addrIndex++
	}

	return addr, nil
}

func (km *Manager) Unlock(passphrase string) error {
	entropy, e := km.ks.ExtractEntropy(passphrase)
	if e != nil {
		return e
	}
	km.unlockedEntropy = entropy

	if km.unlockChangedLis != nil {
		km.unlockChangedLis(UnlockEvent{
			EntropyStoreFile: km.GetEntropyStoreFile(),
			PrimaryAddr:      km.primaryAddr,
			event:            UnLocked})
	}
	return nil
}

func (km *Manager) Lock() {
	km.unlockedEntropy = nil
	if km.unlockChangedLis != nil {
		km.unlockChangedLis(UnlockEvent{
			EntropyStoreFile: km.GetEntropyStoreFile(),
			PrimaryAddr:      km.primaryAddr,
			event:            Locked})
	}
}

func (km *Manager) FindAddrWithPassphrase(passphrase string, addr types.Address, extensionWord *string) (key *derivation.Key, index uint32, e error) {
	entropy, err := km.ks.ExtractEntropy(passphrase)
	if err != nil {
		return nil, 0, err
	}
	return FindAddrFromEntropy(*entropy, addr, extensionWord, km.maxSearchIndex)
}

func (km *Manager) SignData(a types.Address, data []byte, bip44index *uint32, extensionWord *string) (signedData, pubkey []byte, err error) {
	if !km.IsUnlocked() {
		return nil, nil, walleterrors.ErrLocked
	}

	if bip44index != nil {
		_, key, e := km.DeriveForIndexPath(*bip44index, extensionWord)
		if e != nil {
			return nil, nil, e
		}
		return key.SignData(data)
	}

	key, _, e := FindAddrFromEntropy(*km.unlockedEntropy, a, extensionWord, km.maxSearchIndex)
	if e != nil {
		return nil, nil, walleterrors.ErrAddressNotFound
	}
	return key.SignData(data)
}

func (km *Manager) SignDataWithPassphrase(addr types.Address, passphrase string, data []byte, bip44index *uint32, extensionWord *string) (signedData, pubkey []byte, err error) {

	entropyProfile, err := km.ks.ExtractEntropy(passphrase)
	if err != nil {
		return nil, nil, err
	}

	if bip44index != nil {
		seed, err := km.unlockedEntropy.GetSeed(extensionWord)
		if err != nil {
			return nil, nil, err
		}

		key, e := derivation.DeriveWithIndex(*bip44index, seed)
		if e != nil {
			return nil, nil, e
		}

		return key.SignData(data)
	}

	key, _, e := FindAddrFromEntropy(*entropyProfile, addr, extensionWord, km.maxSearchIndex)
	if e != nil {
		return nil, nil, e
	}

	return key.SignData(data)
}

func (km *Manager) DeriveForFullPath(path string, extensionWord *string) (fpath string, key *derivation.Key, err error) {
	if !km.IsUnlocked() {
		return "", nil, walleterrors.ErrLocked
	}

	seed, err := km.unlockedEntropy.GetSeed(extensionWord)
	if err != nil {
		return "", nil, err
	}

	key, e := derivation.DeriveForPath(path, seed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPath(index uint32, extensionWord *string) (path string, key *derivation.Key, err error) {
	return km.DeriveForFullPath(fmt.Sprintf(derivation.ViteAccountPathFormat, index), extensionWord)
}

func (km *Manager) DeriveForFullPathWithPassphrase(path, passphrase string, extensionWord *string) (fpath string, key *derivation.Key, err error) {
	seed, err := km.unlockedEntropy.GetSeed(extensionWord)
	if err != nil {
		return "", nil, err
	}

	key, e := derivation.DeriveForPath(path, seed)
	if e != nil {
		return "", nil, e
	}

	return path, key, nil
}

func (km *Manager) DeriveForIndexPathWithPassphrase(index uint32, passphrase string, extensionWord *string) (path string, key *derivation.Key, err error) {
	return km.DeriveForFullPathWithPassphrase(fmt.Sprintf(derivation.ViteAccountPathFormat, index), passphrase, extensionWord)
}

func (km Manager) GetPrimaryAddr() (primaryAddr types.Address) {
	return km.primaryAddr
}

func (km Manager) GetEntropyStoreFile() string {
	return km.ks.EntropyStoreFilename
}

func (km *Manager) SetLockEventListener(lis func(event UnlockEvent)) {
	km.unlockChangedLis = lis
}

func (km *Manager) RemoveUnlockChangeChannel() {
	km.unlockChangedLis = nil
}
