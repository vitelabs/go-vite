package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
)

const (
	PENDING VerifyResult = iota
	FAIL
	SUCCESS
)

type SnapshotPendingTask struct {
	Hash *types.Hash
}

type AccountPendingTask struct {
	Addr *types.Address
	Hash *types.Hash
}
