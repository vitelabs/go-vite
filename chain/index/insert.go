package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) InsertAccountBlock(accountBlock *ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()

	if err := iDB.insertAccountBlock(batch, accountBlock); err != nil {
		return err
	}
	iDB.store.WriteAccountBlock(batch, accountBlock)

	return nil
}

func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock, snapshotBlockLocation *chain_file_manager.Location, abLocationsList []*chain_file_manager.Location) {

	batch := iDB.store.NewBatch()

	heightBytes := chain_utils.Uint64ToBytes(snapshotBlock.Height)
	// hash -> height
	batch.Put(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash), heightBytes)

	// height -> location
	batch.Put(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height), append(snapshotBlock.Hash.Bytes(), chain_utils.SerializeLocation(snapshotBlockLocation)...))

	// confirm block
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		batch.Put(chain_utils.CreateConfirmHeightKey(&addr, hashHeight.Height), heightBytes)
	}

	// flush account block indexes
	for index, block := range confirmedBlocks {
		// height -> account block location
		iDB.insertHeightLocation(batch, block, abLocationsList[index])
	}

	// write snapshot
	iDB.store.WriteSnapshot(batch, confirmedBlocks)
}

func (iDB *IndexDB) insertAccountBlock(batch *leveldb.Batch, accountBlock *ledger.AccountBlock) error {

	blockHash := &accountBlock.Hash

	if ok, err := iDB.HasAccount(accountBlock.AccountAddress); err != nil {
		return err
	} else if !ok {
		iDB.createAccount(batch, &accountBlock.AccountAddress)
	}
	// hash -> addr & height
	addrHeightValue := append(accountBlock.AccountAddress.Bytes(), chain_utils.Uint64ToBytes(accountBlock.Height)...)
	iDB.insertHashHeight(batch, accountBlock.Hash, addrHeightValue)

	// height -> hash
	batch.Put(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height), blockHash.Bytes())

	if accountBlock.IsReceiveBlock() {
		// not genesis
		if accountBlock.BlockType != ledger.BlockTypeGenesisReceive {
			// close send block
			batch.Put(chain_utils.CreateReceiveKey(&accountBlock.FromBlockHash), blockHash.Bytes())

			// receive on road
			iDB.deleteOnRoad(batch, accountBlock.AccountAddress, accountBlock.FromBlockHash)
		}
	} else {
		// insert on road block
		iDB.insertOnRoad(batch, accountBlock.ToAddress, accountBlock.Hash)
	}

	for _, sendBlock := range accountBlock.SendBlockList {
		// send block hash -> addr & height
		iDB.insertHashHeight(batch, sendBlock.Hash, addrHeightValue)

		// insert on road block
		iDB.insertOnRoad(batch, sendBlock.ToAddress, sendBlock.Hash)
	}

	return nil
}

func (iDB *IndexDB) insertHashHeight(batch interfaces.Batch, hash types.Hash, value []byte) {
	key := chain_utils.CreateAccountBlockHashKey(&hash)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertHeightLocation(batch interfaces.Batch, block *ledger.AccountBlock, location *chain_file_manager.Location) {
	key := chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height)
	value := append(block.Hash.Bytes(), chain_utils.SerializeLocation(location)...)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}
