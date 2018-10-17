package mobile

import (
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

func Hash256(data []byte) []byte {
	return crypto.Hash256(data)
}

type Ed25519KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

func GenerateEd25519KeyPair(seed []byte) (p *Ed25519KeyPair, _ error) {
	var s [32]byte
	copy(s[:], seed[:])
	publicKey, privateKey, err := ed25519.GenerateKeyFromD(s)
	if err != nil {
		return nil, err
	}
	return &Ed25519KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}
