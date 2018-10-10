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
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"time"
)

const (
	MaxBigIntLen  = 256
	TimeOutHeight = uint64(24 * 30 * 3600)
)

type AccountVerifier struct {
	chain     Chain
	signer    Signer
	consensus Consensus

	log log15.Logger
}

func NewAccountVerifier(chain Chain, consensus Consensus, signer Signer) *AccountVerifier {
	return &AccountVerifier{
		chain:     chain,
		signer:    signer,
		consensus: consensus,

		log: log15.New("class", "AccountVerifier"),
	}
}

func (verifier *AccountVerifier) newVerifyStat() *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}

func (verifier *AccountVerifier) VerifyforP2P(block *ledger.AccountBlock) bool {
	if verifier.VerifyTimeNotYet(block) {
		return false
	}
	if err := verifier.VerifyDataValidity(block); err != nil {
		return false
	}
	return true
}

func (verifier *AccountVerifier) VerifyforRPC(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforRPC", time.Now())
	if verifier.VerifyTimeNotYet(block) {
		return nil, errors.New("VerifyTimeNotYet failed")
	}
	if err := verifier.VerifyDataValidity(block); err != nil {
		//return nil, errors.New("VerifyDataValidity failed")
		return nil, err
	}
	if verifyResult, _ := verifier.VerifyReferred(block); verifyResult != SUCCESS {
		return nil, errors.New("VerifyReferred failed")
	}
	// fixme: whether to do pool-insert, distinguish common and contract
	return verifier.VerifyforVM(block)
}

func (verifier *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforVM", time.Now())

	var preHash *types.Hash
	if block.Height > 1 {
		preHash = &block.PrevHash
	}
	gen, err := generator.NewGenerator(verifier.chain, verifier.signer, &block.SnapshotHash, preHash, &block.AccountAddress)
	if err != nil {
		return nil, err
	}

	genResult, err := gen.GenerateWithBlock(block, nil)
	if err != nil {
		return nil, errors.New("GenerateWithBlock failed")
	}
	if len(genResult.BlockGenList) == 0 {
		return nil, errors.New("genResult is empty")
	}
	if err := verifier.verifyVMResult(block, genResult.BlockGenList[0].AccountBlock); err != nil {
		return nil, err
	}
	return genResult.BlockGenList, nil
}

func (verifier *AccountVerifier) verifyVMResult(origBlock *ledger.AccountBlock, genBlock *ledger.AccountBlock) error {
	if origBlock.BlockType != genBlock.BlockType {
		return errors.New("verify BlockType failed")
	}
	if origBlock.ToAddress != genBlock.ToAddress {
		return errors.New("verify ToAddress failed")
	}
	if origBlock.Fee.Cmp(genBlock.Fee) != 0 {
		return errors.New("verify Fee failed")
	}
	if !bytes.Equal(origBlock.Data, genBlock.Data) {
		return errors.New("verify Data failed")
	}
	if (origBlock.LogHash == nil && genBlock.LogHash != nil) || (origBlock.LogHash != nil && genBlock.LogHash == nil) {
		return errors.New("verify LogHash failed")
	}
	if origBlock.LogHash != nil && genBlock.LogHash != nil && *origBlock.LogHash != *genBlock.LogHash {
		return errors.New("verify LogHash failed")
	}

	return nil
}

// contractAddr's sendBlock don't call VerifyReferredforPool
func (verifier *AccountVerifier) VerifyReferred(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountReferredforPool", time.Now())

	stat := verifier.newVerifyStat()

	verifier.verifySelf(block, stat)
	verifier.verifyFrom(block, stat)
	verifier.verifySnapshot(block, stat)

	return stat.VerifyResult(), stat
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

	//if isTrue, err := verifier.VerifyDataValidity(block); !isTrue {
	//	verifyStatResult.referredSelfResult = FAIL
	//	verifyStatResult.errMsg += err.Error()
	//	return
	//}

	step1, err1 := verifier.verifyProducerLegality(block, verifyStatResult.accountTask)
	if isFail(step1, err1, verifyStatResult) {
		return
	}
	step2, err2 := verifier.verifySelfPrev(block, verifyStatResult.accountTask)
	if isFail(step2, err2, verifyStatResult) {
		return
	}

	if step1 == SUCCESS && step2 == SUCCESS {
		verifyStatResult.referredSelfResult = SUCCESS
	} else {
		verifyStatResult.accountTask = append(verifyStatResult.accountTask,
			&AccountPendingTask{Addr: &block.AccountAddress, Hash: &block.Hash})
		verifyStatResult.referredSelfResult = PENDING
	}
}

