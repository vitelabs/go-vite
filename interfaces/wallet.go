package interfaces

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

// SignFunc is the function type defining the callback when a block requires a
// method to sign the transaction in generator.
type SignFunc func(hash types.Hash) (signedData []byte, pub ed25519.PublicKey, err error)

type Account interface {
	Address() types.Address
	Sign(hash types.Hash) (signData []byte, pub ed25519.PublicKey, err error)
}
