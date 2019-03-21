package pmverifier

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type Verifier interface {
	VerifyNetAccBlock(block ledger.AccountBlock) error
	VerifyRPCAccBlock(block ledger.AccountBlock) (*vm_context.VmAccountBlock, error)
	VerifyPoolAccBlock(block ledger.AccountBlock) (*AccBlockPendingTask, *vm_context.VmAccountBlock, error)

	VerifyReferred(block ledger.AccountBlock) (VerifyResult, *AccBlockPendingTask, error)
	VerifyVM(block ledger.AccountBlock) (*vm_context.VmAccountBlock, error)

	VerifyAccBlockNonce(block ledger.AccountBlock) error
	VerifyAccBlockHash(block ledger.AccountBlock) error
	VerifyAccBlockSignature(block ledger.AccountBlock) error
	VerifyAccBlockConfirmedTimes(block ledger.AccountBlock) error
	VerifyAccBlockProducerLegality(block ledger.AccountBlock) error
}
