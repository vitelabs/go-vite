package interfaces

import "github.com/vitelabs/go-vite/common/types"

// SignFunc is the function type defining the callback when a block requires a
// method to sign the transaction in generator.
type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)
