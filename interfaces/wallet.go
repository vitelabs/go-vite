package interfaces

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
)

// SignFunc is the function type defining the callback when a block requires a
// method to sign the transaction in generator.
type SignFunc func(msg []byte) (signedData []byte, pub ed25519.PublicKey, err error)

// VerifyFunc is the function
type VerifyFunc func(pub ed25519.PublicKey, message, signdata []byte) error

type Account interface {
	Address() types.Address
	Sign(msg []byte) (signData []byte, pub ed25519.PublicKey, err error)
	Verify(pub ed25519.PublicKey, message, signdata []byte) error
}
