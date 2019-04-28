package chain_cache

import (
	"github.com/patrickmn/go-cache"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type dataSet struct {
	store *cache.Cache
}

func NewDataSet() *dataSet {
	return &dataSet{
		store: cache.New(cache.NoExpiration, cache.NoExpiration),
	}
}

func (ds *dataSet) Close() {
	ds.store.Flush()
	ds.store = nil
}

func (ds *dataSet) InsertAccountBlock(accountBlock *ledger.AccountBlock) {
	hashKey := string(chain_utils.CreateAccountBlockHashKey(&accountBlock.Hash))
	heightKey := string(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height))

	ds.store.Set(hashKey, accountBlock, cache.NoExpiration)
	ds.store.Set(heightKey, hashKey, cache.NoExpiration)
}

func (ds *dataSet) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	hashKey := string(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash))
	heightKey := string(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height))

	ds.store.Set(hashKey, snapshotBlock, time.Second*1800)
	ds.store.Set(heightKey, hashKey, time.Second*1800)
}

func (ds *dataSet) DeleteAccountBlocks(accountBlocks []*ledger.AccountBlock) {
	for _, accountBlock := range accountBlocks {
		hashKey := string(chain_utils.CreateAccountBlockHashKey(&accountBlock.Hash))
		heightKey := string(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height))

		ds.store.Delete(hashKey)
		ds.store.Delete(heightKey)
	}

}

func (ds *dataSet) DeleteSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	hashKey := string(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash))
	heightKey := string(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height))

	ds.store.Delete(hashKey)
	ds.store.Delete(heightKey)
}

func (ds *dataSet) GetAccountBlock(hash types.Hash) *ledger.AccountBlock {
	hashKey := string(chain_utils.CreateAccountBlockHashKey(&hash))
	block, ok := ds.store.Get(hashKey)
	if !ok {
		return nil
	}

	return block.(*ledger.AccountBlock)
}

func (ds *dataSet) GetAccountBlockByHeight(address types.Address, height uint64) *ledger.AccountBlock {
	heightKey := string(chain_utils.CreateAccountBlockHeightKey(&address, height))
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
	_, ok := ds.store.Get(string(hashKey))
	return ok
}

func (ds *dataSet) GetSnapshotBlock(hash types.Hash) *ledger.SnapshotBlock {
	hashKey := chain_utils.CreateSnapshotBlockHashKey(&hash)
	block, ok := ds.store.Get(string(hashKey))
	if !ok {
		return nil
	}

	return block.(*ledger.SnapshotBlock)
}

func (ds *dataSet) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	heightKey := string(chain_utils.CreateSnapshotBlockHeightKey(height))
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
	_, ok := ds.store.Get(string(hashKey))
	return ok
}
