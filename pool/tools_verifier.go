package pool

import (
	"fmt"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/viteshan/naive-vite/common/face"
)

type verifyTask interface {
	done() bool
	requests() []face.FetchRequest
}

type snapshotVerifier struct {
	v verifier.SnapshotVerifier
}

func (self *snapshotVerifier) verifySnapshot(block *snapshotPoolBlock) (result *poolSnapshotVerifyStat) {
	stat := self.v.VerifyReferred(block.block)
	result.results = stat.Results()
	return
}
func (self *snapshotVerifier) verifyAccountTimeout(current *ledger.SnapshotBlock, refer *ledger.SnapshotBlock) bool {
	return self.v.VerifyTimeout(current.Height, refer.Height)
}

type accountVerifier struct {
	v   verifier.AccountVerifier
	log log15.Logger
}

/**
if b is contract send block, result must be FAIL.
*/
func (self *accountVerifier) verifyAccount(b *accountPoolBlock) (result *poolAccountVerifyStat) {
	// todo how to fix for stat
	verifyResult, _ := self.v.VerifyReferred(b.block)
	result.result = verifyResult
	switch verifyResult {
	case verifier.SUCCESS:
		blocks, err := self.v.VerifyforVM(b.block)
		if err != nil {
			result.result = verifier.FAIL
			return
		}
		var bs []*accountPoolBlock
		for _, v := range blocks {
			bs = append(bs, newAccountPoolBlock(v.AccountBlock, v.VmContext, b.v))
		}
		result.blocks = bs
		return
	case verifier.PENDING:
		return
	case verifier.FAIL:
		return
	}

	return
}
func (self *accountVerifier) newSuccessTask() verifyTask {
	return nil
}

func (self *accountVerifier) newFailTask() verifyTask {
	return nil
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
}

func (self *poolSnapshotVerifyStat) verifyResult() verifier.VerifyResult {
	return verifier.SUCCESS
}
func (self *poolSnapshotVerifyStat) errMsg() string {
	return ""
}
func (self *poolAccountVerifyStat) task() verifyTask {
	return nil
}

type poolAccountVerifyStat struct {
	blocks []*accountPoolBlock
	result verifier.VerifyResult
}

func (self *poolAccountVerifyStat) verifyResult() verifier.VerifyResult {
	return verifier.SUCCESS
}
func (self *poolAccountVerifyStat) errMsg() string {
	return ""
}
