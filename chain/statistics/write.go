package statistics

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/ledger"
)

// todo
func (sDB *StatisticsDB) InsertAccountBlock(block *ledger.AccountBlock) {
	batch := new(leveldb.Batch)

	if err := sDB.writeOnroadInfo(batch, &block.AccountAddress, &block.TokenId, block.Amount); err != nil {

	}
}
