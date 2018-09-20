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
	TimeOutHeight       = uint64(24 * 30 / 3600)
	MaxRecvTypeCount    = 1
	MaxRecvErrTypeCount = 3
	MaxBigIntLen        = 256
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
		log:             nil,
	}
	verifier.log = log15.New("")
	return verifier
}

func (self *AccountVerifier) newVerifyStat(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}

// fixme contractAddr's sendBlock don't call VerifyReferredforPool
func (self *AccountVerifier) VerifyReferredforPool(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountReferredforPool", time.Now())

	stat := self.newVerifyStat(block)

	self.verifySelf(block, stat)
	self.verifyFrom(block, stat)
	self.verifySnapshot(block, stat)

	return stat.VerifyResult(), stat
}

func (self *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (this *vm_context.VmAccountBlock, others []*vm_context.VmAccountBlock, err error) {
	defer monitor.LogTime("verify", "VerifyforVM", time.Now())

	gen := generator.NewGenerator(self.chain.Chain(), self.signer)
	if err = gen.PrepareVm(&block.SnapshotHash, &block.PrevHash, &block.AccountAddress); err != nil {
		return nil, nil, err
	}
	genResult := gen.GenerateWithP2PBlock(block, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return gen.Sign(addr, nil, data)
	})
	if genResult == nil {
		return nil, nil, errors.New("GenerateWithP2PBlock failed")
	}

	this = genResult.BlockGenList[0]
	if len(genResult.BlockGenList) > 1 {
		others = genResult.BlockGenList[1:]
	}
	return this, others, nil
}

func (self *AccountVerifier) VerifyforP2P(block *ledger.AccountBlock) bool {
	if result, err := self.verifySelfDataValidity(block); result != FAIL && err == nil {
		return true
	}
	return false
}

func (self *AccountVerifier) VerifyforRPC() ([]*vm_context.VmAccountBlock, error) {
	// todo 1.arg to be message or block
	// todo 2.generateBlock
	return nil, nil
}

func (self *AccountVerifier) verifySelf(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	stat1, err1 := self.verifyProducerLegality(block)
	stat2, err2 := self.verifySelfDataValidity(block)
	stat3, err3 := self.verifySelfDependence(block)

	select {
	case stat1 == FAIL || stat2 == FAIL || stat3 == FAIL:
		if err1 != nil {
			stat.errMsg += err1.Error()
		}
		if err2 != nil {
			stat.errMsg += err2.Error()
		}
		if err3 != nil {
			stat.errMsg += err3.Error()
		}
		stat.referredSelfResult = FAIL
	case stat1 == SUCCESS && stat2 == SUCCESS && stat3 == SUCCESS:
		stat.referredSelfResult = SUCCESS
	default:
		stat.referredSelfResult = PENDING
		stat.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
	}
}

func (self *AccountVerifier) verifyProducerLegality(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	var errMsg error
	if self.verifyIsContractAddress(&block.AccountAddress) {
		if self.verifyIsReceiveBlock(block) {
			// contractAddr receiveBlock
			if block.PublicKey == nil {
				errMsg = errors.New("VerifyIsProducerLegals: block.PublicKey of contractAddr's receiveBlock can't be nil")
				self.log.Error(errMsg.Error())
				return FAIL, errMsg
			} else {
				if conErr := self.consensusReader.VerifyAccountProducer(block); errMsg != nil {
					errMsg = errors.New("verifySelfDataValidity: the block producer is illegal")
					self.log.Error(errMsg.Error(), "error", conErr)
					return FAIL, errMsg
				}
				return SUCCESS, nil
			}
		} else {
			// contractAddr sendBlock
			if block.Signature == nil && block.PublicKey == nil {
				return PENDING, nil
			} else {
				errMsg = errors.New("verifySelfDataValidity: Signature and PublicKey of the contractAddress's sendBlock must be nil")
				self.log.Error(errMsg.Error())
				return FAIL, errMsg
			}
		}
	} else {
		// commonAddr
		if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
			errMsg = errors.New("verifySelfDataValidity: PublicKey match AccountAddress failed")
			self.log.Error(errMsg.Error())
			return FAIL, errMsg
		} else {
			return SUCCESS, nil
		}
	}
}

func (self *AccountVerifier) verifyFrom(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if self.verifyIsReceiveBlock(block) {
		fromBlock, err := self.chain.Chain().GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil || fromBlock == nil {
			self.log.Info("verifyFrom.GetAccountBlockByHash", "error", err)
			stat.accountTask = &AccountPendingTask{
				Addr:   nil,
				Hash:   &block.FromBlockHash,
				Height: 0,
			}
			stat.referredFromResult = PENDING
		} else {
			stat.referredFromResult = SUCCESS
		}
	} else {
		self.log.Info("verifyFrom: send doesn't have fromBlock")
		stat.referredFromResult = SUCCESS
	}
}

