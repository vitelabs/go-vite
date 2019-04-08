package chain_cache

import (
	"github.com/pkg/errors"
)

func (cache *Cache) Init() error {

	// init genesis snapshot block
	if err := cache.initGenesisSnapshotBlock(); err != nil {
		return err
	}

	// init latest snapshot block
	if err := cache.initLatestSnapshotBlock(); err != nil {
		return err
	}

	// init quota list
	if err := cache.quotaList.init(); err != nil {
		return err
	}
	return nil
}

func (cache *Cache) initGenesisSnapshotBlock() error {
	genesisSnapshotBlock, err := cache.chain.QuerySnapshotBlockByHeight(1)
	if err != nil {
		return err
	}

	if genesisSnapshotBlock == nil {
		return errors.New("genesisSnapshotBlock is nil")
	}
	dataId := cache.ds.InsertSnapshotBlock(genesisSnapshotBlock)
	cache.hd.SetGenesisSnapshotBlock(dataId)
	return nil
}

func (cache *Cache) initLatestSnapshotBlock() error {
	latestSnapshotBlock, err := cache.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}
	cache.setLatestSnapshotBlock(latestSnapshotBlock)
	return nil
}
