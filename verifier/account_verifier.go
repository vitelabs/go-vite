package verifier

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
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

type AccountType int

type AccountVerifier struct {
	chain     chain.Chain
	consensus consensus
	orManager onRoadPool

	log log15.Logger
}

func NewAccountVerifier(chain chain.Chain, consensus consensus) *AccountVerifier {
	return &AccountVerifier{
		chain:     chain,
		consensus: consensus,

		log: log15.New("module", "AccountVerifier"),
	}
}

func (v *AccountVerifier) InitOnRoadPool(manager *onroad.Manager) {
	v.orManager = manager
}

func (v *AccountVerifier) verifyReferred(block *ledger.AccountBlock) (VerifyResult, *AccBlockPendingTask, error) {
	pendingTask := &AccBlockPendingTask{}

	if err := v.verifySelf(block); err != nil {
		return FAIL, pendingTask, err
	}

	result, err := v.verifyDependency(pendingTask, block)
	if result != SUCCESS {
		if result == PENDING {
			pendingTask.AccountTask = append(pendingTask.AccountTask, &AccountPendingTask{Addr: &block.AccountAddress, Hash: &block.Hash})
		}
		return result, pendingTask, err
	}
	return SUCCESS, nil, nil
}

func (v *AccountVerifier) verifyConfirmedTimes(recvBlock *ledger.AccountBlock) error {
	if !types.IsContractAddr(recvBlock.AccountAddress) {
		return nil
	}
	meta, err := v.chain.GetContractMeta(recvBlock.AccountAddress)
	if err != nil {
		return errors.New("call GetContractMeta failed," + err.Error())
	}
	if meta == nil {
		return errors.New("contract meta is nil")
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
	return nil
}

func (v *AccountVerifier) verifySelf(block *ledger.AccountBlock) error {
	if err := v.checkAccountAddress(block); err != nil {
		return err
	}
	if block.IsSendBlock() {
		if err := v.verifySendBlockIntegrity(block); err != nil {
			return err
		}
	} else {
		if err := v.verifyReceiveBlockIntegrity(block); err != nil {
			return err
		}
	}
	if err := v.verifyProducerLegality(block); err != nil {
		return err
	}
	if err := v.verifyNonce(block); err != nil {
		return err
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
			return errors.New("contract address's meta is nil")
		}
	}
	return nil
}

func (v *AccountVerifier) verifyDependency(pendingTask *AccBlockPendingTask, block *ledger.AccountBlock) (VerifyResult, error) {
	// check the prev
	latestBlock, err := v.chain.GetLatestAccountBlock(block.AccountAddress)
	if err != nil {
		return FAIL, err
	}
	if latestBlock == nil {
		if block.Height != 1 || !block.PrevHash.IsZero() {
			return FAIL, ErrVerifyPrevBlockFailed
		}
	} else {
		if block.Height != latestBlock.Height+1 || block.PrevHash != latestBlock.Hash {
			return FAIL, ErrVerifyPrevBlockFailed
		}
	}

	if block.IsReceiveBlock() {
		// check the existence of receive's send
		if block.FromBlockHash.IsZero() {
			return FAIL, errors.New("recvBlock FromBlockHash can't be ZERO_HASH")
		}
		sendBlock, err := v.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return FAIL, err
		}
		if sendBlock == nil {
			pendingTask.AccountTask = append(pendingTask.AccountTask,
				&AccountPendingTask{Addr: nil, Hash: &block.FromBlockHash})
			return PENDING, nil
		}

		// check whether the send referred is already received
		isReceived, err := v.chain.IsReceived(sendBlock.Hash)
		if err != nil {
			return FAIL, err
		}
		if isReceived {
			received, err := v.chain.GetReceiveAbBySendAb(block.FromBlockHash)
			if err == nil && received != nil {
				return FAIL, errors.Errorf("block is already received successfully[received:%s, from:%s]", received.Hash, sendBlock.Hash)
			}
			return FAIL, errors.New("block is already received successfully")
		}

		// check contract receive sequence
		if types.IsContractAddr(sendBlock.ToAddress) {
			isCorrect, err := v.verifySequenceOfContractReceive(sendBlock)
			if err != nil {
				return FAIL, errors.New(fmt.Sprintf("verifySequenceOfContractReceive failed, err:%v", err))
			}
			if !isCorrect {
				return FAIL, errors.New("verifySequenceOfContractReceive failed")
			}
		}

		// check confirmedTimes of the send referred
		if err := v.verifyConfirmedTimes(block); err != nil {
			return FAIL, err
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
		if block.Amount != nil && block.Amount.Cmp(math.ZeroInt) > 0 {
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
			return errors.New("general's sendBlock.Height must be larger than 1")
		}
	}
	return nil
}

