package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	AccountTypeError    = 0
	AccountTypeNotExist = 1
	AccountTypeGeneral  = 2
	AccountTypeContract = 3
)

type Account struct {
	AccountAddress types.Address
	AccountId      uint64
	PublicKey      ed25519.PublicKey
}

func (*Account) Serialize() ([]byte, error) {
	return nil, nil
}

func (*Account) Deserialize([]byte) error {
	return nil
}
