package chain_cache

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type dataSet struct {
	store *cache.Cache

	snapshotKeepCount uint64
}

func NewDataSet() *dataSet {
	return &dataSet{
		store:             cache.New(cache.NoExpiration, 1*time.Minute),
		snapshotKeepCount: 1800,
	}
}

func (ds *dataSet) Close() {
	ds.store.Flush()
	ds.store = nil
}

func (ds *dataSet) IsLarge() bool {
	return ds.store.ItemCount() > 20*10000
}

func (ds *dataSet) InsertAccountBlock(accountBlock *ledger.AccountBlock) {
	ds.insertAccountBlock(accountBlock, cache.NoExpiration)
}

func (ds *dataSet) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	hashKey := chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash).String()
	heightKey := chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height).String()

	ds.store.Set(hashKey, snapshotBlock, time.Second*1800)
	ds.store.Set(heightKey, hashKey, time.Second*1800)
	// delete stale
	ds.deleteStaleSnapshotBlock(snapshotBlock.Height)
}

func (ds *dataSet) DeleteAccountBlocks(accountBlocks []*ledger.AccountBlock) {
	for _, accountBlock := range accountBlocks {
		hashKey := chain_utils.CreateAccountBlockHashKey(&accountBlock.Hash).String()
		heightKey := chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height).String()

		ds.store.Delete(hashKey)
		ds.store.Delete(heightKey)

		for _, sendBlock := range accountBlock.SendBlockList {
			// delete send block
			ds.store.Delete(chain_utils.CreateAccountBlockHashKey(&sendBlock.Hash).String())
		}
	}
}

func (ds *dataSet) DelayDeleteAccountBlocks(accountBlocks []*ledger.AccountBlock, delay time.Duration) {
	for _, accountBlock := range accountBlocks {
		ds.insertAccountBlock(accountBlock, delay)
	}
}

func (ds *dataSet) DeleteSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	hashKey := chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash)
	heightKey := chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height)

	ds.store.Delete(hashKey.String())
	ds.store.Delete(heightKey.String())
}

func (ds *dataSet) GetAccountBlock(hash types.Hash) *ledger.AccountBlock {
	hashKey := chain_utils.CreateAccountBlockHashKey(&hash)
	block, ok := ds.store.Get(hashKey.String())
	if !ok {
		return nil
	}

	return block.(*ledger.AccountBlock)
}

func (ds *dataSet) GetAccountBlockByHeight(address types.Address, height uint64) *ledger.AccountBlock {
	heightKey := chain_utils.CreateAccountBlockHeightKey(&address, height).String()
	hashKey, ok := ds.store.Get(heightKey)
	if !ok {
		return nil
	}

	block, ok := ds.store.Get(hashKey.(string))
	if !ok {
		return nil
	}

	return block.(*ledger.AccountBlock)
}

func (ds *dataSet) IsAccountBlockExisted(hash types.Hash) bool {
	hashKey := chain_utils.CreateAccountBlockHashKey(&hash)
	_, ok := ds.store.Get(hashKey.String())
	return ok
}

func (ds *dataSet) GetSnapshotBlock(hash types.Hash) *ledger.SnapshotBlock {
	hashKey := chain_utils.CreateSnapshotBlockHashKey(&hash)
	block, ok := ds.store.Get(hashKey.String())
	if !ok {
		return nil
	}

	return block.(*ledger.SnapshotBlock)
}

func (ds *dataSet) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	heightKey := chain_utils.CreateSnapshotBlockHeightKey(height).String()
	hashKey, ok := ds.store.Get(heightKey)
	if !ok {
		return nil
	}

	block, ok := ds.store.Get(hashKey.(string))
	if !ok {
		return nil
	}

	return block.(*ledger.SnapshotBlock)
}

func (ds *dataSet) IsSnapshotBlockExisted(hash types.Hash) bool {
	hashKey := chain_utils.CreateSnapshotBlockHashKey(&hash)
	_, ok := ds.store.Get(hashKey.String())
	return ok
}

func (ds *dataSet) GetStatus() []interfaces.DBStatus {
	count := uint64(ds.store.ItemCount())
	return []interfaces.DBStatus{{
		Name:   "dataSet.store",
		Count:  count,
		Size:   count * 400,
		Status: "",
	}}
}

func (ds *dataSet) insertAccountBlock(accountBlock *ledger.AccountBlock, delay time.Duration) {
	hashKey := chain_utils.CreateAccountBlockHashKey(&accountBlock.Hash).String()
	heightKey := chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height).String()

	ds.store.Set(hashKey, accountBlock, delay)
	ds.store.Set(heightKey, hashKey, delay)

	for _, sendBlock := range accountBlock.SendBlockList {
		// set send block hash
		ds.store.Set(chain_utils.CreateAccountBlockHashKey(&sendBlock.Hash).String(), accountBlock, delay)
	}
}

func (ds *dataSet) deleteStaleSnapshotBlock(height uint64) {
	if height <= ds.snapshotKeepCount {
		return
	}
	staleHeight := height - ds.snapshotKeepCount
	heightKey := chain_utils.CreateSnapshotBlockHeightKey(staleHeight).String()
	hash, ok := ds.store.Get(heightKey)
	if !ok {
		return
	}

	ds.store.Delete(heightKey)
	ds.store.Delete(hash.(string))
}
