package keystore

import (
	"github.com/pborman/uuid"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"sync"
)

// Manage keys from various wallet in here we will cache account

type Manager struct {
	ks           keyStore
	kc           *KeyConfig
	unlockedAddr map[types.Address]*unlocked
	mutex        sync.RWMutex
	isInited     bool
}

type unlocked struct {
	*Key
	abort chan struct{}
}

func NewManager(kcc *KeyConfig) *Manager {
	kp := Manager{ks: KeyStorePassphrase{kcc.KeyStoreDir}, kc: kcc}
	return &kp
}

func (kp *Manager) init() {
	if kp.isInited {
		return
	}
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	kp.unlockedAddr = make(map[types.Address]*unlocked)

	kp.isInited = true

}

func (kp *Manager) StoreNewKey(pwd string) (*Key, types.Address, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, types.Address{}, err
	}
	key := newKeyFromEd25519(&priv)

	if err := kp.ks.StoreKey(key, pwd); err != nil {
		return nil, types.Address{}, err
	}
	return key, key.Address, err
}

func (kp *Manager) ExtractKey(a types.Address, pwd string) (types.Address, *Key, error) {
	key, err := kp.ks.ExtractKey(a, pwd)
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
