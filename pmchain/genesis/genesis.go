package chain_genesis

func InitLedger(indexDB IndexDB, blockDB BlockDB) error {
	// insert genesis account blocks

	// insert genesis snapshot block
	indexDB.InsertSnapshotBlock(nil, nil, nil, nil)
	return nil
}

func CheckLedger(indexDB IndexDB, blockDB BlockDB) bool {
	return false
}
