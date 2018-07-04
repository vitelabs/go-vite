package keystore

import (
	"errors"
	"github.com/deckarep/golang-set"
	"github.com/pborman/uuid"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"strings"
	"sync"
)

var (
	ErrLocked  = errors.New("need password or unlock")
	ErrNotFind = errors.New("not found the give address in any file")
)

// Manage keys from various wallet in here we will cache account

type Manager struct {
	ks           keyStore
	keyConfig    *KeyConfig
	kc           *keyCache
	kcListener   chan struct{}
	unlockedAddr map[types.Address]*unlocked
	addrs        mapset.Set
	mutex        sync.RWMutex
	isInited     bool
}

type unlocked struct {
	*Key
	abort chan struct{}
}

func (km *Manager) Status() (string, error) {
	var sb strings.Builder

	km.addrs.Each(func(v interface{}) bool {
		a := v.(types.Address)
		if _, ok := km.unlockedAddr[a]; ok {
			sb.WriteString(a.Hex() + " Unlocked\n")
		} else {
			sb.WriteString(a.Hex() + " Locked\n")
		}
		return false
	})
	return sb.String(), nil

}

func (km *Manager) Close() error {
	return nil
}

func (km *Manager) Open(passphrase string) error {
	return nil
}

func (km *Manager) ListAddress() []types.Address {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	result := make([]types.Address, km.addrs.Cardinality())
	for v := range km.addrs.Iterator().C {
		result = append(result, v.(types.Address))
	}
	return result
}

func (km *Manager) SignData(a types.Address, data []byte) ([]byte, error) {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	unlockedKey, found := km.unlockedAddr[a]
	if !found {
		return nil, ErrLocked
	}
	return unlockedKey.Sign(data)
}

func (km *Manager) SignDataWithPassphrase(a types.Address, passphrase string, data []byte) ([]byte, error) {
	_, err := km.Find(a)
	if err != nil {
		return nil, err
	}
	_, key, err := km.ExtractKey(a, passphrase)
	defer key.PrivateKey.Clear()

	if err != nil {
		return nil, err
	}
	return key.Sign(data)
}

func (km *Manager) Find(a types.Address) (string, error) {
	km.kc.intervalRefresh()
	km.mutex.Lock()
	exist := km.kc.cacheAddr.Contains(a)
	km.mutex.Unlock()
	if exist {
		return fullKeyFileName(km.keyConfig.KeyStoreDir, a), nil
	} else {
		return "", ErrNotFind
	}

}

// if a keystore file name is changed it will read the file content if then content is a legal the function will fix the filename
func (km *Manager) FixAll() {
	km.kc.intervalRefresh()
}

func NewManager(kcc *KeyConfig) *Manager {
	kp := Manager{ks: KeyStorePassphrase{kcc.KeyStoreDir}, keyConfig: kcc}
	return &kp
}

func (km *Manager) init() {
	if km.isInited {
		return
	}
	km.mutex.Lock()
	defer km.mutex.Unlock()

	km.unlockedAddr = make(map[types.Address]*unlocked)
	km.kc, km.kcListener = newKeyCache(km.keyConfig.KeyStoreDir)

	km.addrs = km.kc.ListAllAddress()

	km.isInited = true

}

func (km *Manager) StoreNewKey(pwd string) (*Key, types.Address, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, types.Address{}, err
	}
	key := newKeyFromEd25519(&priv)

	if err := km.ks.StoreKey(key, pwd); err != nil {
		return nil, types.Address{}, err
	}
	return key, key.Address, err
}

func (km *Manager) ExtractKey(a types.Address, pwd string) (types.Address, *Key, error) {
	key, err := km.ks.ExtractKey(a, pwd)
	return a, key, err
}

func newKeyFromEd25519(priv *ed25519.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:         id,
		Address:    types.PrikeyToAddress(*priv),
		PrivateKey: priv,
	}
	return key
}