func (verifier *AccountVerifier) verifyProducerLegality(block *ledger.AccountBlock, task []*AccountPendingTask) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	var errMsg error
	code, err := verifier.chain.AccountType(&block.AccountAddress)
	if err != nil {
		verifier.log.Error("AccountType", "error", err)
	}
	if code == ledger.AccountTypeContract {
		if block.IsReceiveBlock() {
			if conErr := verifier.consensus.VerifyAccountProducer(block); conErr != nil {
				errMsg = errors.New("the block producer is illegal")
				verifier.log.Error(errMsg.Error(), "error", conErr)
				return FAIL, errMsg
			}
		}
	}
	if code == ledger.AccountTypeGeneral {
		if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
			errMsg = errors.New("PublicKey match AccountAddress failed")
			verifier.log.Error(errMsg.Error())
			return FAIL, errMsg
		}
	}
	// include VerifyAccountProducer success、the contractAddress's sendBlock and unknow's sendBlock
	return SUCCESS, nil
}

func (verifier *AccountVerifier) verifyFrom(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if block.IsReceiveBlock() {
		fromBlock, err := verifier.chain.GetAccountBlockByHash(&block.FromBlockHash)
		if fromBlock == nil {
			if err != nil {
				msgErr := errors.New("GetAccountBlockByHash failed.")
				verifier.log.Error(msgErr.Error(), "error", err)
				verifyStatResult.referredFromResult = FAIL
				verifyStatResult.errMsg += msgErr.Error()
			}
			verifyStatResult.accountTask = append(verifyStatResult.accountTask,
				&AccountPendingTask{Addr: nil, Hash: &block.FromBlockHash})
			verifyStatResult.referredFromResult = PENDING
		} else {
			if verifier.VerifySnapshotOfReferredBlock(block, fromBlock) {
				verifyStatResult.referredFromResult = SUCCESS
			} else {
				verifier.log.Error("GetAccountBlockByHash", "error", err)
				verifyStatResult.referredFromResult = FAIL
			}
		}
	} else {
		verifier.log.Info("sendBlock doesn't need to verifyFrom")
		verifyStatResult.referredFromResult = SUCCESS
	}
}

func (verifier *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())

	snapshotBlock, err := verifier.chain.GetSnapshotBlockByHash(&block.SnapshotHash)
	if snapshotBlock == nil {
		if err != nil {
			verifier.log.Error("GetAccountBlockByHash", "error", err)
		}
		verifyStatResult.snapshotTask = &SnapshotPendingTask{Hash: &block.SnapshotHash}
		verifyStatResult.referredSnapshotResult = PENDING
	} else {
		if isSucc := verifier.VerifyTimeOut(snapshotBlock); !isSucc {
			verifyStatResult.errMsg += errors.New("VerifyTimeOut").Error()
			verifyStatResult.referredSnapshotResult = FAIL
		} else {
			verifyStatResult.referredSnapshotResult = SUCCESS
		}
	}
}

func (verifier *AccountVerifier) VerifyDataValidity(block *ledger.AccountBlock) error {
	defer monitor.LogTime("verify", "accountSelfDataValidity", time.Now())

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	}
	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	}
	if block.Amount.Sign() < 0 || block.Amount.BitLen() > MaxBigIntLen {
		return errors.New("block.Amount out of bounds")
	}
	if block.Fee.Sign() < 0 || block.Fee.BitLen() > MaxBigIntLen {
		return errors.New("block.Fee out of bounds")
	}

	if block.Timestamp == nil {
		return errors.New("Timestamp can't be nil")
	}

	if !verifier.VerifyHash(block) {
		return errors.New("VerifyHash failed")
	}

	if !verifier.VerifyNonce(block) {
		return errors.New("VerifyNonce failed")
	}

	if !verifier.VerifySigature(block) {
		return errors.New("VerifySigature failed")
	}

	return nil
}

func (verifier *AccountVerifier) verifySelfPrev(block *ledger.AccountBlock, task []*AccountPendingTask) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())

	latestBlock, err := verifier.chain.GetLatestAccountBlock(&block.AccountAddress)
	if latestBlock == nil {
		if err != nil {
			errMsg := errors.New("GetLatestAccountBlock failed")
			verifier.log.Error(errMsg.Error(), "error", err)
			return FAIL, errMsg
		} else {
			if block.Height == 1 {
				prevZero := &types.Hash{}
				if !bytes.Equal(block.PrevHash.Bytes(), prevZero.Bytes()) {
					return FAIL, errors.New("check Account's first Block's PrevHash failed")
				}
				return SUCCESS, nil
			}
			task = append(task, &AccountPendingTask{nil, &block.PrevHash})
			return PENDING, nil
		}
	} else {
		if !verifier.VerifySnapshotOfReferredBlock(block, latestBlock) {
			return FAIL, errors.New("check referredBlock's snapshotBlock failed")
		}
		switch {
		case block.PrevHash == latestBlock.Hash && block.Height == latestBlock.Height+1:
			return SUCCESS, nil
		case block.PrevHash != latestBlock.Hash && block.Height > latestBlock.Height+1:
			task = append(task, &AccountPendingTask{nil, &block.PrevHash})
			return PENDING, nil
		default:
			return FAIL, errors.New("PreHash or Height is invalid")
		}
	}
}

