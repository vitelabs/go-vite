package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

type Account struct {
	AccountAddress types.Address
	AccountId      uint64
	PublicKey      ed25519.PublicKey
}

func (*Account) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*Account) DbDeserialize([]byte) error {
	return nil
}
