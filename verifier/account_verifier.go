package verifier

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

const (
	TimeOut             = float64(48)
	MaxRecvTypeCount    = 1
	MaxRecvErrTypeCount = 3
)

type AccountVerifier struct {
	chainReader     ChainReader
	committeeReader ProducerReader

	log log15.Logger
}

func NewAccountVerifier(chain ChainReader, committee ProducerReader) *AccountVerifier {
	verifier := &AccountVerifier{
		chainReader:     chain,
		committeeReader: committee,
		log:             nil,
	}
	verifier.log = log15.New("")
	return verifier
}

func (self *AccountVerifier) newVerifyStat(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}

func (self *AccountVerifier) verifyGenesis(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountGenesis", time.Now())
	//fixme ask Liz for the GetGenesisBlockFirst() and GetGenesisBlockSecond()
	return block.PrevHash.Bytes() == nil &&
		bytes.Equal(block.Signature, self.chainReader.GetGenesisBlockFirst().Signature) &&
		bytes.Equal(block.Hash.Bytes(), self.chainReader.GetGenesisBlockFirst().Hash.Bytes())
}

func (self *AccountVerifier) verifySelf(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	stat1 := self.VerifySelfValidity(block)
	stat2 := self.VerifySelfDependence(block)

	select {
	case stat1 == PENDING && stat2 == PENDING:
		stat.referredSelfResult = PENDING
		return true
	case stat1 == SUCCESS && stat2 == SUCCESS:
		stat.referredSelfResult = SUCCESS
		return true
	default:
		stat.referredSelfResult = FAIL
		return false
	}
}

func (self *AccountVerifier) verifyFrom(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		fromBlock, err := self.chainReader.GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil || fromBlock == nil {
			self.log.Error("verifyFrom.GetAccountBlockByHash", "error", err)
			stat.referredFromResult = FAIL
			return false
		} else {
			stat.referredFromResult = SUCCESS
			return true
		}
	} else {
		self.log.Info("verifyFrom: send doesn't have fromBlock")
		stat.referredFromResult = FAIL
		return false
	}
}

func (self *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())

	snapshotBlock, err := self.chainReader.GetSnapshotBlockByHash(&block.SnapshotHash)
	if err != nil || snapshotBlock == nil {
		self.log.Error("verifySnapshot.GetSnapshotBlockByHash", "error", err)
		return false
	}

	if !self.VerifyTimeOut(block) && self.VerifyConfirmed(block) {
		stat.referredSnapshotResult = SUCCESS
		return true
	} else {
		stat.referredSnapshotResult = FAIL
		return false
	}
}

func (self *AccountVerifier) VerifySelfValidity(block *ledger.AccountBlock) VerifyResult {
	defer monitor.LogTime("verify", "accountSelfValidity", time.Now())

	computedHash := block.GetComputeHash()
	if block.Hash.Bytes() == nil || !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		self.log.Error("VerifySelfValidity: CheckHash failed", "Hash", block.Hash.String())
		return FAIL
	}

	if gid, _ := self.chainReader.GetContractGid(&block.AccountAddress); gid != nil {
		if (block.BlockType == ledger.BlockTypeSendCall || block.BlockType == ledger.BlockTypeSendCreate) &&
			block.Signature == nil {
			return PENDING
		}
		if err := self.committeeReader.VerifyAccountProducer(block); err != nil {
			self.log.Error("VerifySelfValidity: the block producer is illegal")
			return FAIL
		}

	}

	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if verifyErr != nil || !isVerified {
		self.log.Error("VerifySelfValidity.VerifySig failed", verifyErr)
		return FAIL
	}
	return SUCCESS
}

func (self *AccountVerifier) VerifySelfDependence(block *ledger.AccountBlock) VerifyResult {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())

	latestBlock, err := self.chainReader.GetLatestAccountBlock(&block.AccountAddress)
	if err != nil {
		self.log.Error("VerifySelfDependence.GetLatestAccountBlock failed", "error", err)
		return FAIL
	}

	var isContractSend bool
	if gid, _ := self.chainReader.GetContractGid(&block.AccountAddress); gid != nil &&
		(block.BlockType == ledger.BlockTypeSendCall || block.BlockType == ledger.BlockTypeSendCreate) {
		isContractSend = true
	} else {
		isContractSend = false
	}

	if block.Height == latestBlock.Height+1 && block.PrevHash == latestBlock.Hash {
		return SUCCESS
	}
	if isContractSend == true && block.Height > latestBlock.Height+1 && block.PrevHash != latestBlock.Hash {
		return PENDING
	}

	self.log.Error("VerifySelfDependence: PreHash or Height is invalid")
	return FAIL
}

func (self *AccountVerifier) VerifyTimeOut(block *ledger.AccountBlock) bool {
	defer monitor.LogTime("verify", "accountSnapshotTimeout", time.Now())

	currentTime := time.Now()
	if currentTime.Sub(*block.Timestamp).Hours() > TimeOut {
		self.log.Error("snapshot time out of limit")
		return false
	}
	return true
}

func (self *AccountVerifier) VerifyConfirmed(block *ledger.AccountBlock) bool {
	defer monitor.LogTime("verify", "accountConfirmed", time.Now())

	if confirmedBlock, err := self.chainReader.GetConfirmBlock(block); confirmedBlock == nil || err != nil {
		self.log.Error("not Confirmed yet")
		return false
	}
	return true
}

func (self *AccountVerifier) VerifyReceiveReachLimit(sendBlock *ledger.AccountBlock) bool {
	recvTime := self.chainReader.GetReceiveTimes(&sendBlock.ToAddress, &sendBlock.Hash)

	if sendBlock.BlockType == ledger.BlockTypeReceiveError && recvTime >= MaxRecvErrTypeCount {
		return true
	}
	if sendBlock.BlockType == ledger.BlockTypeReceive && recvTime >= MaxRecvTypeCount {
		return true
	}

	return false
}

func (self *AccountVerifier) VerifyUnconfirmedPriorBlockReceived(priorBlockHash *types.Hash) VerifyResult {
	existBlock, err := self.chainReader.GetAccountBlockByHash(priorBlockHash)
	if err != nil || existBlock == nil {
		self.log.Info("VerifyUnconfirmedPriorBlockReceived: the prev block hasn't existed in Chain. ")
		return PENDING
	}
	return SUCCESS
}

func (self *AccountVerifier) VerifyforProducer(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	defer monitor.LogTime("verify", "accountProducer", time.Now())

	stat := self.newVerifyStat(block)

	self.verifySelf(block, stat)
	self.verifyFrom(block, stat)
	self.verifySnapshot(block, stat)

	if stat.VerifyResult() == PENDING {
		stat.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
		stat.snapshotTask = &SnapshotPendingTask{
			Hash: &block.SnapshotHash,
		}
	}

	return stat
}

func (self *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) bool {
	result := self.VerifySelfDependence(block)
	if result != FAIL {
		return true
	}
	return false
}

func (self *AccountVerifier) VerifyforP2P(block *ledger.AccountBlock) bool {
	result := self.VerifySelfValidity(block)
	if result != FAIL {
		return true
	}
	return false
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	accountTask            *AccountPendingTask
	snapshotTask           *SnapshotPendingTask
	//errMsg                 string
}

//func (self *AccountBlockVerifyStat) ErrMsg() string {
//	return self.errMsg
//}

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
