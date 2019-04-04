package chain_block

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/ledger"
)

func (bDB *BlockDB) GetAccountBlock(location *chain_file_manager.Location) (*ledger.AccountBlock, error) {
	buf, err := bDB.Read(location)
	if err != nil {
		return nil, err
	}

	ab := &ledger.AccountBlock{}
	if err := ab.Deserialize(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("ab.Deserialize failed, [Error] %s", err.Error()))
	}

	return ab, nil
}
