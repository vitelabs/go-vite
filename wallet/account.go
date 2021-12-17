package wallet

import (
	"github.com/vitelabs/go-vite/v2/common/errors"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
	"github.com/vitelabs/go-vite/v2/interfaces"
	"github.com/vitelabs/go-vite/v2/wallet/hd-bip/derivation"
)

type Account struct {
	interfaces.Account
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

func RandomAccount() (*Account, error) {
	addr, priv, _, err := RandomMnemonic24()
	if err != nil {
		return nil, err
	}
	return &Account{
		address: addr,
		priv:    priv,
	}, nil
}

func (acct Account) Address() types.Address {
	return acct.address
}

func (acct Account) Sign(msg []byte) (signData []byte, pub ed25519.PublicKey, err error) {
	return ed25519.Sign(acct.priv, msg), acct.priv.PubByte(), nil
}

func (acct Account) Verify(pub ed25519.PublicKey, message, signdata []byte) error {
	return ed25519.VerifySig(pub, message, signdata)
}

func (acct Account) PrivateKey() (ed25519.PrivateKey, error) {
	return acct.priv, nil
}
