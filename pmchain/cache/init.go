package chain_cache

func (cache *Cache) init() error {
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
