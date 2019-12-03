package verifier

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm_db"
)

// AccountVerifier implements all method to verify the transaction.
type AccountVerifier struct {
	chain     accountChain
	consensus cssConsensus
	orManager onRoadPool

	log log15.Logger
}

// NewAccountVerifier needs two args, the implementation methods of the "accountChain" and "cssConsensus"
func NewAccountVerifier(chain accountChain, consensus cssConsensus) *AccountVerifier {
	return &AccountVerifier{
		chain:     chain,
		consensus: consensus,

		log: log15.New("module", "AccountVerifier"),
	}
}

// InitOnRoadPool method implements the inspection on whether the Contract's onRoad block is at the lowest height.
func (v *AccountVerifier) InitOnRoadPool(manager *onroad.Manager) {
	v.log.Info("InitOnRoadPool")
	v.orManager = manager
}

func (v *AccountVerifier) verifyReferred(block *ledger.AccountBlock, snapshotHashHeight *ledger.HashHeight) (VerifyResult, *AccBlockPendingTask, *VerifierError) {
	pendingTask := &AccBlockPendingTask{}

	if err := v.verifySelf(block); err != nil {
		return FAIL, pendingTask, err
	}

	result, err := v.verifyDependency(pendingTask, block, snapshotHashHeight)
	if result != SUCCESS {
		if result == PENDING {
			pendingTask.AccountTask = append(pendingTask.AccountTask, &AccountPendingTask{Addr: &block.AccountAddress, Hash: &block.Hash})
		}
		return result, pendingTask, err
	}
	return SUCCESS, nil, nil
}

func (v *AccountVerifier) verifyConfirmedTimes(recvBlock *ledger.AccountBlock, sbHeight uint64) error {
	meta, err := v.chain.GetContractMeta(recvBlock.AccountAddress)
	if err != nil {
		return errors.New("call GetContractMeta failed," + err.Error())
	}
	if meta == nil {
		return ErrVerifyContractMetaNotExists
	}
	if meta.SendConfirmedTimes == 0 {
		return nil
	}
	sendConfirmedTimes, err := v.chain.GetConfirmedTimes(recvBlock.FromBlockHash)
	if err != nil {
		return errors.New("call GetConfirmedTimes failed," + err.Error())
	}
	if sendConfirmedTimes < uint64(meta.SendConfirmedTimes) {
		return ErrVerifyConfirmedTimesNotEnough
	}
	if fork.IsSeedFork(sbHeight) && meta.SeedConfirmedTimes > 0 {
		isSeedCountOk, err := v.chain.IsSeedConfirmedNTimes(recvBlock.FromBlockHash, uint64(meta.SeedConfirmedTimes))
		if err != nil {
			return err
		}
		if !isSeedCountOk {
			return ErrVerifySeedConfirmedTimesNotEnough
		}
	}
	return nil
}

func (v *AccountVerifier) verifySelf(block *ledger.AccountBlock) *VerifierError {
	if err := v.checkAccountAddress(block); err != nil {
		return newError(err.Error())
	}
	if block.IsSendBlock() {
		if err := v.verifySendBlockIntegrity(block); err != nil {
			return newDetailError(ErrVerifyBlockFieldData.Error(), err.Error())
		}
	} else {
		if err := v.verifyReceiveBlockIntegrity(block); err != nil {
			return newDetailError(ErrVerifyBlockFieldData.Error(), err.Error())
		}
	}
	if err := v.verifyProducerLegality(block); err != nil {
		return newError(err.Error())
	}
	if err := v.verifyNonce(block); err != nil {
		return newError(err.Error())
	}
	return nil
}

func (v *AccountVerifier) checkAccountAddress(block *ledger.AccountBlock) error {
	if types.IsContractAddr(block.AccountAddress) {
		meta, err := v.chain.GetContractMeta(block.AccountAddress)
		if err != nil {
			return err
		}
		if meta == nil {
			return ErrVerifyContractMetaNotExists
		}
	} else {
		if block.IsSendBlock() && block.Height <= 1 {
			return ErrVerifyAccountNotInvalid
		}
	}
	return nil
}