func (v *AccountVerifier) verifyReceiveBlockIntegrity(block *ledger.AccountBlock) error {

	if block.TokenId != types.ZERO_TOKENID {
		return errors.New("receive.TokenId must be ZERO_TOKENID")
	}
	if block.Amount != nil && block.Amount.Cmp(math.ZeroInt) != 0 {
		return errors.New("receive.Amount can't be anything other than nil or 0 ")
	}
	if block.Fee != nil && block.Fee.Cmp(math.ZeroInt) != 0 {
		return errors.New("receive.Fee can't be anything other than nil or 0")
	}
	if block.ToAddress != types.ZERO_ADDRESS {
		return errors.New("receive.ToAddress must be ZERO_ADDRESS")
	}
	if block.Height <= 0 {
		return errors.New("receive.Height must be larger than 0")
	}

	if len(block.SendBlockList) > 0 && !types.IsContractAddr(block.AccountAddress) {
		return errors.New("generalAddr's receive.SendBlockList must be nil")
	}
	for k, sendBlock := range block.SendBlockList {
		if err := v.verifySendBlockIntegrity(sendBlock); err != nil {
			v.log.Error(fmt.Sprintf("err:%v, contract:%v, recv-subSends[%v](%v, %v)",
				err.Error(), block.AccountAddress, k, block.Hash, sendBlock.Hash), "method", "verifyReceiveBlockIntegrity")
			return err
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
		return errors.New("hash can't be allzero")
	}
	if computedHash != block.Hash {
		//verifier.log.Error("checkHash failed", "originHash", block.Hash, "computedHash", computedHash)
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
			return errors.New("nonce of contractAddr's block must be nil")
		}
		if block.Difficulty == nil {
			return errors.New("difficulty can't be nil")
		}
		if len(block.Nonce) != 8 {
			return errors.New("nonce length doesn't satisfy with 8")
		}
		hash256Data := crypto.Hash256(block.AccountAddress.Bytes(), block.PrevHash.Bytes())
		if !pow.CheckPowNonce(block.Difficulty, block.Nonce, hash256Data) {
			return ErrVerifyNonceFailed
		}
	} else {
		if block.Difficulty != nil {
			return errors.New("difficulty must be nil when nonce is nil")
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
			return errors.New("fail to find receive's send in verifyProducerLegality")
		}
		if send.ToAddress != block.AccountAddress {
			return errors.New("receive's AccountAddress doesn't match the send'ToAddress")
		}
	}
	if types.IsContractAddr(block.AccountAddress) {
		if block.IsReceiveBlock() {
			if result, err := v.consensus.VerifyAccountProducer(block); !result {
				if err != nil {
					v.log.Error(err.Error())
				}
				return errors.New("contract-block's producer is illegal")
			}
		}
		return nil
	}
	if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
		return errors.New("general-account's publicKey doesn't match with the address")
	}
	return nil
}

func (v *AccountVerifier) vmVerify(block *ledger.AccountBlock, snapshotHash *types.Hash) (*vm_db.VmAccountBlock, error) {
	var fromBlock *ledger.AccountBlock
	var recvErr error
	if block.IsReceiveBlock() {
		fromBlock, recvErr = v.chain.GetAccountBlockByHash(block.FromBlockHash)
		if recvErr != recvErr {
			return nil, recvErr
		}
		if fromBlock == nil {
			return nil, errors.New("failed to find the recvBlock's fromBlock")
		}
	}
	gen, err := generator.NewGenerator(v.chain, v.consensus, block.AccountAddress, snapshotHash, &block.PrevHash)
	if err != nil {
		return nil, ErrVerifyForVmGeneratorFailed
	}
	genResult, err := gen.GenerateWithBlock(block, fromBlock)
	if err != nil {
		return nil, ErrVerifyForVmGeneratorFailed
	}
	if genResult == nil {
		return nil, errors.New("genResult is nil")
	}
	if genResult.VmBlock == nil {
		if genResult.Err != nil {
			return nil, genResult.Err
		}
		return nil, errors.New("vm failed, blockList is empty")
	}
	if err := v.verifyVMResult(block, genResult.VmBlock.AccountBlock); err != nil {
		return nil, errors.New("Inconsistent execution results in vm, err:" + err.Error())
	}
	return genResult.VmBlock, nil
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
		return errors.New("data")
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
