package chain_cache

func (cache *Cache) init() error {
	genesisSnapshotBlock, err := cache.chain.GetSnapshotBlockByHeight(1)
	if err != nil {
		return err
	}

	// init genesis snapshot block
	dataId := cache.ds.InsertSnapshotBlock(genesisSnapshotBlock)
	cache.hd.SetGenesisSnapshotBlock(dataId)

	// init latest snapshot block
	latestSnapshotBlock, err := cache.chain.GetLatestSnapshotBlock()
	if err != nil {
		return err
	}

	if latestSnapshotBlock != nil {
		cache.UpdateLatestSnapshotBlock(latestSnapshotBlock)
	}

	return nil
}
