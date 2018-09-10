package keystore

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"sync"
	"time"
)

const (
	Locked   = "Locked"
	UnLocked = "Unlocked"
)

type UnlockEvent struct {
	Address types.Address
	event   string // "Unlocked Locked "
}

func (ue UnlockEvent) String() string {
	return ue.Address.Hex() + " " + ue.event
}

func (ue UnlockEvent) Unlocked() bool {
	return ue.event == UnLocked
}

// Manage keys from various wallet in here we will cache account
// Manager is a keystore wallet and an interface
type Manager struct {
	ks          keyStorePassphrase
	KeyStoreDir string
	kc          *keyCache
	kcChanged   chan struct{}
	unlocked    map[types.Address]*unlocked
	mutex       sync.RWMutex
	isInited    bool

	//unlockChanged      map[int]chan<- UnlockEvent
	unlockChangedLis   map[int]func(event UnlockEvent)
	unlockChangedIndex int
	log                log15.Logger
}

type unlocked struct {
	*Key
	breaker chan struct{}
}

func NewManager(dir string) *Manager {
	kp := Manager{ks: keyStorePassphrase{dir}, KeyStoreDir: dir, log: log15.New("module", "wallet/keystore/Manager")}
	return &kp
}

func (km *Manager) Init() {
	if km.isInited {
		return
	}
	km.mutex.Lock()
	defer km.mutex.Unlock()
	km.kc, km.kcChanged = newKeyCache(km.KeyStoreDir)
	km.unlocked = make(map[types.Address]*unlocked)
	km.unlockChangedLis = make(map[int]func(event UnlockEvent))
	km.unlockChangedIndex = 100
	km.isInited = true
}

func (km *Manager) AddLockEventListener(lis func(event UnlockEvent)) int {
	km.log.Info("AddLockEventListener")
	km.mutex.Lock()
	defer km.mutex.Unlock()

	km.unlockChangedIndex++
	km.log.Info("AddLockEventListener", "id", km.unlockChangedIndex)
	km.unlockChangedLis[km.unlockChangedIndex] = lis

	return km.unlockChangedIndex
}

func (km *Manager) RemoveUnlockChangeChannel(id int) {
	km.log.Info("RemoveUnlockChangeChannel", "id", id)
	km.mutex.Lock()
	defer km.mutex.Unlock()
	delete(km.unlockChangedLis, id)
}

func (km Manager) Status() (map[types.Address]string, error) {
	m := make(map[types.Address]string)
	km.kc.ListAllAddress().Each(func(v interface{}) bool {
		a := v.(types.Address)
		if _, ok := km.unlocked[a]; ok {
			m[a] = UnLocked
		} else {
			m[a] = Locked
		}
		return false
	})
	return m, nil
}

func (km *Manager) IsUnLocked(addr types.Address) bool {
	km.mutex.Lock()
	_, exist := km.unlocked[addr]
	km.mutex.Unlock()
	return exist
}

// if the timeout is <=0 we will keep the unlock state until the program exit
func (km *Manager) Unlock(addr types.Address, passphrase string, timeout time.Duration) error {
	key, err := km.ks.ExtractKey(addr, passphrase)
	if err != nil {
		return err
	}
	km.mutex.Lock()
	defer km.mutex.Unlock()
	u, exist := km.unlocked[addr]
	if exist {
		// if the address was unlocked
		return walleterrors.ErrAlreadyLocked
	}
	if timeout > 0 {
		u = &unlocked{Key: key, breaker: make(chan struct{})}
		go km.expire(key.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	km.unlocked[key.Address] = u
	for _, f := range km.unlockChangedLis {
		f(UnlockEvent{Address: addr, event: UnLocked})
	}
	return nil
}

func (km *Manager) Lock(addr types.Address) error {
	km.mutex.Lock()
	if unl, found := km.unlocked[addr]; found {
		km.mutex.Unlock()
		km.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		km.mutex.Unlock()
	}
	return nil
}

func (km *Manager) expire(addr types.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.breaker:
	case <-t.C:
		km.mutex.Lock()
		if km.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(km.unlocked, addr)

			for _, v := range km.unlockChangedLis {
				v(UnlockEvent{
					Address: addr,
					event:   Locked,
				})
			}

		}
		km.mutex.Unlock()
	}
}

