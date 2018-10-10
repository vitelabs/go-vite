package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type commonSyncer interface {
	fetch(hashHeight ledger.HashHeight, prevCnt uint64)
}

type accountSyncer struct {
	address types.Address
	fetcher syncer
}

func (self *accountSyncer) broadcastBlock(block *ledger.AccountBlock) {
	self.fetcher.BroadcastAccountBlock(self.address, block)
}
func (self *accountSyncer) broadcastBlocks(blocks []*ledger.AccountBlock) {
	self.fetcher.BroadcastAccountBlocks(self.address, blocks)
}

func (self *accountSyncer) broadcastReceivedBlocks(received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) {
	var blocks []*ledger.AccountBlock

	blocks = append(blocks, received.AccountBlock)
	for _, b := range sendBlocks {
		blocks = append(blocks, b.AccountBlock)
	}
}

func (self *accountSyncer) fetch(hashHeight ledger.HashHeight, prevCnt uint64) {
	self.fetcher.FetchAccountBlocks(hashHeight.Hash, prevCnt, &self.address)
}
func (self *accountSyncer) fetchByHash(hash types.Hash, prevCnt uint64) {
	self.fetcher.FetchAccountBlocks(hash, prevCnt, &self.address)
}

type snapshotSyncer struct {
	fetcher syncer
}

func (self *snapshotSyncer) broadcastBlock(block *ledger.SnapshotBlock) {
	self.fetcher.BroadcastSnapshotBlock(block)
}

func (self *snapshotSyncer) fetch(hashHeight ledger.HashHeight, prevCnt uint64) {
	self.fetcher.FetchSnapshotBlocks(hashHeight.Hash, prevCnt)
}

func (self *snapshotSyncer) fetchByHash(hash types.Hash, prevCnt uint64) {
	self.fetcher.FetchSnapshotBlocks(hash, prevCnt)
}