func (self *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())

	snapshotBlock, err := self.chain.Chain().GetSnapshotBlockByHash(&block.SnapshotHash)
	if err != nil || snapshotBlock == nil {
		self.log.Info("verifySnapshot.GetSnapshotBlockByHash failed", "error", err)
		stat.referredSnapshotResult = PENDING
	} else {
		stat.referredSnapshotResult = SUCCESS
	}

	verifyResult, err := self.VerifyTimeOut(snapshotBlock)
	if err != nil {
		self.log.Error(err.Error())
	}

	select {
	case verifyResult == FAIL:
		stat.errMsg += err.Error()
		stat.referredSnapshotResult = FAIL
	case stat.referredSnapshotResult == SUCCESS && verifyResult == SUCCESS:
		stat.referredSnapshotResult = SUCCESS
	default:
		stat.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
		stat.referredSnapshotResult = PENDING
	}
}

func (self *AccountVerifier) verifySelfDataValidity(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDataValidity", time.Now())

	var errMsg error
	isContractAddr := self.verifyIsContractAddress(&block.AccountAddress)

	if !self.verifyBlockIntegrity(block, isContractAddr) {
		return FAIL, errors.New("verifySelfDataValidity.verifyBlockIntegrity failed.")
	}

	if block.Amount.Sign() < 0 || block.Amount.BitLen() > MaxBigIntLen {
		return FAIL, errors.New("block.Amount out of bounds")
	}
	if block.Fee.Sign() < 0 || block.Amount.BitLen() > MaxBigIntLen {
		return FAIL, errors.New("block.Fee out of bounds")
	}

	computedHash := block.GetComputeHash()
	if block.Hash.Bytes() == nil || !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		errMsg = errors.New("verifySelfDataValidity: CheckHash failed")
		self.log.Error(errMsg.Error(), "Hash", block.Hash.String())
		return FAIL, errMsg
	}

	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if verifyErr != nil || !isVerified {
		errMsg = errors.New("verifySelfDataValidity.VerifySig failed")
		self.log.Error(errMsg.Error(), "error", verifyErr)
		return FAIL, errMsg
	}

	return SUCCESS, nil
}

func (self *AccountVerifier) verifySelfDependence(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())
	var errMsg error

	latestBlock, err := self.chain.Chain().GetLatestAccountBlock(&block.AccountAddress)
	if err != nil || latestBlock == nil {
		errMsg = errors.New("verifySelfDependence.GetLatestAccountBlock failed")
		self.log.Error(errMsg.Error(), "error", err)
		return FAIL, errMsg
	}

	if block.Height == latestBlock.Height+1 && block.PrevHash == latestBlock.Hash {
		return SUCCESS, nil
	}
	if block.Height > latestBlock.Height+1 && block.PrevHash != latestBlock.Hash {
		return PENDING, nil
	}
	errMsg = errors.New("verifySelfDependence: PreHash or Height is invalid")
	self.log.Error(errMsg.Error())
	return FAIL, errMsg
}

func (self *AccountVerifier) VerifyChainInsertQualification(block *ledger.AccountBlock) bool {
	return false
}

func (self *AccountVerifier) verifyBlockIntegrity(block *ledger.AccountBlock, isContractAddr bool) bool {
	if block.Amount == nil {
		block.Amount = big.NewInt(0)
	}
	if block.Fee == nil {
		block.Fee = big.NewInt(0)
	}
	if block.Timestamp == nil || block.Data == nil || block.Signature == nil ||
		(block.LogHash == nil && isContractAddr && self.verifyIsReceiveBlock(block)) {
		return false
	}
	return true
}

func (self *AccountVerifier) VerifyTimeOut(blockReferSb *ledger.SnapshotBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSnapshotTimeout", time.Now())

	currentSb, err := self.chain.Chain().GetLatestSnapshotBlock()
	if err != nil || currentSb == nil {
		errMsg := errors.New("VerifyTimeOut.GetLatestSnapshotBlock failed")
		self.log.Error(errMsg.Error(), "error", err)
		return PENDING, errMsg
	}
	if currentSb.Height > blockReferSb.Height+TimeOutHeight {
		errMsg := errors.New("VerifyTimeOut: snapshot time out of limit")
		self.log.Error(errMsg.Error())
		return FAIL, errMsg
	}
	return SUCCESS, nil
}

func (self *AccountVerifier) verifyIsContractAddress(addr *types.Address) bool {
	gid, err := self.chain.Chain().GetContractGid(addr)
	if err != nil {
		self.log.Error("verifyIsContractAddress.GetContractGid", "Error", err)
	}
	if gid != nil {
		return true
	}
	return false
}

func (self *AccountVerifier) verifyIsReceiveBlock(block *ledger.AccountBlock) bool {
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