func (km *Manager) Addresses() []types.Address {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	addrs := km.kc.ListAllAddress()
	if addrs.Cardinality() == 0 {
		return []types.Address{}
	}

	result := make([]types.Address, addrs.Cardinality())
	i := 0
	for v := range addrs.Iter() {
		result[i] = v.(types.Address)
		i++
	}
	return result
}

func (km *Manager) SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error) {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	unlockedKey, found := km.unlocked[a]
	if !found {
		return nil, nil, walleterrors.ErrLocked
	}
	return unlockedKey.Sign(data)
}

func (km *Manager) SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error) {
	_, err = km.Find(a)
	if err != nil {
		return nil, nil, err
	}
	key, err := km.ExtractKey(a, passphrase)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if key != nil && key.PrivateKey != nil {
			key.PrivateKey.Clear()
			passphrase = ""
		}
	}()

	return key.Sign(data)
}

func (km *Manager) Find(a types.Address) (string, error) {
	km.kc.maybeReload()
	km.mutex.Lock()
	exist := km.kc.cacheAddr.Contains(a)
	km.mutex.Unlock()
	if exist {
		return fullKeyFileName(km.KeyStoreDir, a), nil
	} else {
		return "", walleterrors.ErrNotFind
	}
}

func (km *Manager) ReloadAndFixAddressFile() {
	km.kc.refreshAndFixAddressFile()
}

func (km *Manager) StoreNewKey(pwd string) (*Key, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	key := newKeyFromEd25519(&priv)

	if err := km.ks.StoreKey(key, pwd); err != nil {
		return nil, err
	}

	km.kc.add(key.Address)
	km.Unlock(key.Address, pwd, 0)

	return key, nil
}

func (km *Manager) ImportPriv(hexPrikey, newpwd string) (*Key, error) {
	priv, err := ed25519.HexToPrivateKey(hexPrikey)
	if err != nil {
		return nil, err
	}
	if !ed25519.IsValidPrivateKey(priv) {
		return nil, walleterrors.ErrInvalidPrikey
	}
	key := newKeyFromEd25519(&priv)

	if err := km.ks.StoreKey(key, newpwd); err != nil {
		return nil, err
	}
	return key, nil

}

func (km *Manager) ExportPriv(hexaddr, pwd string) (string, error) {
	addr, err := types.HexToAddress(hexaddr)
	if err != nil {
		return "", err
	}
	key, err := km.ExtractKey(addr, pwd)
	if err != nil {
		return "", err
	}

	return key.PrivateKey.Hex(), nil
}

func (km *Manager) ImportKeystore(keyjson, originPwd, newPwd string) (*Key, error) {
	key, err := DecryptKey([]byte(keyjson), originPwd)
	if err != nil {
		return nil, err
	}
	km.ks.StoreKey(key, newPwd)
	return key, nil
}

func (km *Manager) ExportKeystore(hexaddr, originPwd, newPwd string) (string, error) {
	addr, err := types.HexToAddress(hexaddr)
	if err != nil {
		return "", err
	}
	key, err := km.ExtractKey(addr, originPwd)
	if err != nil {
		return "", err
	}
	keyjson, err := EncryptKey(key, newPwd)
	if err != nil {
		return "", err
	}
	return string(keyjson), nil
}

func (km *Manager) ExtractKey(a types.Address, pwd string) (*Key, error) {
	key, err := km.ks.ExtractKey(a, pwd)
	return key, err
}

func zeroKey(priv *ed25519.PrivateKey) {
	if priv != nil {
		priv.Clear()
	}
}
