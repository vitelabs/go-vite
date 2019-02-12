package verifier

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/math"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fork"
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
	chain     chain.Chain
	consensus Consensus

	log log15.Logger
}

func NewAccountVerifier(chain chain.Chain, consensus Consensus) *AccountVerifier {
	return &AccountVerifier{
		chain:     chain,
		consensus: consensus,

		log: log15.New("module", "AccountVerifier"),
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
	defer monitor.LogTime("AccountVerifier", "VerifyNetAb", time.Now())

	if err := verifier.VerifyDealTime(block); err != nil {
		return err
	}

	if err := verifier.VerifyP2PDataValidity(block); err != nil {
		return err
	}

	return nil
}

func (verifier *AccountVerifier) VerifyforRPC(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("AccountVerifier", "accountVerifyforRPC", time.Now())

	if err := verifier.VerifyDealTime(block); err != nil {
		return nil, err
	}

	if verifyResult, stat := verifier.VerifyReferred(block); verifyResult != SUCCESS {
		if stat.errMsg != "" {
			return nil, errors.New(stat.errMsg)
		}
		return nil, errors.New("verify referred block failed")
	}

	return verifier.VerifyforVM(block)
}

type BlockState struct {
	block   *ledger.AccountBlock
	accType uint64

	sbHeight uint64 // fork

	vStat *AccountBlockVerifyStat
}

// contractAddr's sendBlock don't call VerifyReferredforPool
func (verifier *AccountVerifier) VerifyReferred(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("AccountVerifier", "accountVerifyReferred", time.Now())

	bs := &BlockState{
		block:    block,
		sbHeight: 2,
		accType:  ledger.AccountTypeNotExist,
		vStat:    verifier.newVerifyStat(),
	}

	if !verifier.verifySnapshot(bs) {
		return bs.vStat.referredSnapshotResult, bs.vStat
	}

	if !verifier.checkAccAddressType(bs) {
		return bs.vStat.referredSelfResult, bs.vStat
	}

	if !verifier.verifySelf(bs) {
		return bs.vStat.referredSelfResult, bs.vStat
	}

	if !verifier.verifyFrom(bs) {
		return bs.vStat.referredFromResult, bs.vStat
	}

	return bs.vStat.VerifyResult(), bs.vStat
}

func (verifier *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (blocks []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("AccountVerifier", "VerifyforVM", time.Now())
	vLog := verifier.log.New("method", "VerifyforVM")

	var preHash *types.Hash
	if block.Height > 1 {
		preHash = &block.PrevHash
	}
	gen, err := generator.NewGenerator(verifier.chain, &block.SnapshotHash, preHash, &block.AccountAddress)
	if err != nil {
		vLog.Error("new generator error," + err.Error())
		return nil, ErrVerifyForVmGeneratorFailed
	}

	genResult, err := gen.GenerateWithBlock(block, nil)
	if err != nil {
		vLog.Error("generator block error," + err.Error())
		return nil, ErrVerifyForVmGeneratorFailed
	}
	if len(genResult.BlockGenList) == 0 {
		if genResult.Err != nil {
			errInf := fmt.Sprintf("sbHash %v, addr %v,", block.SnapshotHash, block.AccountAddress)
			if block.IsReceiveBlock() {
				errInf += fmt.Sprintf("fromHash %v", block.FromBlockHash)
			}
			vLog.Error(genResult.Err.Error(), "block:", errInf)
			return nil, genResult.Err
		}
		return nil, errors.New("vm failed, blockList is empty")
	}

	if err := verifier.verifyVMResult(block, genResult.BlockGenList[0].AccountBlock); err != nil {
		return nil, errors.New(ErrVerifyWithVmResultFailed.Error() + "," + err.Error())
	}
	return genResult.BlockGenList, nil
}

// referredBlock' snapshotBlock's sbHeight can't lower than thisBlock
func (verifier *AccountVerifier) VerifySnapshotOfReferredBlock(thisBlock *ledger.AccountBlock, referredBlock *ledger.AccountBlock) (VerifyResult, error) {
	thisSnapshotBlock, _ := verifier.chain.GetSnapshotBlockHeadByHash(&thisBlock.SnapshotHash)
	referredSnapshotBlock, _ := verifier.chain.GetSnapshotBlockHeadByHash(&referredBlock.SnapshotHash)
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
	if currentSb.Height > blockReferSb.Height+TimeOutHeight {
		return errors.New("snapshot timeout, sbHeight is too low")
	}
	return nil
}

//  don't accept which timestamp doesn't satisfy within the (now + 1h) limit
func (verifier *AccountVerifier) VerifyDealTime(block *ledger.AccountBlock) error {
	currentSb := time.Now()
	if block.Timestamp.After(currentSb.Add(time.Hour)) {
		return errors.New("block timestamp is too far in the future, not arrive yet")
	}
	return nil
}

func (verifier *AccountVerifier) VerifyProducerLegality(block *ledger.AccountBlock, accType uint64) error {
	defer monitor.LogTime("AccountVerifier", "verifyProducerLegality", time.Now())

	if accType == ledger.AccountTypeContract {
		if block.IsReceiveBlock() {
			if result, err := verifier.consensus.VerifyAccountProducer(block); !result {
				if err != nil {
					verifier.log.Error(err.Error())
				}
				return errors.New("block producer is illegal")
			}
		}
	}
	if accType == ledger.AccountTypeGeneral {
		if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
			return errors.New("publicKey doesn't match with the accountAddress")
		}
	}

	return nil
}

func (verifier *AccountVerifier) VerifyDataValidity(block *ledger.AccountBlock, sbHeight uint64, accType uint64) error {
	defer monitor.LogTime("AccountVerifier", "VerifyDataValidity", time.Now())

	if err := verifier.verifyDatasIntergrity(block, sbHeight); err != nil {
		return err
	}

	if fork.IsSmartFork(sbHeight) {
		if block.IsReceiveBlock() && block.Data != nil && accType == ledger.AccountTypeGeneral {
			return errors.New("receiveBlock data must be nil when addr is general")
		}
	}

	if err := verifier.VerifyHash(block); err != nil {
		return err
	}

	if err := verifier.VerifyNonce(block, accType); err != nil {
		return err
	}

	if block.IsReceiveBlock() || (block.IsSendBlock() && accType != ledger.AccountTypeContract) {
		if err := verifier.VerifySigature(block); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *AccountVerifier) VerifyP2PDataValidity(block *ledger.AccountBlock) error {
	defer monitor.LogTime("AccountVerifier", "VerifyP2PDataValidity", time.Now())

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	} else {
		if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
			return errors.New("block amount out of bounds")
		}
	}

	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	} else {
		if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
			return errors.New("block fee out of bounds")
		}
	}

	if block.Timestamp == nil {
		return errors.New("block timestamp can't be nil")
	}

	if err := verifier.VerifyHash(block); err != nil {
		return err
	}

	if verifier.chain.IsGenesisAccountBlock(block) {
		return nil
	}

	if block.IsReceiveBlock() || (block.IsSendBlock() && (len(block.Signature) > 0 || len(block.PublicKey) > 0)) {
		if err := verifier.VerifySigature(block); err != nil {
			return err
		}
	}

	return nil
}

