package account

import (
	"github.com/pborman/uuid"
	"go-vite/crypto/ed25519"
	"go-vite/common"
)

type Key struct {
	Id         uuid.UUID
	address    common.Address
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
