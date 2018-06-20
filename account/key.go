package account

import (
	"github.com/pborman/uuid"
	"go-vite/crypto/ed25519"
	"go-vite/common"
)

const (
	version = 1
)
type keyStore interface {
	// Returns the key associated with the given address , using the given password to recover it from a file.
	ExtractKey(address common.Address, password string) (*Key, error)

	StoreKey(k *Key, password string) error
}

type Key struct {
	Id         uuid.UUID
	Address    common.Address
	PrivateKey *ed25519.PrivateKey
}

type encryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version int     `json:"version"`
}

type cryptoJSON struct {
	Cipher     string `json:"cipher"`
	CipherText string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	KDF        string `json:"kdf"`
	KDFParams map[string]interface {
	} `json:"kdfparams"`
}

func newKeyFromEd25519(priv *ed25519.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:         id,
		Address:    common.PrikeyToAddress(*priv),
		PrivateKey: priv,
	}
	return key
}

func newKey() (*Key, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return newKeyFromEd25519(&priv), nil
}

func storeNewKey(ks keyStore, pwd string) (*Key, common.Address, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, common.InvalidAddress, err
	}
	key := newKeyFromEd25519(&priv)

	if err := ks.StoreKey(key, pwd); err != nil {
		return nil, common.InvalidAddress, err
	}
	return key, key.Address, err
}
