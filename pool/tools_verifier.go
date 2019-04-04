package pool

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"

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
	v   verifier.Verifier
	log log15.Logger
}

func (self *accountVerifier) verifyAccountData(b *ledger.AccountBlock) error {
	//if err := self.v.VerifyNetAb(b); err != nil {
	//	return err
	//}
	return nil
}

/**
if b is contract send block, result must be FAIL.
*/
func (self *accountVerifier) verifyAccount(b *accountPoolBlock, latest *ledger.SnapshotBlock) *poolAccountVerifyStat {
	result := &poolAccountVerifyStat{}
	// todo how to fix for stat

	task, blocks, err := self.v.VerifyPoolAccBlock(b.block, &latest.Hash)
	if err != nil {
		result.err = err
		result.result = verifier.FAIL
		return result
	}
	if task != nil {
		result.result = verifier.PENDING
		result.taskList = task
		return result
	}
	//result.stat =

	if blocks != nil {
		var bs []*accountPoolBlock
		bs = append(bs, newAccountPoolBlock(blocks.AccountBlock, blocks.VmDb, b.v, b.source))
		result.blocks = bs
		result.result = verifier.SUCCESS
		return result
	}
	result.result = verifier.FAIL
	msg := fmt.Sprintf("error verify result. %s-%s-%d", b.block.AccountAddress, b.Hash(), b.Height())
	result.err = errors.New(msg)
	return result
}
func (self *accountVerifier) newSuccessTask() verifyTask {
	return successT
}

func (self *accountVerifier) newFailTask() verifyTask {
	return &failTask{t: time.Now()}
}

func (self *accountVerifier) verifyDirectAccount(received *accountPoolBlock, latestSb *ledger.SnapshotBlock) (result *poolAccountVerifyStat) {
	result = self.verifyAccount(received, latestSb)
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

	for _, v := range self.taskList.AccountTask {
		result = append(result, fetchRequest{snapshot: false, chain: v.Addr, hash: *v.Hash, prevCnt: 1})
	}
	return &accountTask{result: result, t: time.Now()}
}

type poolAccountVerifyStat struct {
	blocks   []*accountPoolBlock
	result   verifier.VerifyResult
	taskList *verifier.AccBlockPendingTask
	err      error
}

func (self *poolAccountVerifyStat) verifyResult() verifier.VerifyResult {
	return self.result
}

var successT = &successTask{}

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
	if time.Now().After(self.t.Add(time.Millisecond * 200)) {
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
			block, _ := c.GetSnapshotHeaderByHash(v.hash)
			if block == nil {
				return false
			}
		} else {
			block, _ := c.GetAccountBlockByHash(v.hash)
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
