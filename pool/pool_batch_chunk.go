package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool/batch"
	"github.com/vitelabs/go-vite/pool/tree"
	"github.com/vitelabs/go-vite/vite/net"
)

type ChainState uint8

const (
	DISCONNECT ChainState = 1
	CONNECTED             = 2
	FORKED                = 3
)

func (self *pool) insertChunks(chunks *net.Chunk) bool {
	source := chunks.Source
	hashes := chunks.HashMap
	snapshot := chunks.SnapshotRange
	head := *snapshot[1]
	tail := *snapshot[0]

	state := self.checkSnapshotInsert(head, tail, hashes)
	if state == FORKED {
		self.insertChunksToPool(chunks.SnapshotChunks, source)
		self.PopDownloadedChunks(head)
		return false
	}
	if state == DISCONNECT {
		return false
	}

	accountRange := chunks.AccountRange
	state = self.checkAccountsInsert(accountRange, hashes)
	if state == FORKED {
		self.insertChunksToPool(chunks.SnapshotChunks, source)
		self.PopDownloadedChunks(head)
		return false
	}
	if state == DISCONNECT {
		return false
	}

	self.LockInsert()
	defer self.UnLockInsert()
	state = self.checkSnapshotInsert(head, tail, hashes)
	if state != CONNECTED {
		return false
	}
	state = self.checkAccountsInsert(accountRange, hashes)
	if state != CONNECTED {
		return false
	}
	err := self.insertChunksToChain(chunks.SnapshotChunks, source)
	if err != nil {
		self.log.Error("insert chunks fail.", "err", err)
		self.PopDownloadedChunks(head)
		return false
	} else {
		self.log.Info("insert chunks success.", "headHeight", head.Height, "headHash", head.Hash, "tailHeight", tail.Height, "tailHash", tail.Hash)
		self.PopDownloadedChunks(head)
		return true
	}
}

// insert chunks to blocks pool
func (self *pool) insertChunksToPool(chunks []ledger.SnapshotChunk, source types.BlockSource) {
	for _, v := range chunks {
		if v.AccountBlocks != nil {
			for _, vv := range v.AccountBlocks {
				self.AddAccountBlock(vv.AccountAddress, vv, source)
			}
		}
		if v.SnapshotBlock != nil {
			self.AddSnapshotBlock(v.SnapshotBlock, source)
		}
	}
}

// insert chunks to chain, ignore blocks pool and snippet and tree
func (self *pool) insertChunksToChain(chunks []ledger.SnapshotChunk, source types.BlockSource) error {
	b := batch.NewBatch(self.snapshotExists, self.accountExists, self.version.Val(), 50)
	for _, v := range chunks {
		if v.AccountBlocks != nil {
			for _, vv := range v.AccountBlocks {
				if err := self.accountExists(vv.Hash); err == nil {
					self.log.Info("[A]block exist, ignore.", "block", vv.Hash)
					continue
				}
				block := newAccountPoolBlock(vv, nil, self.version, source)
				self.log.Info("[A]add block to batch.", "account", vv.AccountAddress, "height", vv.Height, "block", vv.Hash, "batchId", b.Id())
				err := b.AddItem(block)
				if err != nil && err == batch.MAX_ERROR {
					err := b.Batch(self.insertSnapshotBucketForChunks, self.insertAccountsBucketForChunks)
					if err != nil {
						return err
					} else {
						b = batch.NewBatch(self.snapshotExists, self.accountExists, self.version.Val(), 50)
						err = b.AddItem(block)
						if err != nil {
							return err
						}
						continue
					}
				}
				if err != nil {
					return err
				}
			}
		}
		if v.SnapshotBlock != nil {
			if err := self.snapshotExists(v.SnapshotBlock.Hash); err == nil {
				self.log.Info("[S]block exist, ignore.", "block", v.SnapshotBlock.Hash)
				continue
			}

			block := newSnapshotPoolBlock(v.SnapshotBlock, self.version, source)
			self.log.Info("[S]add block to batch.", "block", v.SnapshotBlock.Hash, "batchId", b.Id())
			err := b.AddItem(block)
			if err != nil && err == batch.MAX_ERROR {
				err := b.Batch(self.insertSnapshotBucketForChunks, self.insertAccountsBucketForChunks)
				if err != nil {
					return err
				} else {
					b = batch.NewBatch(self.snapshotExists, self.accountExists, self.version.Val(), 50)
					err = b.AddItem(block)
					if err != nil {
						return err
					}
					continue
				}
			}
			if err != nil {
				return err
			}
		}
	}

	if b.Size() > 0 {
		return b.Batch(self.insertSnapshotBucketForChunks, self.insertAccountsBucketForChunks)
	}
	return nil
}

func (self *pool) insertSnapshotBucketForChunks(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return self.insertSnapshotBucket(p, bucket, version)
}

func (self *pool) insertAccountsBucketForChunks(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return self.insertAccountBucket(p, bucket, version)
}

func (self *pool) getHashSet(chunks []ledger.SnapshotChunk) map[types.Hash]bool {
	result := make(map[types.Hash]bool)
	for _, v := range chunks {
		if v.AccountBlocks != nil {
			for _, ab := range v.AccountBlocks {
				result[ab.Hash] = true
			}
		}
		if v.SnapshotBlock != nil {
			result[v.SnapshotBlock.Hash] = true
		}
	}

	return result
}

func (self *pool) getMinAccountBlocks(chunks []ledger.SnapshotChunk) map[types.Address][2]*ledger.HashHeight {
	addrM := make(map[types.Address][2]*ledger.HashHeight)

	for _, v := range chunks {
		if v.AccountBlocks == nil {
			continue
		}
		for _, ab := range v.AccountBlocks {
			tmp, ok := addrM[ab.AccountAddress]
			if !ok {
				tmp = [2]*ledger.HashHeight{{Hash: ab.Hash, Height: ab.Height}, {Hash: ab.Hash, Height: ab.Height}}
				addrM[ab.AccountAddress] = tmp
				continue
			}
			if tmp[0].Height > ab.Height {
				tmp[0].Height = ab.Height
				tmp[0].Hash = ab.Hash
			}

			if tmp[1].Height < ab.Height {
				tmp[1].Height = ab.Height
				tmp[1].Hash = ab.Hash
			}
		}
	}

	return addrM
}

func (self *pool) checkSnapshotInsert(headHH ledger.HashHeight, tailHH ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	cur := self.pendingSc.CurrentChain()

	return self.checkInsert(cur, headHH, tailHH, hashes)
}

func (self *pool) checkAccountsInsert(minAddrs map[types.Address][2]*ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	for k, v := range minAddrs {
		cur := self.selfPendingAc(k).CurrentChain()
		result := self.checkInsert(cur, *v[1], *v[0], hashes)
		if result != CONNECTED {
			return result
		}
	}
	return CONNECTED
}

func (self *pool) checkInsert(branch tree.Branch, waitingHeadH, waitingTailH ledger.HashHeight, hashes map[types.Hash]struct{}) ChainState {
	curHeight, curHash := branch.HeadHH()
	if waitingTailH.Height == curHeight {
		if waitingTailH.Hash == curHash {
			return CONNECTED
		} else {
			return FORKED
		}
	}
	if curHeight > waitingTailH.Height {
		if curHeight >= waitingHeadH.Height {
			if branch.ContainsKnot(waitingHeadH.Height, waitingHeadH.Hash, true) {
				return CONNECTED
			} else {
				return FORKED
			}
		}

		if _, ok := hashes[curHash]; ok {
			return CONNECTED
		} else {
			return FORKED
		}
	}
	return DISCONNECT
}
