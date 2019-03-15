package chain_cache

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) Init(latestSnapshotBlock *ledger.SnapshotBlock) error {
	genesisSnapshotBlock, err := cache.chain.GetSnapshotBlockByHeight(1)
	if err != nil {
		return err
	}

	if genesisSnapshotBlock == nil {
		return errors.New("genesisSnapshotBlock is nil")
	}

	// init genesis snapshot block
	dataId := cache.ds.InsertSnapshotBlock(genesisSnapshotBlock)
	cache.hd.SetGenesisSnapshotBlock(dataId)

	// init latest snapshot block
	cache.UpdateLatestSnapshotBlock(latestSnapshotBlock)

	return nil
}
