package account

import (
	"github.com/pborman/uuid"
	"go-vite/crypto/ed25519"
	"go-vite/common"
	"time"
	"fmt"
	"encoding/hex"
)

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
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
	Version string     `json:"version"`
}

type cryptoJSON struct {
	Cipher     string                 `json:"cipher"`
	CipherText string                 `json:"ciphertext"`
	IV         string                 `json:"iv"`
	KDF        string                 `json:"kdf"`
	KDFParams  map[string]interface{} `json:"kdfparams"`
	MAC        string                 `json:"mac"`
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
	key, err := newKey()
	if err != nil {
		return nil, common.InvalidAddress, err
	}

	if err := ks.StoreKey("~/" +keyFileName(key.Address) , key, pwd); err != nil {
		return nil, common.InvalidAddress, err
	}
	return key, key.Address, err
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}