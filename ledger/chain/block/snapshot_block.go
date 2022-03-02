package chain_block

import (
	"fmt"

	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_file_manager "github.com/vitelabs/go-vite/v2/ledger/chain/file_manager"
)

// GetSnapshotBlock by location
func (bDB *BlockDB) GetSnapshotBlock(location *chain_file_manager.Location) (*ledger.SnapshotBlock, error) {
	buf, err := bDB.Read(location)
	if err != nil {
		return nil, err
	}
	if len(buf) <= 0 {
		return nil, nil
	}
	sb := &ledger.SnapshotBlock{}
	if err := sb.Deserialize(buf); err != nil {
		return nil, fmt.Errorf("sb.Deserialize failed, Error: %s", err.Error())
	}

	return sb, nil
}

// TODO optimize
func (bDB *BlockDB) GetSnapshotHeader(location *chain_file_manager.Location) (*ledger.SnapshotBlock, error) {
	sb, err := bDB.GetSnapshotBlock(location)
	if err != nil {
		return nil, err
	}
	if sb == nil {
		return nil, nil
	}
	sb.SnapshotContent = nil
	return sb, nil
}
