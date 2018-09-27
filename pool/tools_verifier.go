package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
	v verifier.AccountVerifier
}

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

func (self *accountVerifier) verifyNormalAccount(b commonBlock) *poolAccountVerifyStat {
	//block := b.(*accountPoolBlock)
	//result, stat := self.accountVerifier.VerifyforProducer(block.block)
	//
	//this, otherBlocks, e := self.accountVerifier.VerifyforVM(block.block)
	return &poolAccountVerifyStat{}
}

func (self *accountVerifier) verifyContractAccount(received *accountPoolBlock, sends []*accountPoolBlock) *poolAccountVerifyStat {
	//block := b.(*accountPoolBlock)
	//result, stat := self.accountVerifier.VerifyforProducer(block.block)
	//
	//this, otherBlocks, e := self.accountVerifier.VerifyforVM(block.block)
	return &poolAccountVerifyStat{}
}

func (self *accountVerifier) verifyTimeout(block *ledger.AccountBlock) *poolAccountVerifyStat {
	//block := b.(*accountPoolBlock)
	//result, stat := self.accountVerifier.VerifyforProducer(block.block)
	//
	//this, otherBlocks, e := self.accountVerifier.VerifyforVM(block.block)
	return &poolAccountVerifyStat{}
}

type poolVerifyStat struct {
}

func (self *poolVerifyStat) verifyResult() verifier.VerifyResult {
	return verifier.SUCCESS
}
func (self *poolVerifyStat) errMsg() string {
	return ""
}

func (self *poolVerifyStat) task() verifyTask {
	return nil
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
