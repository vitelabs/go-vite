package verifier

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type VerifyResult int

type verifier struct {
	Sv *SnapshotVerifier
	Av *AccountVerifier
}

func NewVerifier(sv *SnapshotVerifier, av *AccountVerifier) Verifier {
	return &verifier{
		Sv: sv,
		Av: av,
	}
}

func (v *verifier) VerifyNetSnapshotBlock(block *ledger.SnapshotBlock) error {
	return v.Sv.VerifyNetSb(block)
}

func (v *verifier) VerifyNetAccBlock(block *ledger.AccountBlock) error {
	//todo 1. makesure genesis and initial-balance blocks don't need to check, return nil
	//todo 2. block referred snapshot not arrive yet, return error
	//3. VerifyHash
	if err := v.Av.verifyHash(block); err != nil {
		return err
	}
	//4. VerifySignature
	if block.IsReceiveBlock() || (len(block.Signature) > 0 || len(block.PublicKey) > 0) {
		if err := v.Av.verifySignature(block); err != nil {
			return err
		}
	}
	return nil
}

func (v *verifier) VerifyPoolAccBlock(block *ledger.AccountBlock, snapshotHash *types.Hash) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error) {
	verifyResult, task, err := v.Av.verifyReferred(block)
	switch verifyResult {
	case PENDING:
		return task, nil, nil
	case SUCCESS:
		blocks, err := v.Av.vmVerify(block, snapshotHash)
		if err != nil {
			return nil, nil, err
		}
		return nil, blocks, nil
	default:
		return nil, nil, err
	}
}

func (v *verifier) VerifyRPCAccBlock(block *ledger.AccountBlock, snapshotHash *types.Hash) (*vm_db.VmAccountBlock, error) {
	if verifyResult, task, err := v.Av.verifyReferred(block); verifyResult != SUCCESS {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("verify block failed, pending for:" + task.pendingHashListToStr())
	}
	return v.Av.vmVerify(block, snapshotHash)
}

func (v *verifier) VerifyAccBlockHash(block *ledger.AccountBlock) error {
	return v.Av.verifyHash(block)
}

func (v *verifier) VerifyAccBlockSignature(block *ledger.AccountBlock) error {
	accType, err := v.Av.verifyAccAddress(block)
	if err != nil {
		return err
	}
	if accType == AccountTypeNotSure {
		return ErrVerifyAccountTypeNotSure
	}
	isGeneralAddr := isAccTypeGeneral(accType)
	if !isGeneralAddr && block.IsSendBlock() {
		if len(block.Signature) != 0 || len(block.PublicKey) != 0 {
			return errors.New("signature and publicKey of the contract's send must be nil")
		}
	} else {
		if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
			return errors.New("signature or publicKey can't be nil")
		}
		if err := v.Av.verifySignature(block); err != nil {
			return err
		}
	}
	return nil
}

func (v *verifier) VerifyAccBlockNonce(block *ledger.AccountBlock) error {
	accType, err := v.Av.verifyAccAddress(block)
	if err != nil {
		return err
	}
	if accType == AccountTypeNotSure {
		return ErrVerifyAccountTypeNotSure
	}
	return v.Av.verifyNonce(block, isAccTypeGeneral(accType))
}

func (v *verifier) VerifyAccBlockConfirmedTimes(block *ledger.AccountBlock) error {
	accType, err := v.Av.verifyAccAddress(block)
	if err != nil {
		return err
	}
	if accType == AccountTypeNotSure {
		return ErrVerifyAccountTypeNotSure
	}
	return v.Av.verifyComfirmedTimes(block, isAccTypeGeneral(accType))
}

func (v *verifier) VerifyAccBlockProducerLegality(block *ledger.AccountBlock) error {
	accType, err := v.Av.verifyAccAddress(block)
	if err != nil {
		return err
	}
	if accType == AccountTypeNotSure {
		return ErrVerifyAccountTypeNotSure
	}
	return v.Av.verifyProducerLegality(block, isAccTypeGeneral(accType))
}
