package verifier

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
	"time"
)

const (
	TimeOut             = float64(48)
	MaxRecvTypeCount    = 1
	MaxRecvErrTypeCount = 3
)

type AccountVerifier struct {
	chainReader     ChainReader
	consensusReader ConsensusReader

	log log15.Logger
}

func NewAccountVerifier(chain ChainReader, consensus ConsensusReader) *AccountVerifier {
	verifier := &AccountVerifier{
		chainReader:     chain,
		consensusReader: consensus,
		log:             nil,
	}
	verifier.log = log15.New("")
	return verifier
}

func (self *AccountVerifier) newVerifyStat(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}

func (self *AccountVerifier) VerifyforProducer(block *ledger.AccountBlock) (VerifyResult, *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountProducer", time.Now())

	stat := self.newVerifyStat(block)

	if !self.VerifyIsProducerLegal(block) {
		stat.referredSnapshotResult = FAIL
		stat.referredSelfResult = FAIL
		stat.referredFromResult = FAIL
		stat.accountTask = nil
		stat.snapshotTask = nil
	} else {
		self.verifySelf(block, stat)
		self.verifyFrom(block, stat)
		self.verifySnapshot(block, stat)

	}
	return stat.VerifyResult(), stat
}

func (self *AccountVerifier) VerifyforVM(block *ledger.AccountBlock) (this *vm_context.VmAccountBlock, others []*vm_context.VmAccountBlock, err error) {
	//if result, err := self.VerifySelfDependence(block); result != FAIL && err == nil {
	//}
	// todo add generatorP2PTx
	return nil, nil, nil
}

func (self *AccountVerifier) VerifyforRPC() ([]*vm_context.VmAccountBlock, error) {
	// todo 1.arg to be message or block
	// todo 2.generateBlock
	return nil, nil
}

func (self *AccountVerifier) VerifyforP2P(block *ledger.AccountBlock) bool {
	if result, err := self.VerifySelfDataValidity(block); result != FAIL && err == nil {
		return true
	}
	return false
}

func (self *AccountVerifier) verifyGenesis(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountGenesis", time.Now())
	//fixme ask Liz for the GetGenesisBlockFirst() and GetGenesisBlockSecond()
	return block.PrevHash.Bytes() == nil &&
		bytes.Equal(block.Signature, self.chainReader.GetGenesisBlockFirst().Signature) &&
		bytes.Equal(block.Hash.Bytes(), self.chainReader.GetGenesisBlockFirst().Hash.Bytes())
}

func (self *AccountVerifier) verifySelf(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	stat1, err1 := self.VerifySelfDataValidity(block)
	stat2, err2 := self.VerifySelfDependence(block)
	if err1 != nil || err2 != nil {
		stat.errMsg += err1.Error() + err2.Error()
	}

	select {
	case stat1 == PENDING && stat2 == PENDING:
		stat.referredSelfResult = PENDING
		stat.accountTask = &AccountPendingTask{
			Addr:   &block.AccountAddress,
			Hash:   &block.Hash,
			Height: block.Height,
		}
	case stat1 == SUCCESS && stat2 == SUCCESS:
		stat.referredSelfResult = SUCCESS
	default:
		stat.referredSelfResult = FAIL
	}
}

func (self *AccountVerifier) verifyFrom(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) {
	defer monitor.LogTime("verify", "accountFrom", time.Now())

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		fromBlock, err := self.chainReader.GetAccountBlockByHash(&block.FromBlockHash)
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

	snapshotBlock, err := self.chainReader.GetSnapshotBlockByHash(&block.SnapshotHash)
	if err != nil || snapshotBlock == nil {
		self.log.Info("verifySnapshot.GetSnapshotBlockByHash", "error", err)
		stat.snapshotTask = &SnapshotPendingTask{
			Hash: &block.SnapshotHash,
		}
		stat.referredFromResult = PENDING
	}

	if !self.VerifyTimeOut(block) && self.VerifyConfirmed(block) {
		stat.referredSnapshotResult = SUCCESS
	} else {
		stat.errMsg += errors.New("verify Snapshot timeout or prevBlock still not confirmed").Error()
		self.log.Error(stat.errMsg)
		stat.referredSnapshotResult = FAIL
	}
}