func (verifier *AccountVerifier) VerifyChainInsertQualification(block *ledger.AccountBlock) bool {
	return false
}

func (verifier *AccountVerifier) VerifyHash(block *ledger.AccountBlock) bool {
	computedHash := block.ComputeHash()
	if block.Hash.IsZero() || computedHash != block.Hash {
		verifier.log.Error("checkHash failed", "originHash", block.Hash, "computedHash", computedHash)
		return false
	}
	return true
}

func (verifier *AccountVerifier) VerifySigature(block *ledger.AccountBlock) bool {
	code, err := verifier.chain.AccountType(&block.AccountAddress)
	if err != nil {
		verifier.log.Error("AccountType", "error", err)
		return false
	}

	if block.IsSendBlock() && (code == ledger.AccountTypeNotExist || code == ledger.AccountTypeContract) {
		if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
			return true
		}
		return false
	}

	if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
		return false
	}
	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if !isVerified {
		if verifyErr != nil {
			verifier.log.Error("VerifySig failed", "error", verifyErr)
		}
		return false
	}
	return true
}

func (verifier *AccountVerifier) VerifyNonce(block *ledger.AccountBlock) bool {
	if len(block.Nonce) != 0 {
		var nonce [8]byte
		copy(nonce[:], block.Nonce[:8])
		hash256Data := crypto.Hash256(block.AccountAddress.Bytes(), block.PrevHash.Bytes())
		if !pow.CheckPowNonce(nil, nonce, hash256Data) {
			return false
		}
	}
	return true
}

func (verifier *AccountVerifier) VerifyTimeOut(blockReferSb *ledger.SnapshotBlock) bool {
	currentSb := verifier.chain.GetLatestSnapshotBlock()
	if currentSb.Height > blockReferSb.Height+TimeOutHeight {
		errMsg := errors.New("snapshot time out of limit")
		verifier.log.Error(errMsg.Error())
		return false
	}
	return true
}

func (verifier *AccountVerifier) VerifyTimeNotYet(block *ledger.AccountBlock) bool {
	//  don't accept which timestamp doesn't satisfy within the (latestSnapshotBlock's + 1h) limit
	currentSb := verifier.chain.GetLatestSnapshotBlock()
	if block.Timestamp.Before(currentSb.Timestamp.Add(time.Hour)) {
		return false
	}
	return true
}

func (verifier *AccountVerifier) VerifySnapshotOfReferredBlock(thisBlock *ledger.AccountBlock, referredBlock *ledger.AccountBlock) bool {
	// referredBlock' snapshotBlock's height can't lower than thisBlock
	thisSnapshotBlock, _ := verifier.chain.GetSnapshotBlockByHash(&thisBlock.SnapshotHash)
	referredSnapshotBlock, _ := verifier.chain.GetSnapshotBlockByHash(&referredBlock.SnapshotHash)
	if thisSnapshotBlock != nil && referredSnapshotBlock != nil {
		if referredSnapshotBlock.Height <= thisSnapshotBlock.Height {
			return true
		}
	}
	return false
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	accountTask            []*AccountPendingTask
	snapshotTask           *SnapshotPendingTask
	errMsg                 string
}

func (result *AccountBlockVerifyStat) ErrMsg() string {
	return result.errMsg
}

func (result *AccountBlockVerifyStat) GetPendingTasks() ([]*AccountPendingTask, *SnapshotPendingTask) {
	return result.accountTask, result.snapshotTask
}

func (result *AccountBlockVerifyStat) VerifyResult() VerifyResult {
	if result.referredSelfResult == FAIL ||
		result.referredFromResult == FAIL ||
		result.referredSnapshotResult == FAIL {
		return FAIL
	}
	if result.referredSelfResult == SUCCESS &&
		result.referredFromResult == SUCCESS &&
		result.referredSnapshotResult == SUCCESS {
		return SUCCESS
	}
	return PENDING
}
