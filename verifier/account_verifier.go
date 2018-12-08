package verifier

import (
	"bytes"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm_context"
)

var TimeOutHeight = uint64(30 * types.SnapshotDayHeight)

type AccountVerifier struct {
	chain     Chain
	consensus Consensus

	log log15.Logger
}

func NewAccountVerifier(chain Chain, consensus Consensus) *AccountVerifier {
	return &AccountVerifier{
		chain:     chain,
		consensus: consensus,

		log: log15.New("class", "AccountVerifier"),
	}
}

func (verifier *AccountVerifier) newVerifyStat() *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{
		referredSnapshotResult: PENDING,
		referredSelfResult:     PENDING,
		referredFromResult:     PENDING,
	}
}

func (verifier *AccountVerifier) VerifyNetAb(block *ledger.AccountBlock) error {
	if err := verifier.VerifyTimeNotYet(block); err != nil {
		return err
	}
	if err := verifier.VerifyP2PDataValidity(block); err != nil {
		return err
	}
	return nil
}

func (verifier *AccountVerifier) VerifyforRPC(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforRPC", time.Now())
	if err := verifier.VerifyTimeNotYet(block); err != nil {
		return nil, err
	}
	//if err := verifier.VerifyDataValidity(block); err != nil {
	//	return nil, err
	//}
	if verifyResult, stat := verifier.VerifyReferred(block); verifyResult != SUCCESS {
		if stat.errMsg != "" {
			return nil, errors.New(stat.errMsg)
		}
		return nil, errors.New("verify referred block failed")
	}
	return verifier.VerifyforVM(block)
}

// contractAddr's sendBlock don't call VerifyReferredforPool
func (verifier *AccountVerifier) VerifyReferred(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountReferredforPool", time.Now())

	stat := verifier.newVerifyStat()

	if !verifier.verifySnapshot(block, stat) {
		return stat.referredSnapshotResult, stat
	}

	if !verifier.verifySelf(block, stat) {
		return FAIL, stat
	}

	if !verifier.verifyFrom(block, stat) {
		return FAIL, stat
	}

	return stat.VerifyResult(), stat
}

func (verifier *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforVM", time.Now())

	var preHash *types.Hash
	if block.Height > 1 {
		preHash = &block.PrevHash
	}
	gen, err := generator.NewGenerator(verifier.chain, &block.SnapshotHash, preHash, &block.AccountAddress)
	if err != nil {
		verifier.log.Error("new generator error," + err.Error())
		return nil, ErrVerifyForVmGeneratorFailed
	}

	genResult, err := gen.GenerateWithBlock(block, nil)
	if err != nil {
		verifier.log.Error("generator block error," + err.Error())
		return nil, ErrVerifyForVmGeneratorFailed
	}
	if len(genResult.BlockGenList) == 0 {
		if genResult.Err != nil {
			verifier.log.Error(genResult.Err.Error())
			return nil, genResult.Err
		}
		return nil, errors.New("vm failed, blockList is empty")
	}
	if err := verifier.verifyVMResult(block, genResult.BlockGenList[0].AccountBlock); err != nil {
		return nil, errors.New(ErrVerifyWithVmResultFailed.Error() + "," + err.Error())
	}
	return genResult.BlockGenList, nil
}

// block from Net or Rpc doesn't have stateHash、Quota, so don't need to verify
func (verifier *AccountVerifier) verifyVMResult(origBlock *ledger.AccountBlock, genBlock *ledger.AccountBlock) error {
	if origBlock.BlockType != genBlock.BlockType {
		return errors.New("blockType")
	}
	if origBlock.ToAddress != genBlock.ToAddress {
		return errors.New("toAddress")
	}
	if origBlock.Fee.Cmp(genBlock.Fee) != 0 {
		return errors.New("fee")
	}
	if !bytes.Equal(origBlock.Data, genBlock.Data) {
		return errors.New("data")
	}
	if (origBlock.LogHash == nil && genBlock.LogHash != nil) || (origBlock.LogHash != nil && genBlock.LogHash == nil) {
		return errors.New("logHash")
	}
	if origBlock.LogHash != nil && genBlock.LogHash != nil && *origBlock.LogHash != *genBlock.LogHash {
		return errors.New("logHash")
	}
	return nil
}

func (verifier *AccountVerifier) verifySelf(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	if err := verifier.VerifyDataValidity(block); err != nil {
		verifyStatResult.referredSelfResult = FAIL
		verifyStatResult.errMsg += err.Error()
		return false
	}

	step1, err1 := verifier.verifyProducerLegality(block, verifyStatResult.accountTask)
	if isFail(step1, err1, verifyStatResult) {
		return false
	}
	step2, err2 := verifier.verifySelfPrev(block, verifyStatResult.accountTask)
	if isFail(step2, err2, verifyStatResult) {
		return false
	}

	if step1 == SUCCESS && step2 == SUCCESS {
		verifyStatResult.referredSelfResult = SUCCESS
	} else {
		verifyStatResult.accountTask = append(verifyStatResult.accountTask,
			&AccountPendingTask{Addr: &block.AccountAddress, Hash: &block.Hash})
		verifyStatResult.referredSelfResult = PENDING
	}
	return true
}

