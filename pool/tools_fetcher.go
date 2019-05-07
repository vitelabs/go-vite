package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
)

type commonSyncer interface {
	fetch(hashHeight ledger.HashHeight, prevCnt uint64)
}

type accountSyncer struct {
	address types.Address
	fetcher syncer
	log     log15.Logger
}

func (accSyn *accountSyncer) broadcastBlock(block *ledger.AccountBlock) {
	accSyn.fetcher.BroadcastAccountBlock(block)
}
func (accSyn *accountSyncer) broadcastBlocks(blocks []*ledger.AccountBlock) {
	accSyn.fetcher.BroadcastAccountBlocks(blocks)
}

func (accSyn *accountSyncer) broadcastReceivedBlocks(received *vm_db.VmAccountBlock, sendBlocks []*vm_db.VmAccountBlock) {
	var blocks []*ledger.AccountBlock

	blocks = append(blocks, received.AccountBlock)
	for _, b := range sendBlocks {
		blocks = append(blocks, b.AccountBlock)
	}
	accSyn.fetcher.BroadcastAccountBlocks(blocks)
}

func (accSyn *accountSyncer) fetch(hashHeight ledger.HashHeight, prevCnt uint64) {
	if hashHeight.Height > 0 {
		if prevCnt > 100 {
			prevCnt = 100
		}
		accSyn.log.Debug("fetch account block", "height", hashHeight.Height, "hash", hashHeight.Hash, "prevCnt", prevCnt)
		accSyn.fetcher.FetchAccountBlocks(hashHeight.Hash, prevCnt, &accSyn.address)
	}
}
func (accSyn *accountSyncer) fetchBySnapshot(hashHeight ledger.HashHeight, account types.Address, prevCnt uint64, sHeight uint64, sHash types.Hash) {
	if hashHeight.Height > 0 {
		accSyn.log.Debug("fetch account block", "height", hashHeight.Height, "address", account, "hash", hashHeight.Hash, "prevCnt", prevCnt, "sHeight", sHeight, "sHash", sHash)
		accSyn.fetcher.FetchAccountBlocks(hashHeight.Hash, prevCnt, &accSyn.address)
	}
}
func (accSyn *accountSyncer) fetchByHash(hash types.Hash, prevCnt uint64) {
	accSyn.fetcher.FetchAccountBlocks(hash, prevCnt, &accSyn.address)
}

type snapshotSyncer struct {
	fetcher syncer
	log     log15.Logger
}

func (sSync *snapshotSyncer) broadcastBlock(block *ledger.SnapshotBlock) {
	sSync.fetcher.BroadcastSnapshotBlock(block)
}

func (sSync *snapshotSyncer) fetch(hashHeight ledger.HashHeight, prevCnt uint64) {
	if hashHeight.Height > 0 {
		if prevCnt > 100 {
			prevCnt = 100
		}
		sSync.log.Debug("fetch snapshot block", "height", hashHeight.Height, "hash", hashHeight.Hash, "prevCnt", prevCnt)
		sSync.fetcher.FetchSnapshotBlocks(hashHeight.Hash, prevCnt)
	}
}

func (sSync *snapshotSyncer) fetchByHash(hash types.Hash, prevCnt uint64) {
	sSync.fetcher.FetchSnapshotBlocks(hash, prevCnt)
}
