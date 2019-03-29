package chain_index

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

// TODO
func (iDB *IndexDB) Rollback(deletedSnapshotSegments []*chain_block.SnapshotSegment, location *chain_file_manager.Location) error {
	if len(deletedSnapshotSegments) <= 0 {
		return nil
	}

	openSendBlockHashMap := make(map[types.Hash]struct{})

	for _, seg := range deletedSnapshotSegments {
		snapshotBlock := seg.SnapshotBlock
		iDB.deleteSnapshotBlockHash(snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(snapshotBlock.Height)

		for _, block := range seg.AccountBlocks {
			// delete account block hash index
			iDB.deleteAccountBlockHash(block.Hash)

			// delete account block height index
			iDB.deleteAccountBlockHeight(block.AccountAddress, block.Height)

			// delete confirmed index
			iDB.deleteConfirmHeight(block.AccountAddress, block.Height)

			if block.IsReceiveBlock() {

				if _, ok := openSendBlockHashMap[block.FromBlockHash]; !ok {
					delete(openSendBlockHashMap, block.FromBlockHash)

					// delete receive index
					iDB.deleteReceive(block.FromBlockHash)

					// insert onRoad
					iDB.insertOnRoad(block.FromBlockHash, block.AccountAddress)
				}
			} else {
				openSendBlockHashMap[block.Hash] = struct{}{}
			}
		}
	}

	for sendBlockHash := range openSendBlockHashMap {
		if err := iDB.deleteOnRoad(sendBlockHash); err != nil {
			return err
		}
	}

	return nil
}

func (iDB *IndexDB) deleteSnapshotBlockHash(snapshotBlockHash types.Hash) {
	iDB.store.Delete(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlockHash))
}

func (iDB *IndexDB) deleteSnapshotBlockHeight(snapshotBlockHeight uint64) {
	iDB.store.Delete(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlockHeight))
}

func (iDB *IndexDB) deleteAccountBlockHash(accountBlockHash types.Hash) {
	iDB.store.Delete(chain_utils.CreateAccountBlockHashKey(&accountBlockHash))
}

func (iDB *IndexDB) deleteAccountBlockHeight(addr types.Address, height uint64) {
	iDB.store.Delete(chain_utils.CreateAccountBlockHeightKey(&addr, height))
}

func (iDB *IndexDB) deleteReceive(sendBlockHash types.Hash) {
	iDB.store.Delete(chain_utils.CreateReceiveKey(&sendBlockHash))
}

func (iDB *IndexDB) deleteConfirmHeight(addr types.Address, height uint64) {
	iDB.store.Delete(chain_utils.CreateConfirmHeightKey(&addr, height))
}