func (verifier *AccountVerifier) verifyFrom(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if block.IsReceiveBlock() {
		fromBlock, err := verifier.chain.GetAccountBlockByHash(&block.FromBlockHash)
		if fromBlock == nil {
			if err != nil {
				verifyStatResult.referredFromResult = FAIL
				verifyStatResult.errMsg += errors.New("func GetAccountBlockByHash failed").Error()
				return false
			}
			verifyStatResult.accountTask = append(verifyStatResult.accountTask,
				&AccountPendingTask{Addr: nil, Hash: &block.FromBlockHash})
			verifyStatResult.referredFromResult = PENDING
		} else {
			if verifier.VerifyIsReceivedSucceed(block) {
				//verifier.log.Info("hash", fromBlock.Hash, "addr", fromBlock.AccountAddress,
				//	"toAddr", fromBlock.ToAddress, "blockType", fromBlock.BlockType)
				verifyStatResult.referredFromResult = FAIL
				verifyStatResult.errMsg += errors.New("block is already received successfully").Error()
				return false
			}

			result, err := verifier.VerifySnapshotOfReferredBlock(block, fromBlock)
			if isFail(result, err, verifyStatResult) {
				return false
			}
			if result == PENDING {
				verifyStatResult.accountTask = append(verifyStatResult.accountTask,
					&AccountPendingTask{Addr: nil, Hash: &block.FromBlockHash})
			}
			verifyStatResult.referredFromResult = result
		}
	} else {
		verifyStatResult.referredFromResult = SUCCESS
	}
	return true
}

func (verifier *AccountVerifier) verifySelfPrev(block *ledger.AccountBlock, task []*AccountPendingTask) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())

	latestBlock, err := verifier.chain.GetLatestAccountBlock(&block.AccountAddress)
	if latestBlock == nil {
		if err != nil {
			return FAIL, errors.New("func GetLatestAccountBlock failed" + err.Error())
		} else {
			if block.Height == 1 {
				prevZero := &types.Hash{}
				if !bytes.Equal(block.PrevHash.Bytes(), prevZero.Bytes()) {
					return FAIL, errors.New("account first block's prevHash error")
				}
				return SUCCESS, nil
			}
			task = append(task, &AccountPendingTask{nil, &block.PrevHash})
			return PENDING, nil
		}
	} else {
		result, err := verifier.VerifySnapshotOfReferredBlock(block, latestBlock)
		if result == FAIL {
			return FAIL, err
		}
		switch {
		case block.PrevHash == latestBlock.Hash && block.Height == latestBlock.Height+1:
			return SUCCESS, nil
		case block.PrevHash != latestBlock.Hash && block.Height > latestBlock.Height+1:
			task = append(task, &AccountPendingTask{nil, &block.PrevHash})
			return PENDING, nil
		default:
			return FAIL, errors.New("preHash or height is invalid")
		}
	}
}

func (verifier *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, verifyStatResult *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())

	snapshotBlock, err := verifier.chain.GetSnapshotBlockByHash(&block.SnapshotHash)
	if snapshotBlock == nil {
		if err != nil {
			verifyStatResult.referredSnapshotResult = FAIL
			verifyStatResult.errMsg += errors.New("func GetSnapshotBlockByHash failed: " + err.Error()).Error()
			return false
		}
		verifyStatResult.snapshotTask = &SnapshotPendingTask{Hash: &block.SnapshotHash}
		verifyStatResult.referredSnapshotResult = PENDING
		return false
	} else {
		if err := verifier.VerifyTimeOut(snapshotBlock); err != nil {
			verifyStatResult.referredSnapshotResult = FAIL
			verifyStatResult.errMsg += err.Error()
			return false
		} else {
			verifyStatResult.referredSnapshotResult = SUCCESS
			return true
		}
	}
}

func (verifier *AccountVerifier) VerifySnapshotOfReferredBlock(thisBlock *ledger.AccountBlock, referredBlock *ledger.AccountBlock) (VerifyResult, error) {
	// referredBlock' snapshotBlock's height can't lower than thisBlock
	thisSnapshotBlock, _ := verifier.chain.GetSnapshotBlockByHash(&thisBlock.SnapshotHash)
	referredSnapshotBlock, _ := verifier.chain.GetSnapshotBlockByHash(&referredBlock.SnapshotHash)
	if referredSnapshotBlock != nil {
		if thisSnapshotBlock != nil {
			if referredSnapshotBlock.Height > thisSnapshotBlock.Height {
				return FAIL, ErrVerifySnapshotOfReferredBlockFailed
			} else {
				return SUCCESS, nil
			}
		}
		return PENDING, nil
	}
	return FAIL, ErrVerifySnapshotOfReferredBlockFailed
}

