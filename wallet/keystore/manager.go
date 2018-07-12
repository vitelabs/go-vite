package keystore

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"strings"
	"sync"
	"time"
)

var (
	ErrLocked        = errors.New("need password or unlock")
	ErrNotFind       = errors.New("not found the give address in any file")
	ErrInvalidPrikey = errors.New("invalid prikey")
)

// Manage keys from various wallet in here we will cache account
// Manager is a keystore wallet and an interface
type Manager struct {
	ks          keyStorePassphrase
	keyStoreDir string
	kc          *keyCache
	kcChanged   chan struct{}
	unlocked    map[types.Address]*unlocked
	mutex       sync.RWMutex
	isInited    bool
}

type unlocked struct {
	*Key
	abort chan struct{}
}

func NewManager(dir string) *Manager {
	kp := Manager{ks: keyStorePassphrase{dir}, keyStoreDir: dir}
	return &kp
}

func (km *Manager) Init() {
	if km.isInited {
		return
	}
	km.mutex.Lock()
	defer km.mutex.Unlock()

	km.kc, km.kcChanged = newKeyCache(km.keyStoreDir)

	km.unlocked = make(map[types.Address]*unlocked)

	km.isInited = true
}

func (km Manager) Status() (string, error) {
	var sb strings.Builder

	km.kc.ListAllAddress().Each(func(v interface{}) bool {
		a := v.(types.Address)
		if _, ok := km.unlocked[a]; ok {
			sb.WriteString(a.Hex() + " Unlocked\n")
		} else {
			sb.WriteString(a.Hex() + " Locked\n")
		}
		return false
	})
	return sb.String(), nil
}

// if the timeout is <0 we will keep the unlock state until the program exit
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
		return fmt.Errorf("the address %v was previously unlocked", addr.String())
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go km.expire(key.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	km.unlocked[key.Address] = u
	return nil
}

func (km *Manager) expire(addr types.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
	case <-t.C:
		km.mutex.Lock()
		if km.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(km.unlocked, addr)
		}
		km.mutex.Unlock()
	}
}

func (km *Manager) Addresses() []types.Address {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	addrs := km.kc.ListAllAddress()
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
		return nil, nil, ErrLocked
	}
	return unlockedKey.Sign(data)
}

func (km *Manager) SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error) {
	_, err = km.Find(a)
	if err != nil {
		return nil, nil, err
	}
	_, key, err := km.ExtractKey(a, passphrase)
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
		return fullKeyFileName(km.keyStoreDir, a), nil
	} else {
		return "", ErrNotFind
	}
}

// if a keystore file name is changed it will read the file content if then content is a legal the function will fix the filename
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

func (km *Manager) ImportPriv(hexPrikey, newpwd string) (*Key, types.Address, error) {
	priv, err := ed25519.HexToPrivateKey(hexPrikey)
	if err != nil {
		return nil, types.Address{}, err
	}
	if !ed25519.IsValidPrivateKey(priv) {
		return nil, types.Address{}, ErrInvalidPrikey
	}
	key := newKeyFromEd25519(&priv)
	addr := types.PrikeyToAddress(priv)

	if err := km.ks.StoreKey(key, newpwd); err != nil {
		return nil, types.Address{}, err
	}
	return key, addr, nil

}

func (km *Manager) ExportPriv(hexaddr, pwd string) (string, error) {
	addr, err := types.HexToAddress(hexaddr)
	if err != nil {
		return "", err
	}
	_, key, err := km.ExtractKey(addr, pwd)
	if err != nil {
		return "", err
	}

	return key.PrivateKey.Hex(), nil
}

func (km *Manager) Import(keyjson, originPwd, newPwd string) (*Key, types.Address, error) {
	key, err := DecryptKey([]byte(keyjson), originPwd)
	if err != nil {
		return nil, types.Address{}, err
	}
	km.ks.StoreKey(key, newPwd)
	return key, key.Address, nil
}

func (km *Manager) Export(hexaddr, originPwd, newPwd string) (string, error) {
	addr, err := types.HexToAddress(hexaddr)
	if err != nil {
		return "", err
	}
	_, key, err := km.ExtractKey(addr, originPwd)
	if err != nil {
		return "", err
	}
	keyjson, err := EncryptKey(key, newPwd)
	if err != nil {
		return "", err
	}
	return string(keyjson), nil
}

func (km *Manager) ExtractKey(a types.Address, pwd string) (types.Address, *Key, error) {
	key, err := km.ks.ExtractKey(a, pwd)
	return a, key, err
}

func zeroKey(priv *ed25519.PrivateKey) {
	if priv != nil {
		priv.Clear()
	}
}