func (v *AccountVerifier) verifyDependency(pendingTask *AccBlockPendingTask, block *ledger.AccountBlock, snapshotHashHeight *ledger.HashHeight) (VerifyResult, *VerifierError) {
	// check the prev
	latestBlock, err := v.chain.GetLatestAccountBlock(block.AccountAddress)
	if err != nil {
		return FAIL, newError(err.Error())
	}
	if latestBlock == nil {
		if block.Height != 1 || !block.PrevHash.IsZero() {
			return FAIL, newError(ErrVerifyPrevBlockFailed.Error())
		}
	} else {
		if block.Height != latestBlock.Height+1 || block.PrevHash != latestBlock.Hash {
			return FAIL, newError(ErrVerifyPrevBlockFailed.Error())
		}
	}

	if block.IsReceiveBlock() {
		// check the existence of receive's send
		if block.FromBlockHash.IsZero() {
			return FAIL, newDetailError(ErrVerifyBlockFieldData.Error(), "receive block FromBlockHash can't be ZERO_HASH")
		}
		sendBlock, err := v.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return FAIL, newError(err.Error())
		}
		if sendBlock == nil {
			pendingTask.AccountTask = append(pendingTask.AccountTask,
				&AccountPendingTask{Addr: nil, Hash: &block.FromBlockHash})
			return PENDING, nil
		}

		// check whether the send referred is already received
		isReceived, err := v.chain.IsReceived(block.FromBlockHash)
		if err != nil {
			return FAIL, newError(err.Error())
		}
		if isReceived {
			received, err := v.chain.GetReceiveAbBySendAb(block.FromBlockHash)
			if err == nil && received != nil {
				return FAIL, newDetailError(ErrVerifySendIsAlreadyReceived.Error(),
					fmt.Sprintf("already received[received:%s, from:%s]", received.Hash, block.FromBlockHash))
			}
			return FAIL, newError(ErrVerifySendIsAlreadyReceived.Error())
		}

		if types.IsContractAddr(block.AccountAddress) {
			// check contract receive sequence
			isCorrect, err := v.verifySequenceOfContractReceive(sendBlock)
			if err != nil {
				return FAIL, newDetailError(ErrVerifyContractReceiveSequenceFailed.Error(), err.Error())
			}
			if !isCorrect {
				return FAIL, newError(ErrVerifyContractReceiveSequenceFailed.Error())
			}

			// check confirmedTimes of the send referred
			if err := v.verifyConfirmedTimes(block, snapshotHashHeight.Height); err != nil {
				return FAIL, newError(err.Error())
			}
		}
	}

	return SUCCESS, nil
}

func (v *AccountVerifier) verifySequenceOfContractReceive(send *ledger.AccountBlock) (bool, error) {
	if v.orManager == nil {
		return false, errors.New(" onroad manager is not available or supported")
	}
	meta, err := v.chain.GetContractMeta(send.ToAddress)
	if err != nil || meta == nil {
		return false, errors.New("find contract meta nil, err is " + err.Error())
	}
	return v.orManager.IsFrontOnRoadOfCaller(meta.Gid, send.ToAddress, send.AccountAddress, send.Hash)
}

func (v *AccountVerifier) verifySendBlockIntegrity(block *ledger.AccountBlock) error {
	if block.TokenId == types.ZERO_TOKENID {
		if block.Amount != nil && block.Amount.Cmp(helper.Big0) != 0 {
			return errors.New("sendBlock.TokenId can't be ZERO_TOKENID when amount has value")
		}
	}
	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	} else {
		if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
			return errors.New("sendBlock.Amount out of bounds")
		}
	}

	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	} else {
		if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
			return errors.New("sendBlock.Fee out of bounds")
		}
	}
	if block.FromBlockHash != types.ZERO_HASH {
		return errors.New("sendBlock.FromBlockHash must be ZERO_HASH")
	}

	if types.IsContractAddr(block.AccountAddress) {
		if block.Height != 0 {
			return errors.New("contract's sendBlock.Height must be 0")
		}
		if len(block.Signature) != 0 || len(block.PublicKey) != 0 {
			return errors.New("signature and publicKey of the contract's send must be nil")
		}
	} else {
		if block.Height <= 1 {
			return ErrVerifyAccountNotInvalid
		}
	}
	return nil
}

