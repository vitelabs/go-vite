package ledger

import (
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

type Account struct {
	AccountId uint64
	PublicKey ed25519.PublicKey
}

func (*Account) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*Account) DbDeserialize([]byte) error {
	return nil
}
