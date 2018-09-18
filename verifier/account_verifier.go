package verifier

import (
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
)

type AccountVerifier struct {
	reader ChainReader
}

func NewAccountVerifier() *AccountVerifier {
	// todo add reader
	verifier := &AccountVerifier{}
	return verifier
}
func (self *AccountVerifier) newVerifyStat(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	return &AccountBlockVerifyStat{}
}
func (self *AccountVerifier) verifyGenesis(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountGenesis", time.Now())
	return true
}
func (self *AccountVerifier) verifySelf(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSelf", time.Now())

	return false
}

func (self *AccountVerifier) verifyFrom(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountFrom", time.Now())
	return false
}

func (self *AccountVerifier) verifySnapshot(block *ledger.AccountBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())
	return false
}

func (self *AccountVerifier) VerifyReferred(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	defer monitor.LogTime("verify", "accountBlock", time.Now())
	stat := self.newVerifyStat(block)

	if self.verifyGenesis(block, stat) {
		return stat
	}

	// check snapshot
	if self.verifySnapshot(block, stat) {
		return stat
	}

	// check self
	if self.verifySelf(block, stat) {
		return stat
	}
	// check from
	if self.verifyFrom(block, stat) {
		return stat
	}
	return stat
}

func (self *AccountVerifier) VerifyProducer(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	defer monitor.LogTime("verify", "accountProducer", time.Now())
	stat := self.newVerifyStat(block)
	return stat
}
func (self *AccountVerifier) VerifyVM(block *ledger.AccountBlock) *AccountBlockVerifyStat {
	defer monitor.LogTime("verify", "accountVM", time.Now())
	stat := self.newVerifyStat(block)
	return stat
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	errMsg                 string
	accountTask            *AccountPendingTask
	snapshotTask           *SnapshotPendingTask
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
