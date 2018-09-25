package ledger

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	AccountTypeError    = 0
	AccountTypeNotExist = 1
	AccountTypeGeneral  = 2
	AccountTypeContract = 3
)

var publicKey, _ = hex.DecodeString("3af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a")
var GenesisPublicKey = ed25519.PublicKey(publicKey)

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
