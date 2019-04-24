package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

var unreceivedFlag = []byte{0}

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
	iDB.insertSbHashHeight(batch, snapshotBlock.Hash, snapshotBlock.Height)

	// height -> location
	iDB.insertSbHeightLocation(batch, snapshotBlock, snapshotBlockLocation)

	// confirm block
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		batch.Put(chain_utils.CreateConfirmHeightKey(&addr, hashHeight.Height), heightBytes)

	}

	// flush account block indexes
	for index, block := range confirmedBlocks {
		// height -> account block location
		iDB.insertAbHeightLocation(batch, block, abLocationsList[index])
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
	iDB.insertAbHashHeight(batch, accountBlock.Hash, addrHeightValue)

	// height -> hash
	batch.Put(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height), blockHash.Bytes())

	if accountBlock.IsReceiveBlock() {
		// not genesis
		if accountBlock.BlockType != ledger.BlockTypeGenesisReceive {
			// close send block
			iDB.insertReceiveInfo(batch, accountBlock.FromBlockHash, blockHash.Bytes())
			// receive on road
			iDB.deleteOnRoad(batch, accountBlock.AccountAddress, accountBlock.FromBlockHash)
		}
	} else {
		// insert unreceived placeholder. avoid querying all data when no receive
		iDB.insertReceiveInfo(batch, accountBlock.Hash, unreceivedFlag)

		// insert on road block
		iDB.insertOnRoad(batch, accountBlock.ToAddress, accountBlock)
	}

	for _, sendBlock := range accountBlock.SendBlockList {
		// insert unreceived placeholder. avoid querying all data when no receive
		iDB.insertReceiveInfo(batch, sendBlock.Hash, unreceivedFlag)

		// send block hash -> addr & height
		iDB.insertAbHashHeight(batch, sendBlock.Hash, addrHeightValue)

		// insert on road block
		iDB.insertOnRoad(batch, sendBlock.ToAddress, sendBlock)
	}

	return nil
}

func (iDB *IndexDB) insertAbHashHeight(batch interfaces.Batch, hash types.Hash, value []byte) {
	key := chain_utils.CreateAccountBlockHashKey(&hash)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertAbHeightLocation(batch interfaces.Batch, block *ledger.AccountBlock, location *chain_file_manager.Location) {
	key := chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height)
	value := append(block.Hash.Bytes(), chain_utils.SerializeLocation(location)...)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertSbHashHeight(batch interfaces.Batch, hash types.Hash, height uint64) {
	key := chain_utils.CreateSnapshotBlockHashKey(&hash)

	value := chain_utils.Uint64ToBytes(height)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertSbHeightLocation(batch interfaces.Batch, block *ledger.SnapshotBlock, location *chain_file_manager.Location) {
	key := chain_utils.CreateSnapshotBlockHeightKey(block.Height)
	value := append(block.Hash.Bytes(), chain_utils.SerializeLocation(location)...)

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertReceiveInfo(batch interfaces.Batch, sendBlockHash types.Hash, value []byte) {
	key := chain_utils.CreateReceiveKey(&sendBlockHash)
	//value := receiveBlockHash.Bytes()

	iDB.cache.Set(string(key), value)
	batch.Put(key, value)
}
