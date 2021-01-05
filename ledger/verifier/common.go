package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
)

// PENDING represents the block which can't be determined because its dependent transactions were not verified,
// FAIL represents the block which is found illegal or unqualified,
// SUCCESS represents the block is successfully verified.
const (
	PENDING VerifyResult = iota
	FAIL
	SUCCESS
)

// SnapshotPendingTask defines to carry the information of those snapshot block to be processed(PENDING).
type SnapshotPendingTask struct {
	Hash *types.Hash
}

// AccountPendingTask defines to carry the information of those account block to be processed(PENDING).
type AccountPendingTask struct {
	Addr *types.Address
	Hash *types.Hash
}
