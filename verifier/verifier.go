package verifier

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vm_db"
)

// Verifier provides methods that external modules can use.
type Verifier interface {
	VerifyNetSnapshotBlock(block *ledger.SnapshotBlock) error
	VerifyNetAccountBlock(block *ledger.AccountBlock) error

	VerifyRPCAccountBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*vm_db.VmAccountBlock, error)
	VerifyPoolAccountBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error)

	VerifyAccountBlockNonce(block *ledger.AccountBlock) error
	VerifyAccountBlockHash(block *ledger.AccountBlock) error
	VerifyAccountBlockSignature(block *ledger.AccountBlock) error
	VerifyAccountBlockProducerLegality(block *ledger.AccountBlock) error

	VerifySnapshotBlockHash(block *ledger.SnapshotBlock) error
	VerifySnapshotBlockSignature(block *ledger.SnapshotBlock) error

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
func NewVerifier2(ch chain.Chain, cs consensus.Consensus) Verifier {
	return &verifier{
		Sv:  NewSnapshotVerifier(ch, cs),
		Av:  NewAccountVerifier(ch, cs),
		log: log15.New("module", "verifier"),
	}
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

func (v *verifier) VerifyNetSnapshotBlock(block *ledger.SnapshotBlock) error {
	return v.Sv.VerifyNetSb(block)
}

func (v *verifier) VerifyNetAccountBlock(block *ledger.AccountBlock) error {
	//1. VerifyHash
	if err := v.VerifyAccountBlockHash(block); err != nil {
		return err
	}
	//2. VerifySignature
	if err := v.VerifyAccountBlockSignature(block); err != nil {
		return err
	}
	return nil
}

func (v *verifier) VerifyPoolAccountBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*AccBlockPendingTask, *vm_db.VmAccountBlock, error) {
	eLog := v.log.New("method", "VerifyPoolAccountBlock")

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
		eLog.Error(err.Error()+":"+err.Detail(), "d", detail)
	}
	switch verifyResult {
	case PENDING:
		return task, nil, nil
	case SUCCESS:
		blocks, err := v.Av.vmVerify(block, snapshotHashHeight)
		if err != nil {
			eLog.Error(err.Error()+":"+err.Detail(), "d", detail)
			return nil, nil, err
		}
		return nil, blocks, nil
	default:
		return nil, nil, err
	}
}

func (v *verifier) VerifyRPCAccountBlock(block *ledger.AccountBlock, snapshot *ledger.SnapshotBlock) (*vm_db.VmAccountBlock, error) {
	log := v.log.New("method", "VerifyRPCAccountBlock")

	detail := fmt.Sprintf("sbHash:%v %v; addr:%v, height:%v, hash:%v, pow:(%v,%v)", snapshot.Hash, snapshot.Height, block.AccountAddress, block.Height, block.Hash, block.Difficulty, block.Nonce)
	if block.IsReceiveBlock() {
		detail += fmt.Sprintf(",fromH:%v", block.FromBlockHash)
	}
	snapshotHashHeight := &ledger.HashHeight{
		Height: snapshot.Height,
		Hash:   snapshot.Hash,
	}
	if err := v.VerifyNetAccountBlock(block); err != nil {
		log.Error(err.Error(), "d", detail)
		return nil, err
	}

	if verifyResult, task, err := v.Av.verifyReferred(block, snapshotHashHeight); verifyResult != SUCCESS {
		if err != nil {
			log.Error(err.Error()+":"+err.Detail(), "d", detail)
			return nil, err
		}
		log.Error("verify block failed, pending for:"+task.pendingHashListToStr(), "d", detail)
		return nil, ErrVerifyRPCBlockPendingState
	}

	vmBlock, err := v.Av.vmVerify(block, snapshotHashHeight)
	if err != nil {
		log.Error(err.Error()+":"+err.Detail(), "d", detail)
		return nil, err
	}
	return vmBlock, nil
}

func (v *verifier) VerifyAccountBlockHash(block *ledger.AccountBlock) error {
	return v.Av.verifyHash(block)
}

func (v *verifier) VerifyAccountBlockSignature(block *ledger.AccountBlock) error {
	if v.Av.chain.IsGenesisAccountBlock(block.Hash) {
		return nil
	}
	return v.Av.verifySignature(block)
}

func (v *verifier) VerifyAccountBlockNonce(block *ledger.AccountBlock) error {
	return v.Av.verifyNonce(block)
}

func (v *verifier) VerifyAccountBlockProducerLegality(block *ledger.AccountBlock) error {
	return v.Av.verifyProducerLegality(block)
}

func (v *verifier) GetSnapshotVerifier() *SnapshotVerifier {
	return v.Sv
}

func (v *verifier) VerifySnapshotBlockHash(block *ledger.SnapshotBlock) error {
	computedHash := block.ComputeHash()
	if block.Hash.IsZero() || computedHash != block.Hash {
		return ErrVerifyHashFailed
	}
	return nil
}

func (v *verifier) VerifySnapshotBlockSignature(block *ledger.SnapshotBlock) error {
	if v.Sv.reader.IsGenesisSnapshotBlock(block.Hash) {
		return nil
	}

	if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
		return errors.New("signature or publicKey is nil")
	}
	isVerified, _ := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if !isVerified {
		return ErrVerifySignatureFailed
	}
	return nil
}