// verifySnapshot's result as PENDING or FAIL all return false,
// judge whether to trigger the fork logic according to the snapshotBlock referred
func (verifier *AccountVerifier) verifySnapshot(bs *BlockState) bool {
	defer monitor.LogTime("AccountVerifier", "verifySnapshot", time.Now())

	snapshotBlock, err := verifier.chain.GetSnapshotBlockHeadByHash(&bs.block.SnapshotHash)
	if snapshotBlock == nil {
		if err != nil {
			bs.vStat.errMsg += errors.New("func GetSnapshotBlockByHash failed: " + err.Error()).Error()
			bs.vStat.referredSnapshotResult = FAIL
			return false
		}
		bs.vStat.snapshotTask = &SnapshotPendingTask{Hash: &bs.block.SnapshotHash}
		bs.vStat.referredSnapshotResult = PENDING
		return false
	} else {
		if err := verifier.VerifyTimeOut(snapshotBlock); err != nil {
			bs.vStat.errMsg += err.Error()
			bs.vStat.referredSnapshotResult = FAIL
			return false
		} else {
			bs.sbHeight = snapshotBlock.Height
			bs.vStat.referredSnapshotResult = SUCCESS
			return true
		}
	}
}

func (verifier *AccountVerifier) verifySelf(bs *BlockState) bool {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	if err := verifier.VerifyDataValidity(bs.block, bs.sbHeight, bs.accType); err != nil {
		bs.vStat.referredSelfResult = FAIL
		bs.vStat.errMsg += err.Error()
		return false
	}

	if err := verifier.VerifyProducerLegality(bs.block, bs.accType); err != nil {
		bs.vStat.referredSelfResult = FAIL
		bs.vStat.errMsg += err.Error()
		return false
	}

	bs.vStat.referredSelfResult = verifier.verifySelfPrev(bs)
	if bs.vStat.referredSelfResult == FAIL {
		return false
	}
	if bs.vStat.referredSelfResult == PENDING {
		bs.vStat.accountTask = append(bs.vStat.accountTask,
			&AccountPendingTask{Addr: &bs.block.AccountAddress, Hash: &bs.block.Hash})
	}
	return true
}

