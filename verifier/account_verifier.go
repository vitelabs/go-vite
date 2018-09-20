package verifier

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"time"
)

const (
	TimeOutHeight = uint64(24 * 30 * 3600)
	MaxBigIntLen  = 256
)

type AccountVerifier struct {
	chain           Chain
	signer          Signer
	consensusReader Consensus

	log log15.Logger
}

func NewAccountVerifier(chain Chain, consensus Consensus, signer Signer) *AccountVerifier {
	verifier := &AccountVerifier{
		chain:           chain,
		signer:          signer,
		consensusReader: consensus,
		log:             log15.New("class", "AccountVerifier"),
	}
	return verifier
}

func (verifier *AccountVerifier) newVerifyStat(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}

// fixme contractAddr's sendBlock don't call VerifyReferredforPool
func (verifier *AccountVerifier) VerifyReferred(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountReferredforPool", time.Now())

	stat := verifier.newVerifyStat(block)

	verifier.verifySelf(block, stat)
	verifier.verifyFrom(block, stat)
	verifier.verifySnapshot(block, stat)

	return stat.VerifyResult(), stat
}

func (verifier *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (this *vm_context.VmAccountBlock, others []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforVM", time.Now())

	gen := generator.NewGenerator(verifier.chain.Chain(), verifier.signer)
	if err = gen.PrepareVm(&block.SnapshotHash, &block.PrevHash, &block.AccountAddress); err != nil {
		return nil, nil, err
	}
	genResult := gen.GenerateWithP2PBlock(block, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return gen.Sign(addr, nil, data)
	})
	if genResult == nil || len(genResult.BlockGenList) == 0 {
		return nil, nil, errors.New("GenerateWithP2PBlock failed")
	}

	this = genResult.BlockGenList[0]
	if len(genResult.BlockGenList) > 1 {
		others = genResult.BlockGenList[1:]
	}
	return this, others, nil
}

func (verifier *AccountVerifier) VerifyforP2P(block *ledger.AccountBlock) bool {
	if result, err := verifier.verifySelfDataValidity(block); result != FAIL || err == nil {
		return true
	}
	return false
}

func (verifier *AccountVerifier) VerifyforRPC() ([]*vm_context.VmAccountBlock, error) {
	// todo 1.arg to be message or block
	// todo 2.generateBlock
	return nil, nil
}

func (verifier *AccountVerifier) verifySelf(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	isFail := func(result VerifyResult, err error, stat *AccountBlockVerifyStat) bool {
		if result == FAIL {
			stat.referredSelfResult = FAIL
			if err != nil {
				stat.errMsg += err.Error()
			}
			return true
		}
		return false
	}

	step1, err1 := verifier.verifySelfDataValidity(block)
	if isFail(step1, err1, verifyStatResult) {
		return
	}
	step2, err2 := verifier.verifyProducerLegality(block)
	if isFail(step2, err2, verifyStatResult) {
		return
	}
	step3, err3 := verifier.verifySelfPrev(block)
	if isFail(step3, err3, verifyStatResult) {
		return
	}

	if step1 == SUCCESS && step2 == SUCCESS && step3 == SUCCESS {
		verifyStatResult.referredSelfResult = SUCCESS
	} else {
		verifyStatResult.referredSelfResult = PENDING
		verifyStatResult.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
	}
}

func (verifier *AccountVerifier) verifyProducerLegality(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	var errMsg error
	if verifier.verifyIsContractAddress(&block.AccountAddress) {
		if verifier.verifyIsReceiveBlock(block) {
			if conErr := verifier.consensusReader.VerifyAccountProducer(block); conErr != nil {
				errMsg = errors.New("the block producer is illegal")
				verifier.log.Error(errMsg.Error(), "error", conErr)
				return FAIL, errMsg
			}
			return SUCCESS, nil
		} else {
			// fixme delete contractAddr sendBlock
			errMsg = errors.New("contractAddr sendBlock don't verify")
			verifier.log.Error(errMsg.Error())
			return FAIL, errMsg
		}
	} else {
		// commonAddr
		if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
			errMsg = errors.New("PublicKey match AccountAddress failed")
			verifier.log.Error(errMsg.Error())
			return FAIL, errMsg
		} else {
			return SUCCESS, nil
		}
	}
}

func (verifier *AccountVerifier) verifyFrom(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if verifier.verifyIsReceiveBlock(block) {
		fromBlock, err := verifier.chain.Chain().GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil || fromBlock == nil {
			verifier.log.Info("GetAccountBlockByHash", "error", err)
			verifyStatResult.accountTask = &AccountPendingTask{
				Addr:   nil,
				Hash:   &block.FromBlockHash,
				Height: 0,
			}
			verifyStatResult.referredFromResult = PENDING
		} else {
			verifyStatResult.referredFromResult = SUCCESS
		}
	} else {
		verifier.log.Info("verifyFrom: send doesn't have fromBlock")
		verifyStatResult.referredFromResult = SUCCESS
	}
}

func (verifier *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())

	snapshotBlock, err := verifier.chain.Chain().GetSnapshotBlockByHash(&block.SnapshotHash)
	if err != nil || snapshotBlock == nil {
		verifyStatResult.referredSnapshotResult = PENDING
	} else {
		verifyStatResult.referredSnapshotResult = SUCCESS
	}

	verifyResult, err := verifier.VerifyTimeOut(snapshotBlock)
	if err != nil || verifyResult == FAIL {
		verifyStatResult.errMsg += err.Error()
		verifier.log.Error("VerifyTimeOut", "error", err.Error())
		verifyStatResult.referredSnapshotResult = FAIL
	}

	if verifyStatResult.referredSnapshotResult == SUCCESS && verifyResult == SUCCESS {
		verifyStatResult.referredSnapshotResult = SUCCESS
	} else {
		verifyStatResult.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
		verifyStatResult.referredSnapshotResult = PENDING
	}
}

