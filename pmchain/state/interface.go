package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type StateSnapshot interface {
	// ====== balance ======
	GetBalance(tokenId *types.TokenTypeId) (*big.Int, error)

	// ====== code ======
	GetCode() ([]byte, error)

	// ====== Storage ======
	GetValue([]byte) ([]byte, error)
}
