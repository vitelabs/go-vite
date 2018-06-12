package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"go-vite/crypto/ed25519"
)



func GenerateKey() (publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, err error) {
	return ed25519.GenerateKey(rand.Reader)
}

func ToPub(hexPub string) (ed25519.PublicKey) {
	b, err := hex.DecodeString(hexPub)
	if err != nil {
		return nil
	}
	return b
}

func recoverPubKey(hexPrikey string) (ed25519.PublicKey) {
	priv , err := hex.DecodeString(hexPrikey)
	if err != nil {
		return nil
	}
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, priv[32:])
	return publicKey

}



