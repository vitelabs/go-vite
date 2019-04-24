package chain_cache

func (cache *Cache) Init() error {
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

func (cache *Cache) initLatestSnapshotBlock() error {
	latestSnapshotBlock, err := cache.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}

	// set latest block
	cache.hd.SetLatestSnapshotBlock(latestSnapshotBlock)
	return nil
}
