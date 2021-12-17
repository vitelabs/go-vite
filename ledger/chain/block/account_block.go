package chain_block

import (
	"fmt"

	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_file_manager "github.com/vitelabs/go-vite/v2/ledger/chain/file_manager"
)

// GetAccountBlock by location
func (bDB *BlockDB) GetAccountBlock(location *chain_file_manager.Location) (*ledger.AccountBlock, error) {
	buf, err := bDB.Read(location)
	if err != nil {
		return nil, err
	}

	ab := &ledger.AccountBlock{}
	if err := ab.Deserialize(buf); err != nil {
		return nil, fmt.Errorf("ab.Deserialize failed, [Error] %s", err.Error())
	}

	return ab, nil
}
