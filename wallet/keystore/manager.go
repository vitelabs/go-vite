package keystore

import (
	"github.com/deckarep/golang-set"
	"github.com/pborman/uuid"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"strings"
	"sync"
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
	panic("implement me")
}

func (km *Manager) Open(passphrase string) error {
	panic("implement me")
}

func (km *Manager) ListAddress() []types.Address {
	panic("implement me")
}

func (km *Manager) SignData(a types.Address, data []byte) ([]byte, error) {
	panic("implement me")
}

func (km *Manager) SignDataWithPassphrase(a types.Address, passphrase, data []byte) ([]byte, error) {
	panic("implement me")
}

type unlocked struct {
	*Key
	abort chan struct{}
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
