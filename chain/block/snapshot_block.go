package chain_block

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
)

func (bDB *BlockDB) GetSnapshotBlock(location *Location) (*ledger.SnapshotBlock, error) {
	buf, err := bDB.Read(location)
	if err != nil {
		return nil, err
	}
	sb := &ledger.SnapshotBlock{}
	if err := sb.Deserialize(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("sb.Deserialize failed, Error: %s", err.Error()))
	}

	return sb, nil
}

// TODO optimize
func (bDB *BlockDB) GetSnapshotHeader(location *Location) (*ledger.SnapshotBlock, error) {
	sb, err := bDB.GetSnapshotBlock(location)
	if err != nil {
		return nil, err
	}
	sb.SnapshotContent = nil
	return sb, nil
}
