package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type Verifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error

	VerifyRPCAccBlock(block *ledger.AccountBlock, snapshotHash *types.Hash) (*vm_db.VmAccountBlock, error)
	VerifyPoolAccBlock(block *ledger.AccountBlock, snapshotHash *types.Hash) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error)

	VerifyAccBlockNonce(block *ledger.AccountBlock) error
	VerifyAccBlockHash(block *ledger.AccountBlock) error
	VerifyAccBlockSignature(block *ledger.AccountBlock) error
	VerifyAccBlockProducerLegality(block *ledger.AccountBlock) error

	GetSnapshotVerifier() *SnapshotVerifier
}
