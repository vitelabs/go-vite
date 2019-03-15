package chain_genesis

const (
	LedgerUnknown = byte(0)
	LedgerEmpty   = byte(1)
	LedgerValid   = byte(2)
	LedgerInvalid = byte(3)
)

func InitLedger(chain Chain) error {
	// insert genesis account blocks

	// init genesis snapshot block
	genesisSnapshotBlock := NewGenesisSnapshotBlock()

	// insert
	chain.InsertSnapshotBlock(genesisSnapshotBlock)
	return nil
}

func CheckLedger(chain Chain) (byte, error) {
	latestSb, err := chain.GetLatestSnapshotBlock()
	if err != nil {
		return LedgerUnknown, err
	}
	if latestSb == nil {
		return LedgerEmpty, nil
	}

	genesisSnapshotBlock := NewGenesisSnapshotBlock()

	if latestSb.Hash == genesisSnapshotBlock.Hash {
		return LedgerValid, nil
	}
	return LedgerInvalid, nil
}