func (verifier *AccountVerifier) verifySelfPrev(bs *BlockState) VerifyResult {
	defer monitor.LogTime("AccountVerifier", "verifySelfPrev", time.Now())

	latestBlock, err := verifier.chain.GetLatestAccountBlock(&bs.block.AccountAddress)
	if latestBlock == nil {
		if err != nil {
			bs.vStat.errMsg += "func GetLatestAccountBlock failed" + err.Error()
			bs.vStat.referredSelfResult = FAIL
			return FAIL
		} else {
			if bs.block.Height == 1 {
				prevZero := &types.Hash{}
				if !bytes.Equal(bs.block.PrevHash.Bytes(), prevZero.Bytes()) {
					bs.vStat.errMsg += "account first block's prevHash error"
					bs.vStat.referredSelfResult = FAIL
					return FAIL
				}
				return SUCCESS
			}
			bs.vStat.accountTask = append(bs.vStat.accountTask, &AccountPendingTask{nil, &bs.block.PrevHash})
			bs.vStat.referredSelfResult = PENDING
			return PENDING
		}
	} else {
		if _, err := verifier.VerifySnapshotOfReferredBlock(bs.block, latestBlock); err != nil {
			bs.vStat.errMsg += err.Error()
			bs.vStat.referredSelfResult = FAIL
			return FAIL
		}
		switch {
		case bs.block.PrevHash == latestBlock.Hash && bs.block.Height == latestBlock.Height+1:
			return SUCCESS
		case bs.block.PrevHash != latestBlock.Hash && bs.block.Height > latestBlock.Height+1:
			bs.vStat.accountTask = append(bs.vStat.accountTask, &AccountPendingTask{nil, &bs.block.PrevHash})
			bs.vStat.referredSelfResult = PENDING
			return PENDING
		default:
			bs.vStat.errMsg += "preHash or sbHeight is invalid"
			bs.vStat.referredSelfResult = FAIL
			return FAIL
		}
	}
}

func (verifier *AccountVerifier) verifyFrom(bs *BlockState) bool {
	defer monitor.LogTime("AccountVerifier", "verifyFrom", time.Now())

	if bs.block.IsReceiveBlock() {
		fromBlock, err := verifier.chain.GetAccountBlockByHash(&bs.block.FromBlockHash)
		if fromBlock == nil {
			if err != nil {
				bs.vStat.errMsg += errors.New("func GetAccountBlockByHash failed").Error()
				bs.vStat.referredFromResult = FAIL
				return false
			}
			bs.vStat.accountTask = append(bs.vStat.accountTask,
				&AccountPendingTask{Addr: nil, Hash: &bs.block.FromBlockHash})
			bs.vStat.referredFromResult = PENDING
		} else {
			if verifier.VerifyIsReceivedSucceed(bs.block) {
				verifier.log.Debug(fmt.Sprintf("sendBlock: hash=%v, addr=%v, toAddr=%v",
					fromBlock.Hash, fromBlock.AccountAddress, fromBlock.ToAddress), "method", "VerifyIsReceivedSucceed")
				bs.vStat.errMsg += "block is already received successfully"
				bs.vStat.referredFromResult = FAIL
				return false
			}

			result, err := verifier.VerifySnapshotOfReferredBlock(bs.block, fromBlock)
			bs.vStat.referredFromResult = result
			if result == FAIL {
				if err != nil {
					bs.vStat.errMsg += err.Error()
				}
				return false
			}
			if result == PENDING {
				bs.vStat.accountTask = append(bs.vStat.accountTask,
					&AccountPendingTask{Addr: nil, Hash: &bs.block.FromBlockHash})
			}
		}
	} else {
		bs.vStat.referredFromResult = SUCCESS
	}
	return true
}

