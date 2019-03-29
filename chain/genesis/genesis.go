package chain_genesis

import "github.com/vitelabs/go-vite/config"

const (
	LedgerUnknown = byte(0)
	LedgerEmpty   = byte(1)
	LedgerValid   = byte(2)
	LedgerInvalid = byte(3)
)

func InitLedger(chain Chain, cfg *config.Genesis) error {
	// insert genesis account blocks
	// TODO fixme
	genesisAccountBlockList := NewGenesisAccountBlocks(cfg)
	for _, ab := range genesisAccountBlockList {
		err := chain.InsertAccountBlock(ab)
		if err != nil {
			panic(err)
		}
	}

	// init genesis snapshot block
	genesisSnapshotBlock := NewGenesisSnapshotBlock()

	// insert
	chain.InsertSnapshotBlock(genesisSnapshotBlock)
	return nil
}

func CheckLedger(chain Chain) (byte, error) {
	firstSb, err := chain.GetSnapshotHeaderByHeight(1)
	if err != nil {
		return LedgerUnknown, err
	}
	if firstSb == nil {
		return LedgerEmpty, nil
	}

	genesisSnapshotBlock := NewGenesisSnapshotBlock()

	if firstSb.Hash == genesisSnapshotBlock.Hash {
		return LedgerValid, nil
	}
	return LedgerInvalid, nil
}
