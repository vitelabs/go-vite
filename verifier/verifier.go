package verifier

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vm_db"
)

// Verifier provides methods that external modules can use.
type Verifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error

	VerifyRPCAccBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*vm_db.VmAccountBlock, error)
	VerifyPoolAccBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error)

	VerifyAccBlockNonce(block *ledger.AccountBlock) error
	VerifyAccBlockHash(block *ledger.AccountBlock) error
	VerifyAccBlockSignature(block *ledger.AccountBlock) error
	VerifyAccBlockProducerLegality(block *ledger.AccountBlock) error

	GetSnapshotVerifier() *SnapshotVerifier

	InitOnRoadPool(manager *onroad.Manager)
}

// VerifyResult explains the states of transaction validation.
type VerifyResult int

type verifier struct {
	Sv *SnapshotVerifier
	Av *AccountVerifier

	log log15.Logger
}

// NewVerifier needs instances of SnapshotVerifier and AccountVerifier.
func NewVerifier(sv *SnapshotVerifier, av *AccountVerifier) Verifier {
	return &verifier{
		Sv:  sv,
		Av:  av,
		log: log15.New("module", "verifier"),
	}
}

func (v *verifier) InitOnRoadPool(manager *onroad.Manager) {
	v.Av.InitOnRoadPool(manager)
}

func (v *verifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	return v.Sv.VerifyNetSb(block)
}

func (v *verifier) VerifyNetAb(block *ledger.AccountBlock) error {
	// fixme 1. makesure genesis and initial-balance blocks don't need to check, return nil
	//1. VerifyHash
	if err := v.VerifyAccBlockHash(block); err != nil {
		return err
	}
	//2. VerifySignature
	if err := v.VerifyAccBlockSignature(block); err != nil {
		return err
	}
	return nil
}

func (v *verifier) VerifyPoolAccBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error) {
	eLog := v.log.New("method", "VerifyPoolAccBlock")

	detail := fmt.Sprintf("sbHash:%v %v; block:addr=%v height=%v hash=%v; ", snapshot.Hash, snapshot.Height, block.AccountAddress, block.Height, block.Hash)
	if block.IsReceiveBlock() {
		detail += fmt.Sprintf("fromHash=%v;", block.FromBlockHash)
	}
	snapshotHashHeight := &ledger.HashHeight{
		Height: snapshot.Height,
		Hash:   snapshot.Hash,
	}
	verifyResult, task, err := v.Av.verifyReferred(block, snapshotHashHeight)
	if err != nil {
		eLog.Error(err.Error(), "d", detail)
	}
	switch verifyResult {
	case PENDING:
		return task, nil, nil
	case SUCCESS:
		blocks, err := v.Av.vmVerify(block, snapshotHashHeight)
		if err != nil {
			eLog.Error(err.Error(), "d", detail)
			return nil, nil, err
		}
		return nil, blocks, nil
	default:
		return nil, nil, err
	}
}

func (v *verifier) VerifyRPCAccBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*vm_db.VmAccountBlock, error) {
	log := v.log.New("method", "VerifyRPCAccBlock")

	detail := fmt.Sprintf("sbHash:%v %v; addr:%v, height:%v, hash:%v", snapshot.Hash, snapshot.Height, block.AccountAddress, block.Height, block.Hash)
	if block.IsReceiveBlock() {
		detail += fmt.Sprintf(",fromH:%v", block.FromBlockHash)
	}
	snapshotHashHeight := &ledger.HashHeight{
		Height: snapshot.Height,
		Hash:   snapshot.Hash,
	}

	if verifyResult, task, err := v.Av.verifyReferred(block, snapshotHashHeight); verifyResult != SUCCESS {
		if err != nil {
			log.Error(err.Error(), "d", detail)
			return nil, err
		}
		log.Error("verify block failed, pending for:"+task.pendingHashListToStr(), "d", detail)
		return nil, ErrVerifyRPCBlockPendingState
	}

	vmBlock, err := v.Av.vmVerify(block, snapshotHashHeight)
	if err != nil {
		log.Error(err.Error(), "d", detail)
		return nil, err
	}
	return vmBlock, nil
}

func (v *verifier) VerifyAccBlockHash(block *ledger.AccountBlock) error {
	return v.Av.verifyHash(block)
}

func (v *verifier) VerifyAccBlockSignature(block *ledger.AccountBlock) error {
	if v.Av.chain.IsGenesisAccountBlock(block.Hash) {
		return nil
	}
	return v.Av.verifySignature(block)
}

func (v *verifier) VerifyAccBlockNonce(block *ledger.AccountBlock) error {
	return v.Av.verifyNonce(block)
}

func (v *verifier) VerifyAccBlockProducerLegality(block *ledger.AccountBlock) error {
	return v.Av.verifyProducerLegality(block)
}

func (v *verifier) GetSnapshotVerifier() *SnapshotVerifier {
	return v.Sv
}