func (v *AccountVerifier) verifyReceiveBlockIntegrity(block *ledger.AccountBlock) error {

	if block.TokenId != types.ZERO_TOKENID {
		return errors.New("receive.TokenId must be ZERO_TOKENID")
	}
	if block.Amount != nil && block.Amount.Cmp(helper.Big0) != 0 {
		return errors.New("receive.Amount can't be anything other than nil or 0 ")
	}
	if block.Fee != nil && block.Fee.Cmp(helper.Big0) != 0 {
		return errors.New("receive.Fee can't be anything other than nil or 0")
	}
	if block.ToAddress != types.ZERO_ADDRESS {
		return errors.New("receive.ToAddress must be ZERO_ADDRESS")
	}
	if block.Height <= 0 {
		return errors.New("receive.Height must be larger than 0")
	}
	if len(block.Data) > 0 && !types.IsContractAddr(block.AccountAddress) {
		return errors.New("receive.Data is not allowed when the account is general user")
	}

	if len(block.SendBlockList) > 0 && !types.IsContractAddr(block.AccountAddress) {
		return errors.New("generalAddr's receive.SendBlockList must be nil")
	}
	for k, sendBlock := range block.SendBlockList {
		if err := v.verifySendBlockIntegrity(sendBlock); err != nil {
			return errors.Errorf("%v, contract:%v, recv-subSends[%v](%v, %v)",
				err.Error(), block.AccountAddress, k, block.Hash, sendBlock.Hash)
		}
	}
	return nil
}

func (v *AccountVerifier) verifySignature(block *ledger.AccountBlock) error {
	if types.IsContractAddr(block.AccountAddress) && block.IsSendBlock() {
		if len(block.Signature) != 0 || len(block.PublicKey) != 0 {
			return errors.New("signature and publicKey of the contract's send must be nil")
		}
		return nil
	}
	if len(block.Signature) <= 0 || len(block.PublicKey) <= 0 {
		return errors.New("signature and publicKey all must have value")
	}
	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if verifyErr != nil {
		v.log.Error(verifyErr.Error(), "call", "VerifySig")
		return ErrVerifySignatureFailed
	}
	if !isVerified {
		return ErrVerifySignatureFailed
	}
	return nil
}

func (v *AccountVerifier) verifyHash(block *ledger.AccountBlock) error {
	computedHash := block.ComputeHash()
	if block.Hash.IsZero() {
		return ErrVerifyHashFailed
	}
	if computedHash != block.Hash {
		return ErrVerifyHashFailed
	}
	if !types.IsContractAddr(block.AccountAddress) || block.IsSendBlock() || len(block.SendBlockList) <= 0 {
		return nil
	}
	for idx, v := range block.SendBlockList {
		if v.Hash != v.ComputeSendHash(block, uint8(idx)) {
			return ErrVerifyHashFailed
		}
	}
	return nil
}

func (v *AccountVerifier) verifyNonce(block *ledger.AccountBlock) error {
	if len(block.Nonce) != 0 {
		if types.IsContractAddr(block.AccountAddress) {
			return ErrVerifyPowNotEligible
		}
		if block.Difficulty == nil {
			return ErrVerifyPowNotEligible
		}
		if len(block.Nonce) != 8 {
			return ErrVerifyPowNotEligible
		}
		hash256Data := crypto.Hash256(block.AccountAddress.Bytes(), block.PrevHash.Bytes())
		if !pow.CheckPowNonce(block.Difficulty, block.Nonce, hash256Data) {
			return ErrVerifyNonceFailed
		}
	} else {
		if block.Difficulty != nil {
			return ErrVerifyPowNotEligible
		}
	}
	return nil
}

func (v *AccountVerifier) verifyProducerLegality(block *ledger.AccountBlock) error {
	if block.IsReceiveBlock() {
		send, err := v.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if send == nil {
			return ErrVerifyDependentSendBlockNotExists
		}
		if send.ToAddress != block.AccountAddress {
			return ErrVerifyProducerIllegal
		}
	}
	if types.IsContractAddr(block.AccountAddress) {
		if block.IsReceiveBlock() {
			if result, err := v.consensus.VerifyAccountProducer(block); !result {
				if err != nil {
					v.log.Error(err.Error())
				}
				return ErrVerifyProducerIllegal
			}
		}
		return nil
	}
	if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
		return ErrVerifyProducerIllegal
	}
	return nil
}

