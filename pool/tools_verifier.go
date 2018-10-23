package pool

import (
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/verifier"
)

type verifyTask interface {
	done(c chainDb) bool
	requests() []fetchRequest
}

type sverifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyReferred(block *ledger.SnapshotBlock) *verifier.SnapshotBlockVerifyStat
	VerifyTimeout(nowHeight uint64, referHeight uint64) bool
}

type snapshotVerifier struct {
	v sverifier
}

func (self *snapshotVerifier) verifySnapshotData(block *ledger.SnapshotBlock) error {
	if err := self.v.VerifyNetSb(block); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *snapshotVerifier) verifySnapshot(block *snapshotPoolBlock) *poolSnapshotVerifyStat {
	result := &poolSnapshotVerifyStat{}
	stat := self.v.VerifyReferred(block.block)
	result.results = stat.Results()
	result.result = stat.VerifyResult()
	result.msg = stat.ErrMsg()
	return result
}
func (self *snapshotVerifier) verifyAccountTimeout(current *ledger.SnapshotBlock, refer *ledger.SnapshotBlock) bool {
	return self.v.VerifyTimeout(current.Height, refer.Height)
}

type accountVerifier struct {
	v   *verifier.AccountVerifier
	log log15.Logger
}

func (self *accountVerifier) verifyAccountData(b *ledger.AccountBlock) error {
	if err := self.v.VerifyNetAb(b); err != nil {
		return err
	}
	return nil
}

/**
if b is contract send block, result must be FAIL.
*/
func (self *accountVerifier) verifyAccount(b *accountPoolBlock) *poolAccountVerifyStat {
	result := &poolAccountVerifyStat{}
	// todo how to fix for stat
	verifyResult, stat := self.v.VerifyReferred(b.block)
	result.result = verifyResult
	result.stat = stat

	switch verifyResult {
	case verifier.SUCCESS:

		blocks, err := self.v.VerifyforVM(b.block)
		if err != nil {
			result.result = verifier.FAIL
			result.err = err
			return result
		}
		var bs []*accountPoolBlock
		for _, v := range blocks {
			bs = append(bs, newAccountPoolBlock(v.AccountBlock, v.VmContext, b.v))
		}
		result.blocks = bs
		return result
	case verifier.PENDING:
		// todo
		return result
	case verifier.FAIL:
		return result
	}

	return result
}
func (self *accountVerifier) newSuccessTask() verifyTask {
	return successT
}

func (self *accountVerifier) newFailTask() verifyTask {
	return failT
}

func (self *accountVerifier) verifyDirectAccount(received *accountPoolBlock, sends []*accountPoolBlock) (result *poolAccountVerifyStat) {
	result = self.verifyAccount(received)
	if result.result == verifier.SUCCESS {
		if len(result.blocks) != len(sends)+1 {
			self.log.Error(fmt.Sprintf("account verify fail. received:%s.", received.Hash()))
			result.result = verifier.FAIL
			return
		}
		for i, b := range sends {
			if b.Hash() != result.blocks[i+1].Hash() {
				self.log.Error(fmt.Sprintf("account verify fail. received:%s, send:%s, %s.", received.Hash(), b.Hash(), result.blocks[i+1].Hash()))
				result.result = verifier.FAIL
				return
			}
		}
	}
	return
}

type poolSnapshotVerifyStat struct {
	results map[types.Address]verifier.VerifyResult
	result  verifier.VerifyResult
	task    verifyTask
	msg     string
}

func (self *poolSnapshotVerifyStat) verifyResult() verifier.VerifyResult {
	return self.result
}
func (self *poolSnapshotVerifyStat) errMsg() string {
	return self.msg
}
func (self *poolAccountVerifyStat) task() verifyTask {
	var result []fetchRequest
	taskA, taskB := self.stat.GetPendingTasks()
	for _, v := range taskA {
		result = append(result, fetchRequest{snapshot: false, chain: v.Addr, hash: *v.Hash, prevCnt: 1})
	}

	if taskB != nil {
		result = append(result, fetchRequest{snapshot: true, hash: *taskB.Hash, prevCnt: 1})
	}
	return &accountTask{result: result, t: time.Now()}
}

type poolAccountVerifyStat struct {
	blocks []*accountPoolBlock
	result verifier.VerifyResult
	stat   *verifier.AccountBlockVerifyStat
	err    error
}

func (self *poolAccountVerifyStat) verifyResult() verifier.VerifyResult {
	return self.result
}
func (self *poolAccountVerifyStat) errMsg() string {

	if self.err != nil {
		return self.err.Error()
	} else if self.stat.ErrMsg() == "" {
		return "has no err msg."
	} else {
		return self.stat.ErrMsg()
	}

}

var successT = &successTask{}
var failT = &failTask{}

type successTask struct {
}

func (self *successTask) done(c chainDb) bool {
	return true
}

func (*successTask) requests() []fetchRequest {
	return nil
}

type failTask struct {
	t time.Time
}

func (self *failTask) done(c chainDb) bool {
	if time.Now().After(self.t.Add(time.Second * 3)) {
		return true
	}
	return false
}

func (*failTask) requests() []fetchRequest {
	return nil
}

type accountTask struct {
	result []fetchRequest
	t      time.Time
}

func (self *accountTask) done(c chainDb) bool {
	if time.Now().After(self.t.Add(time.Second * 5)) {
		return true
	}
	for _, v := range self.result {
		if v.snapshot {
			block, _ := c.GetSnapshotBlockByHash(&v.hash)
			if block == nil {
				return false
			}
		} else {
			block, _ := c.GetAccountBlockByHash(&v.hash)
			if block == nil {
				return false
			}
		}
	}
	return true
}

func (self *accountTask) requests() []fetchRequest {
	return self.result
}
