package pmchain

func (c *chain) RegisterPrepareInsertAccountBlocks(listener PrepareInsertAccountBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterInsertAccountBlocks(listener InsertAccountBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterPrepareInsertSnapshotBlocks(listener PrepareInsertSnapshotBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterInsertSnapshotBlocks(listener InsertSnapshotBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterPrepareDeleteAccountBlocks(listener PrepareDeleteAccountBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterDeleteAccountBlocks(listener DeleteAccountBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterPrepareDeleteSnapshotBlocks(listener PrepareDeleteSnapshotBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) RegisterDeleteSnapshotBlocks(listener DeleteSnapshotBlocksListener) (eventHandler uint64) {
	return 0
}

func (c *chain) UnRegister(eventHandler uint64) {
}