func (self *AccountVerifier) VerifyBlockIntegrity(block *ledger.AccountBlock) bool {
	return false
}

func (self *AccountVerifier) VerifySelfDataValidity(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfValidity", time.Now())
	var errMsg error
	computedHash := block.GetComputeHash()
	if block.Hash.Bytes() == nil || !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		errMsg = errors.New("VerifySelfDataValidity: CheckHash failed")
		self.log.Error(errMsg.Error(), "Hash", block.Hash.String())
		return FAIL, errMsg
	}

	gid, _ := self.chainReader.GetContractGid(&block.AccountAddress)
	if gid != nil && (block.BlockType == ledger.BlockTypeSendCall || block.BlockType == ledger.BlockTypeSendCreate) {
		if block.Signature == nil && block.PublicKey == nil {
			return PENDING, nil
		} else {
			errMsg = errors.New("VerifySelfDataValidity: Signature and PublicKey of the contractAddress's sendBlock must be nil")
			self.log.Error(errMsg.Error())
			return FAIL, errMsg
		}
	}

	if types.PubkeyToAddress(block.PublicKey) != block.AccountAddress {
		errMsg = errors.New("VerifySelfDataValidity: PublicKey match AccountAddress failed")
		self.log.Error(errMsg.Error())
		return FAIL, errMsg
	}

	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if verifyErr != nil || !isVerified {
		errMsg = errors.New("VerifySelfDataValidity.VerifySig failed")
		self.log.Error(errMsg.Error(), "error", verifyErr)
		return FAIL, errMsg
	}
	return SUCCESS, nil
}

func (self *AccountVerifier) VerifySelfDependence(block *ledger.AccountBlock) (VerifyResult, error) {
	defer monitor.LogTime("verify", "accountSelfDependence", time.Now())
	var errMsg error

	latestBlock, err := self.chainReader.GetLatestAccountBlock(&block.AccountAddress)
	if err != nil {
		errMsg = errors.New("VerifySelfDependence.GetLatestAccountBlock failed")
		self.log.Error(errMsg.Error(), "error", err)
		return FAIL, errMsg
	}

	var isContractSend bool
	gid, _ := self.chainReader.GetContractGid(&block.AccountAddress)
	if gid != nil && (block.BlockType == ledger.BlockTypeSendCall || block.BlockType == ledger.BlockTypeSendCreate) {
		isContractSend = true
	} else {
		isContractSend = false
	}

	if block.Height == latestBlock.Height+1 && block.PrevHash == latestBlock.Hash {
		return SUCCESS, nil
	}
	if isContractSend == true && block.Height > latestBlock.Height+1 && block.PrevHash != latestBlock.Hash {
		return PENDING, nil
	}
	errMsg = errors.New("VerifySelfDependence: PreHash or Height is invalid")
	self.log.Error(errMsg.Error())
	return FAIL, errMsg
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

	if confirmedBlock := self.chainReader.GetConfirmBlock(block); confirmedBlock == nil {
		self.log.Error("not Confirmed yet")
		return false
	}
	//if confirmedBlock, err := self.chainReader.GetConfirmBlock(block); confirmedBlock == nil || err != nil {
	//	self.log.Error("not Confirmed yet")
	//	return false
	//}
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

func (self *AccountVerifier) VerifyUnconfirmedPriorBlockReceived(priorBlockHash *types.Hash) bool {
	existBlock, err := self.chainReader.GetAccountBlockByHash(priorBlockHash)
	if err != nil || existBlock == nil {
		self.log.Info("VerifyUnconfirmedPriorBlockReceived: the prev block hasn't existed in Chain. ")
		return false
	}
	return true
}

func (self *AccountVerifier) VerifyIsProducerLegal(block *ledger.AccountBlock) bool {
	if err := self.consensusReader.VerifyAccountProducer(block); err != nil {
		self.log.Error("VerifySelfDataValidity: the block producer is illegal")
		return false
	}
	return true
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	accountTask            *AccountPendingTask
	snapshotTask           *SnapshotPendingTask
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
