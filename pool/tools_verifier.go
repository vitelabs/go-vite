package pool

import (
	"fmt"

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
		result.block = newAccountPoolBlock(blocks.AccountBlock, blocks.VmDb, b.v, b.source)
		result.result = verifier.SUCCESS
		return result
	}
	result.result = verifier.FAIL
	msg := fmt.Sprintf("error verify result. %s-%s-%d", b.block.AccountAddress, b.Hash(), b.Height())
	result.err = errors.New(msg)
	return result
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

type poolAccountVerifyStat struct {
	block    *accountPoolBlock
	result   verifier.VerifyResult
	taskList *verifier.AccBlockPendingTask
	err      error
}

func (self *poolAccountVerifyStat) verifyResult() verifier.VerifyResult {
	return self.result
}
