package verifier

import (
	"math/big"

	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
)

type SnapshotVerifier struct {
	reader ChainReader
}

func NewSnapshotVerifier() *SnapshotVerifier {
	// todo add chain chainReader
	verifier := &SnapshotVerifier{}
	return verifier
}

func (self *SnapshotVerifier) verifySelf(block *ledger.SnapshotBlock, stat *SnapshotBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "snapshotSelf", time.Now())

	return false
}

func (self *SnapshotVerifier) verifyAccounts(block *ledger.SnapshotBlock, stat *SnapshotBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "snapshotAccounts", time.Now())

	return false
}

func (self *SnapshotVerifier) VerifyReferred(block *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
	defer monitor.LogTime("verify", "snapshotBlock", time.Now())
	stat := self.newVerifyStat(block)

	if self.verifySelf(block, stat) {
		return stat
	}

	if self.verifyAccounts(block, stat) {
		return stat
	}

	return stat
}
func (self *SnapshotVerifier) VerifyProducer(block *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
	defer monitor.LogTime("verify", "snapshotProducer", time.Now())
	stat := self.newVerifyStat(block)
	return stat
}

type AccountHashH struct {
	Addr   *types.Address
	Hash   *types.Hash
	Height *big.Int
}

type SnapshotBlockVerifyStat struct {
	result       VerifyResult
	results      map[types.Address]VerifyResult
	errMsg       string
	accountTasks []*AccountPendingTask
	snapshotTask *SnapshotPendingTask
}

func (self *SnapshotBlockVerifyStat) AccountTasks() []*AccountPendingTask {
	return nil
}
func (self *SnapshotBlockVerifyStat) SnapshotTask() []*SnapshotPendingTask {
	return nil
}

func (self *SnapshotBlockVerifyStat) ErrMsg() string {
	return self.errMsg
}

func (self *SnapshotBlockVerifyStat) VerifyResult() VerifyResult {
	return self.result
}

func (self *SnapshotVerifier) newVerifyStat(b *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
	// todo init account hashH
	stat := &SnapshotBlockVerifyStat{result: PENDING}
	stat.results = make(map[types.Address]VerifyResult)
	return stat
}
