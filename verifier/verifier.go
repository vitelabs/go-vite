package verifier

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type VerifyResult int

const (
	PENDING VerifyResult = iota
	FAIL
	SUCCESS
)

type AccountPendingTask struct {
	Addr   *types.Address
	Hash   *types.Hash
	Height *big.Int
}
type SnapshotPendingTask struct {
	Hash *types.Hash
}
