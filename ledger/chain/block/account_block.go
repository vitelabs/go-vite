package chain_block

import (
	"fmt"
	"sort"

	ledger "github.com/vitelabs/go-vite/interfaces/core"
	chain_file_manager "github.com/vitelabs/go-vite/ledger/chain/file_manager"
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

func sortAccountBlocksInChunk(chunk *ledger.SnapshotChunk) []*ledger.AccountBlock {
	if len(chunk.AccountBlocks) == 0 {
		return chunk.AccountBlocks
	}
	result := make([]*ledger.AccountBlock, len(chunk.AccountBlocks))
	copy(result, chunk.AccountBlocks)

	sort.Slice(result, func(i, j int) bool {
		addressResult := result[i].AccountAddress.Compare(result[j].AccountAddress)
		if addressResult == 0 {
			return result[i].Height < result[j].Height
		}
		return addressResult < 0
	})
	return result
}
