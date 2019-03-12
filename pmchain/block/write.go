package chain_block

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (bDB *BlockDB) Write(snapshotBlock *ledger.SnapshotBlock, accountBlockMap map[types.Address][]*ledger.AccountBlock) error {
	return nil
}