func (v *AccountVerifier) vmVerify(block *ledger.AccountBlock, snapshotHashHeight *ledger.HashHeight) (*vm_db.VmAccountBlock, *VerifierError) {
	var fromBlock *ledger.AccountBlock
	if block.IsReceiveBlock() {
		var err error
		fromBlock, err = v.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return nil, newError(err.Error())
		}
		if fromBlock == nil {
			return nil, newError(ErrVerifyDependentSendBlockNotExists.Error())
		}
	}
	gen, err := generator.NewGenerator(v.chain, v.consensus, block.AccountAddress, &snapshotHashHeight.Hash, &block.PrevHash)
	if err != nil {
		return nil, newDetailError(ErrVerifyVmGeneratorFailed.Error(), err.Error())
	}
	genResult, err := gen.GenerateWithBlock(block, fromBlock)
	if err != nil {
		return nil, newDetailError(ErrVerifyVmGeneratorFailed.Error(), err.Error())
	}
	if genResult == nil {
		return nil, newDetailError(ErrVerifyVmGeneratorFailed.Error(), "genResult is nil")
	}
	if genResult.VMBlock == nil {
		if genResult.Err != nil {
			return nil, newError(genResult.Err.Error())
		}
		return nil, newError("vm failed, blockList is empty")
	}
	if err := v.verifyVMResult(block, genResult.VMBlock.AccountBlock); err != nil {
		return nil, newDetailError(ErrVerifyVmResultInconsistent.Error(), err.Error())
	}
	return genResult.VMBlock, nil
}

func (v *AccountVerifier) verifyVMResult(origBlock *ledger.AccountBlock, genBlock *ledger.AccountBlock) error {
	// BlockType AccountAddress ToAddress PrevHash Height Amount TokenId FromBlockHash Data Fee LogHash Nonce SendBlockList
	if origBlock.Hash == genBlock.Hash {
		return nil
	}

	if origBlock.BlockType != genBlock.BlockType {
		return errors.New("BlockType")
	}
	if origBlock.AccountAddress != genBlock.AccountAddress {
		return errors.New("AccountAddress")
	}
	if origBlock.ToAddress != genBlock.ToAddress {
		return errors.New("ToAddress")
	}
	if origBlock.FromBlockHash != genBlock.FromBlockHash {
		return errors.New("FromBlockHash")
	}
	if origBlock.Height != genBlock.Height {
		return errors.New("Height")
	}
	if !bytes.Equal(origBlock.Data, genBlock.Data) {
		return errors.New("Data")
	}
	if !bytes.Equal(origBlock.Nonce, genBlock.Nonce) {
		return errors.New("Nonce")
	}
	if (origBlock.LogHash == nil && genBlock.LogHash != nil) || (origBlock.LogHash != nil && genBlock.LogHash == nil) {
		return errors.New("LogHash")
	}
	if origBlock.LogHash != nil && genBlock.LogHash != nil && *origBlock.LogHash != *genBlock.LogHash {
		return errors.New("LogHash")
	}
	/* fixme
	if origBlock.QuotaUsed != genBlock.QuotaUsed {
		return errors.New("QuotaUsed")
	}*/

	if origBlock.IsSendBlock() {
		if origBlock.Fee.Cmp(genBlock.Fee) != 0 {
			return errors.New("Fee")
		}
		if origBlock.Amount.Cmp(genBlock.Amount) != 0 {
			return errors.New("Amount")
		}
		if origBlock.TokenId != genBlock.TokenId {
			return errors.New("TokenId")
		}
	} else {
		if len(origBlock.SendBlockList) != len(genBlock.SendBlockList) {
			return errors.New("SendBlockList len")
		}
		for k, v := range origBlock.SendBlockList {
			if v.Hash != genBlock.SendBlockList[k].Hash {
				return errors.New(fmt.Sprintf("SendBlockList[%v] Hash", k))
			}
		}
	}
	return errors.New("Hash")
}

func (v *AccountVerifier) verifyIsReceivedSucceed(block *ledger.AccountBlock) (bool, error) {
	return v.chain.IsReceived(block.FromBlockHash)
}

// AccBlockPendingTask defines a data structure for a pending transaction
type AccBlockPendingTask struct {
	AccountTask []*AccountPendingTask
}

func (task *AccBlockPendingTask) pendingHashListToStr() string {
	var pendHashListStr string
	for k, v := range task.AccountTask {
		pendHashListStr += v.Hash.String()
		if k < len(task.AccountTask)-1 {
			pendHashListStr += "|"
		}
	}
	return pendHashListStr
}