func (verifier *AccountVerifier) verifyDatasIntergrity(block *ledger.AccountBlock, vite1Height uint64) error {
	if block.Timestamp == nil {
		return errors.New("block timestamp can't be nil")
	}

	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	} else {
		if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
			return errors.New("block amount out of bounds")
		}
	}

	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	} else {
		if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
			return errors.New("block fee out of bounds")
		}
	}

	if fork.IsSmartFork(vite1Height) {
		if block.IsReceiveBlock() {
			if block.Amount != nil && block.Amount.Cmp(big.NewInt(0)) != 0 {
				return errors.New("block amount can't be anything other than nil or 0 ")
			}
			if block.Fee != nil && block.Fee.Cmp(big.NewInt(0)) != 0 {
				return errors.New("block fee can't be anything other than nil or 0")
			}
			if block.TokenId != types.ZERO_TOKENID {
				return errors.New("block TokenId can't be anything other than ZERO_TOKENID")
			}
		}
	}

	return nil
}

// block from Net or Rpc doesn't have stateHash、Quota, so don't need to verify
func (verifier *AccountVerifier) verifyVMResult(origBlock *ledger.AccountBlock, genBlock *ledger.AccountBlock) error {
	if origBlock.Hash != genBlock.Hash {
		return errors.New("hash")
	}
	//if origBlock.BlockType != genBlock.BlockType {
	//	return errors.New("blockType")
	//}
	//if origBlock.ToAddress != genBlock.ToAddress {
	//	return errors.New("toAddress")
	//}
	//if origBlock.Fee.Cmp(genBlock.Fee) != 0 {
	//	return errors.New("fee")
	//}
	//if !bytes.Equal(origBlock.Data, genBlock.Data) {
	//	return errors.New("data")
	//}
	//if (origBlock.LogHash == nil && genBlock.LogHash != nil) || (origBlock.LogHash != nil && genBlock.LogHash == nil) {
	//	return errors.New("logHash")
	//}
	//if origBlock.LogHash != nil && genBlock.LogHash != nil && *origBlock.LogHash != *genBlock.LogHash {
	//	return errors.New("logHash")
	//}

	return nil
}

func (verifier *AccountVerifier) checkAccAddressType(bs *BlockState) bool {
	var eLog = verifier.log.New("picker", "send1")
	var accErr error
	bs.vStat.referredSelfResult = SUCCESS

	bs.accType, accErr = verifier.chain.AccountType(&bs.block.AccountAddress)
	if accErr != nil || bs.accType == ledger.AccountTypeError {
		bs.vStat.referredSelfResult = FAIL
		bs.vStat.errMsg += "get account type error"
		return false
	}

	if bs.block.Height == 1 && bs.block.IsSendBlock() {
		eLog.Info(fmt.Sprintf("hash:%v, addr:%v, toAddr:%v", bs.block.Hash, bs.block.AccountAddress, bs.block.ToAddress))
	}

	if bs.accType == ledger.AccountTypeNotExist {
		if bs.block.Height == 1 {
			if bs.block.IsSendBlock() {
				if !fork.IsSmartFork(bs.sbHeight) {
					eLog.Info(fmt.Sprintf("solve addr:%v", bs.block.AccountAddress))
					bs.accType = ledger.AccountTypeGeneral
					return true
				}
				bs.vStat.referredSelfResult = FAIL
				bs.vStat.errMsg += ErrVerifyAccountAddrFailed.Error()
				return false
			}
			if sendBlock, _ := verifier.chain.GetAccountBlockByHash(&bs.block.FromBlockHash); sendBlock != nil {
				if sendBlock.BlockType == ledger.BlockTypeSendCreate {
					bs.accType = ledger.AccountTypeContract
				} else {
					bs.accType = ledger.AccountTypeGeneral
				}
				return true
			}
			bs.vStat.referredSelfResult = PENDING
			bs.vStat.accountTask = append(bs.vStat.accountTask,
				&AccountPendingTask{nil, &bs.block.FromBlockHash},
				&AccountPendingTask{Addr: &bs.block.AccountAddress, Hash: &bs.block.Hash})
			return false
		} else {
			bs.vStat.referredSelfResult = PENDING
			bs.vStat.accountTask = append(bs.vStat.accountTask, &AccountPendingTask{Addr: &bs.block.AccountAddress, Hash: &bs.block.Hash})
			return false
		}
	}
	return true
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
