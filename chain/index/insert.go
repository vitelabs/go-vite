package chain_index

import (
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) InsertAccountBlock(accountBlock *ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()

	blockHash := &accountBlock.Hash

	if ok, err := iDB.HasAccount(&accountBlock.AccountAddress); err != nil {
		return err
	} else if !ok {
		iDB.createAccount(batch, &accountBlock.AccountAddress)
	}
	// hash -> addr & height
	addrHeightValue := append(accountBlock.AccountAddress.Bytes(), chain_utils.Uint64ToBytes(accountBlock.Height)...)
	batch.Put(chain_utils.CreateAccountBlockHashKey(blockHash), addrHeightValue)

	// height -> hash
	batch.Put(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height), blockHash.Bytes())

	if accountBlock.IsReceiveBlock() {
		// not genesis
		if accountBlock.BlockType != ledger.BlockTypeGenesisReceive {
			// close send block
			batch.Put(chain_utils.CreateReceiveKey(&accountBlock.FromBlockHash), blockHash.Bytes())

			// receive on road
			if err := iDB.deleteOnRoad(batch, accountBlock.FromBlockHash); err != nil {
				return err
			}
		}
	} else {
		// insert on road block
		if err := iDB.insertOnRoad(batch, accountBlock.Hash, accountBlock.ToAddress); err != nil {
			return err
		}
	}

	for _, sendBlock := range accountBlock.SendBlockList {
		// send block hash -> addr & height
		batch.Put(chain_utils.CreateAccountBlockHashKey(&sendBlock.Hash), addrHeightValue)
		// insert on road block
		if err := iDB.insertOnRoad(batch, sendBlock.Hash, sendBlock.ToAddress); err != nil {
			return err
		}
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
		batch.Put(chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height), append(block.Hash.Bytes(), chain_utils.SerializeLocation(abLocationsList[index])...))
	}

	// latest on road id
	batch.Put(chain_utils.CreateLatestOnRoadIdKey(), chain_utils.Uint64ToBytes(iDB.latestOnRoadId))

	// write snapshot
	iDB.store.WriteSnapshot(batch, confirmedBlocks)
}
