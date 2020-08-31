package wallet

import (
	"github.com/vitelabs/go-vite/common/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
)

type Account struct {
	address types.Address
	priv    ed25519.PrivateKey
}

func newAccount(address types.Address, key *derivation.Key) (*Account, error) {
	priv, err := key.PrivateKey()
	if err != nil {
		return nil, err
	}
	target := types.PrikeyToAddress(priv)
	if target != address {
		panic(errors.NotFound)
	}
	return &Account{address: address, priv: priv}, nil
}

func NewAccountFromHexKey(hexPriv string) (*Account, error) {
	key, err := ed25519.HexToPrivateKey(hexPriv)
	if err != nil {
		return nil, err
	}

	address := types.PrikeyToAddress(key)

	return &Account{
		address: address,
		priv:    key,
	}, nil
}

func (acct Account) Address() types.Address {
	return acct.address
}

func (acct Account) Sign(hash types.Hash) (signData []byte, pub ed25519.PublicKey, err error) {
	return ed25519.Sign(acct.priv, hash.Bytes()), acct.priv.PubByte(), nil
}

func (acct Account) PrivateKey() (ed25519.PrivateKey, error) {
	return acct.priv, nil
}
