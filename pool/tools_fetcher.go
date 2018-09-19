package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/viteshan/naive-vite/syncer"
)

type commonSyncer interface {
	fetch(hashHeight commonHashHeight, prevCnt uint64)
}

type accountSyncer struct {
	address types.Address
	fetcher syncer.Fetcher
}

func (self *accountSyncer) broadcastBlock(block *ledger.AccountBlock) {
	panic("implement me")
}
func (self *accountSyncer) broadcastBlocks(block []*ledger.AccountBlock) {
	panic("implement me")
}

func (self *accountSyncer) broadcastReceivedBlocks(received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) {
	panic("implement me")
}

func (self *accountSyncer) fetch(hashHeight commonHashHeight, prevCnt uint64) {
	//self.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: self.address})
}

type snapshotSyncer struct {
	fetcher syncer.Fetcher
}

func (self *snapshotSyncer) broadcastBlock(block *ledger.SnapshotBlock) {
	panic("implement me")
}

func (self *snapshotSyncer) fetch(hashHeight commonHashHeight, prevCnt uint64) {
	//self.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: self.address})
}
