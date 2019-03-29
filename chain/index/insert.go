package chain_index

import (
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) InsertAccountBlock(accountBlock *ledger.AccountBlock) error {
	blockHash := &accountBlock.Hash
	if ok, err := iDB.HasAccount(&accountBlock.AccountAddress); err != nil {
		return err
	} else if !ok {
		iDB.createAccount(&accountBlock.AccountAddress)
	}
	// hash -> addr & height
	iDB.store.Put(chain_utils.CreateAccountBlockHashKey(blockHash),
		append(accountBlock.AccountAddress.Bytes(), chain_utils.Uint64ToBytes(accountBlock.Height)...))

	// height -> hash
	iDB.store.Put(chain_utils.CreateAccountBlockHeightKey(&accountBlock.AccountAddress, accountBlock.Height), append(blockHash.Bytes()))
	if accountBlock.IsReceiveBlock() {
		// close send block
		iDB.store.Put(chain_utils.CreateReceiveKey(&accountBlock.FromBlockHash), blockHash.Bytes())

		// receive on road
		if err := iDB.receiveOnRoad(&accountBlock.FromBlockHash); err != nil {
			return err
		}
	} else {
		// insert on road block
		iDB.insertOnRoad(accountBlock.Hash, accountBlock.ToAddress)
	}

	for _, sendBlock := range accountBlock.SendBlockList {
		// insert on road block
		if err := iDB.InsertAccountBlock(sendBlock); err != nil {
			return err
		}
	}
	return nil
}

func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock,
	confirmedBlocks []*ledger.AccountBlock,
	snapshotBlockLocation *chain_file_manager.Location,
	abLocationsList []*chain_file_manager.Location,
	invalidBlocks []*ledger.AccountBlock,
	latestLocation *chain_file_manager.Location) error {

	heightBytes := chain_utils.Uint64ToBytes(snapshotBlock.Height)
	// hash -> height
	iDB.store.Put(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash), heightBytes)

	// height -> location
	iDB.store.Put(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height),
		append(snapshotBlock.Hash.Bytes(), chain_utils.SerializeLocation(snapshotBlockLocation)...))

	// confirm block
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		iDB.store.Put(chain_utils.CreateConfirmHeightKey(&addr, hashHeight.Height), heightBytes)
	}

	// flush account block indexes
	for index, block := range confirmedBlocks {
		// height -> account block location
		iDB.store.Put(chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height), append(block.Hash.Bytes(), chain_utils.SerializeLocation(abLocationsList[index])...))

		//iDB.store.Flush(batch, &block.Hash)
	}

	// latest on road id
	iDB.store.Put(chain_utils.CreateLatestOnRoadIdKey(), chain_utils.Uint64ToBytes(iDB.latestOnRoadId))

	// delete invalid blocks
	for i := len(invalidBlocks) - 1; i >= 0; i-- {
		invalidBlock := invalidBlocks[i]
		iDB.deleteAccountBlockHash(invalidBlock.Hash)

		iDB.deleteAccountBlockHeight(invalidBlock.AccountAddress, invalidBlock.Height)

		if invalidBlock.IsReceiveBlock() {
			// delete receive
			iDB.deleteReceive(invalidBlock.FromBlockHash)

			// insert on road
			iDB.insertOnRoad(invalidBlock.FromBlockHash, invalidBlock.AccountAddress)
		} else {
			// delete on road
			iDB.deleteOnRoad(invalidBlock.Hash)
		}

	}

	return nil
}
