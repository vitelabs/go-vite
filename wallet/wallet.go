package wallet

import (
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
)

func RandomMnemonic24() (types.Address, ed25519.PrivateKey, string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return types.Address{}, nil, "", err
	}
	return newMnemonic(entropy)
}

func RandomMnemonic12() (types.Address, ed25519.PrivateKey, string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return types.Address{}, nil, "", err
	}
	return newMnemonic(entropy)
}

func newMnemonic(entropy []byte) (types.Address, ed25519.PrivateKey, string, error) {
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return types.Address{}, nil, "", err
	}
	seed := bip39.NewSeed(mnemonic, "")
	key, err := derivation.DeriveWithIndex(0, seed)
	if err != nil {
		return types.Address{}, nil, "", err
	}
	address, err := key.Address()
	if err != nil {
		return types.Address{}, nil, "", err
	}
	priKeys, err := key.PrivateKey()
	if err != nil {
		return types.Address{}, nil, "", err
	}
	return *address, priKeys, mnemonic, nil
}