func (verifier *AccountVerifier) verifyProducerLegality(block *ledger.AccountBlock, task []*AccountPendingTask) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	code, err := verifier.chain.AccountType(&block.AccountAddress)
	if err != nil || code == ledger.AccountTypeError {
		return FAIL, err
	}
	if code == ledger.AccountTypeContract {
		if block.IsReceiveBlock() {
			if result, err := verifier.consensus.VerifyAccountProducer(block); !result {
				if err != nil {
					verifier.log.Error(err.Error())
				}
				return FAIL, errors.New("block producer is illegal")
			}
		}
	}
	if code == ledger.AccountTypeGeneral {
		if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
			return FAIL, errors.New("publicKey doesn't match with the accountAddress")
		}
	}
	return SUCCESS, nil
}

func (verifier *AccountVerifier) VerifyDataValidity(block *ledger.AccountBlock) error {
	defer monitor.LogTime("verify", "accountSelfDataValidity", time.Now())

	code, err := verifier.chain.AccountType(&block.AccountAddress)
	if err != nil || code == ledger.AccountTypeError {
		return errors.New("get account type error")
	}
	if block.IsSendBlock() && code == ledger.AccountTypeNotExist {
		return ErrVerifyAccountAddrFailed
	}

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	}
	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	}
	if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
		return errors.New("block amount out of bounds")
	}
	if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
		return errors.New("block fee out of bounds")
	}
	if block.Timestamp == nil {
		return errors.New("block timestamp can't be nil")
	}

	if err := verifier.VerifyHash(block); err != nil {
		return err
	}

	if err := verifier.VerifyNonce(block, code); err != nil {
		return err
	}

	if block.IsReceiveBlock() || (block.IsSendBlock() && code != ledger.AccountTypeContract) {
		if err := verifier.VerifySigature(block); err != nil {
			return err
		}
	}

	return nil
}

func (verifier *AccountVerifier) VerifyP2PDataValidity(block *ledger.AccountBlock) error {
	defer monitor.LogTime("verify", "accountSelfP2PDataValidity", time.Now())

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	}
	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	}
	if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
		return errors.New("block amount out of bounds")
	}
	if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
		return errors.New("block fee out of bounds")
	}
	if block.Timestamp == nil {
		return errors.New("block timestamp can't be nil")
	}

	if err := verifier.VerifyHash(block); err != nil {
		return err
	}

	if block.IsReceiveBlock() || (block.IsSendBlock() && (len(block.Signature) > 0 || len(block.PublicKey) > 0)) {
		if err := verifier.VerifySigature(block); err != nil {
			return err
		}
	}

	return nil
}

func (verifier *AccountVerifier) VerifyIsReceivedSucceed(block *ledger.AccountBlock) bool {
	return verifier.chain.IsSuccessReceived(&block.AccountAddress, &block.FromBlockHash)
}

func (verifier *AccountVerifier) VerifyHash(block *ledger.AccountBlock) error {
	computedHash := block.ComputeHash()
	if block.Hash.IsZero() {
		return errors.New("hash can't be allzero")
	}
	if computedHash != block.Hash {
		//verifier.log.Error("checkHash failed", "originHash", block.Hash, "computedHash", computedHash)
		return ErrVerifyHashFailed
	}
	return nil
}

func (verifier *AccountVerifier) VerifySigature(block *ledger.AccountBlock) error {
	if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
		return errors.New("signature or publicKey can't be nil")
	}
	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if !isVerified {
		if verifyErr != nil {
			verifier.log.Error("VerifySig failed", "error", verifyErr)
		}
		return ErrVerifySignatureFailed
	}
	return nil
}

func (verifier *AccountVerifier) VerifyNonce(block *ledger.AccountBlock, accountType uint64) error {
	if len(block.Nonce) != 0 {
		if accountType == ledger.AccountTypeContract {
			return errors.New("nonce of contractAddr's block must be nil")
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

func (verifier *AccountVerifier) VerifyTimeOut(blockReferSb *ledger.SnapshotBlock) error {
	currentSb := verifier.chain.GetLatestSnapshotBlock()
	if currentSb.Height > blockReferSb.Height+types.AccountLimitSnapshotHeight {
		return errors.New("snapshot timeout, height is too low")
	}
	return nil
}

func (verifier *AccountVerifier) VerifyTimeNotYet(block *ledger.AccountBlock) error {
	//  don't accept which timestamp doesn't satisfy within the (now + 1h) limit
	currentSb := time.Now()
	if block.Timestamp.After(currentSb.Add(time.Hour)) {
		return errors.New("block timestamp not arrive yet")
	}
	return nil
}

func isFail(result VerifyResult, err error, stat *AccountBlockVerifyStat) bool {
	if result == FAIL {
		stat.referredSelfResult = FAIL
		if err != nil {
			stat.errMsg += err.Error()
		}
		return true
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
