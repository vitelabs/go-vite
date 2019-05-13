package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool/batch"
	"github.com/vitelabs/go-vite/pool/tree"
	"github.com/vitelabs/go-vite/vite/net"
)

// ChainState represents the relationship between the two branches
type ChainState uint8

const (
	// DISCONNECT means that two branches are not connected
	DISCONNECT ChainState = 1
	// CONNECTED means that two branches are connected
	CONNECTED = 2
	// FORKED means that two branches are connected, but forked
	FORKED = 3
)

func (pl *pool) insertChunks(chunks *net.Chunk) bool {
	source := chunks.Source
	hashes := chunks.HashMap
	snapshot := chunks.SnapshotRange
	head := *snapshot[1]
	tail := *snapshot[0]

	state := pl.checkSnapshotInsert(head, tail, hashes)
	if state == FORKED {
		pl.insertChunksToPool(chunks.SnapshotChunks, source)
		pl.PopDownloadedChunks(head)
		return false
	}
	if state == DISCONNECT {
		return false
	}

	accountRange := chunks.AccountRange
	state = pl.checkAccountsInsert(accountRange, hashes)
	if state == FORKED {
		pl.insertChunksToPool(chunks.SnapshotChunks, source)
		pl.PopDownloadedChunks(head)
		return false
	}
	if state == DISCONNECT {
		// todo
		return false
	}

	pl.LockInsert()
	defer pl.UnLockInsert()
	state = pl.checkSnapshotInsert(head, tail, hashes)
	if state != CONNECTED {
		return false
	}
	state = pl.checkAccountsInsert(accountRange, hashes)
	if state != CONNECTED {
		return false
	}
	err := pl.insertChunksToChain(chunks.SnapshotChunks, source)
	if err != nil {
		pl.log.Error("insert chunks fail.", "err", err)
		pl.PopDownloadedChunks(head)
		return false
	}
	pl.log.Info("insert chunks success.", "headHeight", head.Height, "headHash", head.Hash, "tailHeight", tail.Height, "tailHash", tail.Hash)
	pl.PopDownloadedChunks(head)
	return true
}

// insert chunks to blocks pool
func (pl *pool) insertChunksToPool(chunks []ledger.SnapshotChunk, source types.BlockSource) {
	for _, v := range chunks {
		if v.AccountBlocks != nil {
			for _, vv := range v.AccountBlocks {
				pl.AddAccountBlock(vv.AccountAddress, vv, source)
			}
		}
		if v.SnapshotBlock != nil {
			pl.AddSnapshotBlock(v.SnapshotBlock, source)
		}
	}
}

// insert chunks to chain, ignore blocks pool and snippet and tree
func (pl *pool) insertChunksToChain(chunks []ledger.SnapshotChunk, source types.BlockSource) error {
	b := batch.NewBatch(pl.snapshotExists, pl.accountExists, pl.version.Val(), 50)
	for _, v := range chunks {
		if v.AccountBlocks != nil {
			for _, vv := range v.AccountBlocks {
				if err := pl.accountExists(vv.Hash); err == nil {
					pl.log.Info("[A]block exist, ignore.", "block", vv.Hash)
					continue
				}
				block := newAccountPoolBlock(vv, nil, pl.version, source)
				pl.log.Info("[A]add block to batch.", "account", vv.AccountAddress, "height", vv.Height, "block", vv.Hash, "batchId", b.Id())
				err := b.AddItem(block)
				if err != nil && err == batch.MAX_ERROR {
					err := b.Batch(pl.insertSnapshotBucketForChunks, pl.insertAccountsBucketForChunks)
					if err != nil {
						return err
					}
					b = batch.NewBatch(pl.snapshotExists, pl.accountExists, pl.version.Val(), 50)
					err = b.AddItem(block)
					if err != nil {
						return err
					}
					continue
				}
				if err != nil {
					return err
				}
			}
		}
		if v.SnapshotBlock != nil {
			if err := pl.snapshotExists(v.SnapshotBlock.Hash); err == nil {
				pl.log.Info("[S]block exist, ignore.", "block", v.SnapshotBlock.Hash)
				continue
			}

			block := newSnapshotPoolBlock(v.SnapshotBlock, pl.version, source)
			pl.log.Info("[S]add block to batch.", "block", v.SnapshotBlock.Hash, "batchId", b.Id())
			err := b.AddItem(block)
			if err != nil && err == batch.MAX_ERROR {
				err := b.Batch(pl.insertSnapshotBucketForChunks, pl.insertAccountsBucketForChunks)
				if err != nil {
					return err
				}
				b = batch.NewBatch(pl.snapshotExists, pl.accountExists, pl.version.Val(), 50)
				err = b.AddItem(block)
				if err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
		}
	}

	if b.Size() > 0 {
		return b.Batch(pl.insertSnapshotBucketForChunks, pl.insertAccountsBucketForChunks)
	}
	return nil
}

func (pl *pool) insertSnapshotBucketForChunks(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return pl.insertSnapshotBucket(p, bucket, version)
}

func (pl *pool) insertAccountsBucketForChunks(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return pl.insertAccountBucket(p, bucket, version)
}

func (pl *pool) checkSnapshotInsert(headHH ledger.HashHeight, tailHH ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	cur := pl.pendingSc.CurrentChain()

	return pl.checkInsert(cur, headHH, tailHH, hashes)
}

func (pl *pool) checkAccountsInsert(minAddrs map[types.Address][2]*ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	for k, v := range minAddrs {
		cur := pl.selfPendingAc(k).CurrentChain()
		result := pl.checkInsert(cur, *v[1], *v[0], hashes)
		if result != CONNECTED {
			return result
		}
	}
	return CONNECTED
}

func (pl *pool) checkInsert(branch tree.Branch, waitingHeadH, waitingTailH ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	curHeight, curHash := branch.HeadHH()
	if waitingTailH.Height == curHeight {
		if waitingTailH.Hash == curHash {
			return CONNECTED
		}
		return FORKED
	}
	if curHeight > waitingTailH.Height {
		if curHeight >= waitingHeadH.Height {
			if branch.ContainsKnot(waitingHeadH.Height, waitingHeadH.Hash, true) {
				return CONNECTED
			}
			return FORKED
		}

		if _, ok := hashes[curHash]; ok {
			return CONNECTED
		}
		return FORKED
	}
	return DISCONNECT
}
