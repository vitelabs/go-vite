package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
)

type VerifyResult int

const (
	FAIL VerifyResult = iota
	PENDING
	SUCCESS
)

type AccountPendingTask struct {
	Addr   *types.Address
	Hash   *types.Hash
	Height uint64
}
type SnapshotPendingTask struct {
	Hash *types.Hash
}