func (verifier *AccountVerifier) verifySelfDataValidity(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDataValidity", time.Now())

	isContractAddr := verifier.verifyIsContractAddress(&block.AccountAddress)

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	}
	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	}
	if block.PublicKey == nil || block.Timestamp == nil || block.Data == nil || block.Signature == nil ||
		(block.LogHash == nil && isContractAddr && verifier.verifyIsReceiveBlock(block)) {
		return FAIL, errors.New("block integrity miss")
	}

	// todo verify Nonce

	if block.Amount.Sign() < 0 || block.Amount.BitLen() > MaxBigIntLen {
		return FAIL, errors.New("block.Amount out of bounds")
	}
	if block.Fee.Sign() < 0 || block.Amount.BitLen() > MaxBigIntLen {
		return FAIL, errors.New("block.Fee out of bounds")
	}

	if !verifier.verifySelfSig(block) {
		return FAIL, errors.New("verify hash or signature failed")
	}

	return SUCCESS, nil
}

func (verifier *AccountVerifier) verifySelfSig(block *ledger.AccountBlock) bool {
	computedHash := block.ComputeHash()
	if block.Hash.Bytes() == nil || !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		verifier.log.Error("checkHash failed", "originHash", block.Hash)
		return false
	}

	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if verifyErr != nil || !isVerified {
		verifier.log.Error("VerifySig failed", "error", verifyErr)
		return false
	}
	return true
}

func (verifier *AccountVerifier) verifySelfPrev(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())
	var errMsg error

	latestBlock, err := verifier.chain.Chain().GetLatestAccountBlock(&block.AccountAddress)
	if err != nil || latestBlock == nil {
		errMsg = errors.New("GetLatestAccountBlock failed")
		verifier.log.Error(errMsg.Error(), "error", err)
		return FAIL, errMsg
	}

	if block.Height == latestBlock.Height+1 && block.PrevHash == latestBlock.Hash {
		return SUCCESS, nil
	}
	if block.Height > latestBlock.Height+1 && block.PrevHash != latestBlock.Hash {
		return PENDING, nil
	}
	errMsg = errors.New("PreHash or Height is invalid")
	verifier.log.Error(errMsg.Error())
	return FAIL, errMsg
}

func (verifier *AccountVerifier) VerifyChainInsertQualification(block *ledger.AccountBlock) bool {
	return false
}

func (verifier *AccountVerifier) VerifyTimeOut(blockReferSb *ledger.SnapshotBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSnapshotTimeout", time.Now())

	currentSb, err := verifier.chain.Chain().GetLatestSnapshotBlock()
	if err != nil || currentSb == nil {
		errMsg := errors.New("GetLatestSnapshotBlock failed")
		verifier.log.Error(errMsg.Error(), "error", err)
		return PENDING, errMsg
	}
	if currentSb.Height > blockReferSb.Height+TimeOutHeight {
		errMsg := errors.New("snapshot time out of limit")
		verifier.log.Error(errMsg.Error())
		return FAIL, errMsg
	}
	return SUCCESS, nil
}

func (verifier *AccountVerifier) verifyIsContractAddress(addr *types.Address) bool {
	gid, err := verifier.chain.Chain().GetContractGid(addr)
	if err != nil {
		verifier.log.Error("GetContractGid failed", "Error", err)
	}
	if gid != nil {
		return true
	}
	return false
}

func (verifier *AccountVerifier) verifyIsReceiveBlock(block *ledger.AccountBlock) bool {
	if (block.BlockType != ledger.BlockTypeSendCall) && (block.BlockType != ledger.BlockTypeSendCreate) {
		return true
	}
	return false
}

//func (self *AccountVerifier) VerifyConfirmed(block *ledger.AccountBlock) bool {
//	defer monitor.LogTime("verify", "accountConfirmed", time.Now())
//
//	if confirmedBlock, err := self.chain.Chain().GetConfirmBlock(&block.Hash); err != nil || confirmedBlock == nil {
//		self.log.Error("not Confirmed yet")
//		return false
//	}
//	return true
//}

//func (self *AccountVerifier) verifyGenesis(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
//	defer monitor.LogTime("verify", "accountGenesis", time.Now())
//	//fixme ask Liz for the GetGenesisBlockFirst() and GetGenesisBlockSecond()
//	return block.PrevHash.Bytes() == nil &&
//		bytes.Equal(block.Signature, self.chain.Chain().GetGenesisBlockFirst().Signature) &&
//		bytes.Equal(block.Hash.Bytes(), self.chain.Chain().GetGenesisBlockFirst().Hash.Bytes())
//}

//func (self *AccountVerifier) VerifyUnconfirmedPriorBlockReceived(priorBlockHash *types.Hash) bool {
//	existBlock, err := self.chain.GetAccountBlockByHash(priorBlockHash)
//	if err != nil || existBlock == nil {
//		self.log.Info("VerifyUnconfirmedPriorBlockReceived: the prev block hasn't existed in Chain. ")
//		return false
//	}
//	return true
//}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	accountTask            *AccountPendingTask
	errMsg                 string
}

func (self *AccountBlockVerifyStat) ErrMsg() string {
	return self.errMsg
}

func (self *AccountBlockVerifyStat) VerifyResult() VerifyResult {
	if self.referredSelfResult == FAIL ||
		self.referredFromResult == FAIL ||
		self.referredSnapshotResult == FAIL {
		return FAIL
	}
	if self.referredSelfResult == SUCCESS &&
		self.referredFromResult == SUCCESS &&
		self.referredSnapshotResult == SUCCESS {
		return SUCCESS
	}
	return PENDING
}
