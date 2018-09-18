package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/viteshan/naive-vite/common/face"
)

type verifyTask interface {
	done() bool
	requests() []face.FetchRequest
}
type commonVerifier interface {
	verify(b commonBlock) *poolVerifyStat
}

type snapshotVerifier struct {
}

func (self *snapshotVerifier) verify(b commonBlock) *poolVerifyStat {
	//block := b.(*accountPoolBlock)
	//result, stat := self.accountVerifier.VerifyforProducer(block.block)
	//
	//this, otherBlocks, e := self.accountVerifier.VerifyforVM(block.block)
	return &poolVerifyStat{}
}
func (self *snapshotVerifier) verifySnapshot(block *snapshotPoolBlock) *poolSnapshotVerifyStat {
	return &poolSnapshotVerifyStat{}
}

type accountVerifier struct {
}

func (self *accountVerifier) verify(b commonBlock) *poolVerifyStat {
	//block := b.(*accountPoolBlock)
	//result, stat := self.accountVerifier.VerifyforProducer(block.block)
	//
	//this, otherBlocks, e := self.accountVerifier.VerifyforVM(block.block)
	return &poolVerifyStat{}
}

func (self *accountVerifier) verifyAccount(block *accountPoolBlock) *poolAccountVerifyStat {
	return &poolAccountVerifyStat{}
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

func (self *accountVerifier) verifyContractAccount(b commonBlock) *poolAccountVerifyStat {
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
}

func (self *poolAccountVerifyStat) verifyResult() verifier.VerifyResult {
	return verifier.SUCCESS
}
func (self *poolAccountVerifyStat) errMsg() string {
	return ""
}
