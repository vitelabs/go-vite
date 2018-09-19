package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func getAccountBlockHash(dbKey []byte) *types.Hash {
	hashBytes := dbKey[17:]
	hash, _ := types.BytesToHash(hashBytes)
	return &hash
}

type AccountChain struct {
	db *leveldb.DB
}

func NewAccountChain(db *leveldb.DB) *AccountChain {
	return &AccountChain{
		db: db,
	}
}

func (ac *AccountChain) DeleteBlock(batch *leveldb.Batch, accountId uint64, height uint64, hash *types.Hash) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, height, hash.Bytes())
	batch.Delete(key)
}

func (ac *AccountChain) DeleteBlockMeta(batch *leveldb.Batch, hash *types.Hash) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, hash.Bytes())
	batch.Delete(key)
}

func (ac *AccountChain) WriteBlock(batch *leveldb.Batch, accountId uint64, block *ledger.AccountBlock) error {
	buf, err := block.DbSerialize()
	if err != nil {
		return err
	}

	key, err := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, block.Height, block.Hash.Bytes())

	batch.Put(key, buf)
	return nil
}

func (ac *AccountChain) WriteBlockMeta(batch *leveldb.Batch, blockHash *types.Hash, blockMeta *ledger.AccountBlockMeta) error {
	buf, err := blockMeta.DbSerialize()
	if err != nil {
		return err
	}

	key, err := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, blockHash.Bytes())

	batch.Put(key, buf)
	return nil
}

func (ac *AccountChain) GetHashByHeight(accountId uint64, height uint64) (*types.Hash, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, height)
	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}

		return nil, nil
	}

	return getAccountBlockHash(iter.Key()), nil

}

func (ac *AccountChain) GetAbHashList(accountId uint64, height uint64, count, step int, forward bool) []*types.Hash {
	hashList := make([]*types.Hash, 0)
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, height)
	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if forward {
		iter.Next()
	} else {
		iter.Prev()
	}

	for j := 0; j < count; j++ {
		for i := 0; i < step; i++ {
			var ok bool
			if forward {
				ok = iter.Next()
			} else {
				ok = iter.Prev()
			}

			if !ok {
				return hashList
			}
		}
		hashList = append(hashList, getAccountBlockHash(iter.Key()))
	}
	return hashList
}

func (ac *AccountChain) GetLatestBlock(accountId uint64) (*ledger.AccountBlock, error) {
	key, err := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)
	if err != nil {
		return nil, err
	}

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	block := &ledger.AccountBlock{}
	if ddsErr := block.DbDeserialize(iter.Value()); ddsErr == nil {
		return nil, ddsErr
	}

	block.Hash = *getAccountBlockHash(iter.Key())
	return block, nil
}

func (ac *AccountChain) GetBlockListByAccountId(accountId uint64, startHeight uint64, endHeight uint64) ([]*ledger.AccountBlock, error) {
	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, startHeight)
	limitKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, endHeight+1)

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	var blockList []*ledger.AccountBlock

	for iter.Next() {
		block := &ledger.AccountBlock{}
		err := block.DbDeserialize(iter.Value())

		if err != nil {
			return nil, err
		}

		block.Hash = *getAccountBlockHash(iter.Key())
		blockList = append(blockList, block)
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return blockList, nil
}

func (ac *AccountChain) GetBlock(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	blockMeta, gbmErr := ac.GetBlockMeta(blockHash)
	if gbmErr != nil {
		return nil, gbmErr
	}

	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, blockMeta.AccountId, blockMeta.Height)

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	if !iter.Last() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	accountBlock := &ledger.AccountBlock{}
	if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr != nil {
		return nil, dsErr
	}

	accountBlock.Hash = *blockHash
	accountBlockMeta, err := ac.GetBlockMeta(&accountBlock.Hash)
	if err != nil {
		return nil, err
	}

	accountBlock.Meta = accountBlockMeta

	return accountBlock, nil
}

func (ac *AccountChain) GetBlockMeta(blockHash *types.Hash) (*ledger.AccountBlockMeta, error) {
	key, err := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, blockHash.Bytes())
	if err != nil {
		return nil, err
	}
	blockMetaBytes, err := ac.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	blockMeta := &ledger.AccountBlockMeta{}
	if err := blockMeta.DbDeserialize(blockMetaBytes); err != nil {
		return nil, err
	}

	return blockMeta, nil
}

func (ac *AccountChain) GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error) {
	key, _ := database.EncodeKey(database.DBKP_LOG_LIST, logListHash.Bytes())
	data, err := ac.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	vmLogList := ledger.VmLogList{}
	if dErr := vmLogList.Deserialize(data); dErr != nil {
		return nil, err
	}

	return vmLogList, err
}

func (ac *AccountChain) getConfirmHeight(accountBlockHash *types.Hash) (uint64, *ledger.AccountBlockMeta, error) {

	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, accountBlockHash.Bytes())
	data, err := ac.db.Get(key, nil)
	if err != nil {
		return 0, nil, err
	}

	accountBlockMeta := &ledger.AccountBlockMeta{}
	if dsErr := accountBlockMeta.DbDeserialize(data); dsErr != nil {
		return 0, nil, dsErr
	}

	if accountBlockMeta.SnapshotHeight > 0 {
		return accountBlockMeta.SnapshotHeight, accountBlockMeta, nil
	}
	return 0, accountBlockMeta, nil
}

func (ac *AccountChain) GetConfirmHeight(accountBlockHash *types.Hash) (uint64, error) {

	confirmHeight, accountBlockMeta, err := ac.getConfirmHeight(accountBlockHash)
	if err != nil {
		return 0, err
	}

	if confirmHeight > 0 {
		return confirmHeight, nil
	}

	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountBlockMeta.AccountId, accountBlockMeta.Height+1)
	endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountBlockMeta.AccountId)

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)
	defer iter.Release()

	for iter.Next() {
		blockHash := getAccountBlockHash(iter.Key())
		confirmHeight, _, err := ac.getConfirmHeight(blockHash)
		if err != nil {
			return 0, err
		}

		if confirmHeight > 0 {
			return confirmHeight, nil
		}
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return 0, err
	}

	return 0, nil
}

func (ac *AccountChain) WriteVmLogList(batch *leveldb.Batch, logList ledger.VmLogList) error {
	key, _ := database.EncodeKey(database.DBKP_LOG_LIST, logList.Hash())

	buf, err := logList.Serialize()
	if err != nil {
		return err
	}

	batch.Put(key, buf)

	return nil
}

func (ac *AccountChain) DeleteVmLogList(batch *leveldb.Batch, logListHash *types.Hash) {
	key, _ := database.EncodeKey(database.DBKP_LOG_LIST, logListHash)
	batch.Delete(key)
}

func (ac *AccountChain) WriteContractGid(batch *leveldb.Batch, gid *types.Gid, addr *types.Address) {
	key, _ := database.EncodeKey(database.DBKP_ADDR_GID, addr.Bytes())
	batch.Put(key, gid.Bytes())
}

func (ac *AccountChain) GetContractGid(addr *types.Address) (*types.Gid, error) {
	key, _ := database.EncodeKey(database.DBKP_ADDR_GID, addr.Bytes())
	data, err := ac.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	gid, err2 := types.BytesToGid(data)
	if err2 != nil {
		return nil, err2
	}
	return &gid, nil
}

func (ac *AccountChain) DeleteContract(batch *leveldb.Batch, addr *types.Address) {
	key, _ := database.EncodeKey(database.DBKP_ADDR_GID, addr.Bytes())
	batch.Delete(key)
}

func (ac *AccountChain) ReopenSendBlocks(batch *leveldb.Batch, reopenList []*types.Hash, deletedMap map[uint64]uint64) error {
	for _, blockHash := range reopenList {
		blockMeta, err := ac.GetBlockMeta(blockHash)
		if err != nil {
			return err
		}
		if blockMeta == nil {
			continue
		}

		// The block will be deleted, don't need be write
		if deletedHeight := deletedMap[blockMeta.AccountId]; deletedHeight != 0 && blockMeta.Height >= deletedHeight {
			continue
		}
		blockMeta.ReceiveBlockHeight = 0
		writeErr := ac.WriteBlockMeta(batch, blockHash, blockMeta)
		if writeErr != nil {
			return err
		}
	}
	return nil
}

func (ac *AccountChain) deleteChain(batch *leveldb.Batch, accountId uint64, toHeight uint64) ([]*ledger.AccountBlock, error) {
	deletedChain := make([]*ledger.AccountBlock, 0)

	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, toHeight)
	endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)
	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)
	defer iter.Release()

	for iter.Next() {

		deleteBlock := &ledger.AccountBlock{}
		if dsErr := deleteBlock.DbDeserialize(iter.Value()); dsErr != nil {
			return nil, dsErr
		}

		deleteBlock.Hash = *getAccountBlockHash(iter.Key())

		// Delete vm log list
		ac.DeleteVmLogList(batch, deleteBlock.LogHash)

		// Delete contract gid

		// Delete block
		ac.DeleteBlock(batch, accountId, deleteBlock.Height, &deleteBlock.Hash)

		// Delete block meta
		ac.DeleteBlockMeta(batch, &deleteBlock.Hash)

		deletedChain = append(deletedChain, deleteBlock)

	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return deletedChain, nil
}

func (ac *AccountChain) Delete(batch *leveldb.Batch, deleteMap map[uint64]uint64) (map[types.Address][]*ledger.AccountBlock, error) {
	deleted := make(map[types.Address][]*ledger.AccountBlock)
	for accountId, deleteHeight := range deleteMap {
		deletedChain, err := ac.deleteChain(batch, accountId, deleteHeight)
		if err != nil {
			return nil, err
		}

		if len(deletedChain) > 0 {
			deleted[deletedChain[0].AccountAddress] = deletedChain
		}
	}

	return deleted, nil
}

func (ac *AccountChain) GetDeleteMapAndReopenList(planToDelete map[uint64]uint64) (map[uint64]uint64, []*types.Hash, uint64, error) {
	currentNeedDelete := planToDelete
	deleteMap := make(map[uint64]uint64)
	reopenList := make([]*types.Hash, 0)
	needDeleteSnapshotBlockHeight := uint64(0)

	for len(currentNeedDelete) > 0 {
		nextNeedDelete := make(map[uint64]uint64)

		for accountId, needDeleteHeight := range currentNeedDelete {
			endHeight := uint64(0)
			if deleteHeight := deleteMap[accountId]; deleteHeight != 0 {
				if deleteHeight <= needDeleteHeight {
					continue
				}
				endHeight = deleteHeight
				deleteMap[accountId] = needDeleteHeight
			} else {
				deleteMap[accountId] = needDeleteHeight
			}

			startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, needDeleteHeight)
			var endKey []byte
			if endHeight == 0 {
				endKey, _ = database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)
			} else {
				endKey, _ = database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, endHeight)
			}

			iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)

			for iter.Next() {
				accountBlock := &ledger.AccountBlock{}

				if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr == nil {
					iter.Release()
					return nil, nil, 0, dsErr
				}

				blockHash := getAccountBlockHash(iter.Key())
				accountBlockMeta, getBmErr := ac.GetBlockMeta(blockHash)
				if getBmErr != nil {
					iter.Release()
					return nil, nil, 0, getBmErr
				}

				if needDeleteSnapshotBlockHeight == 0 ||
					(accountBlockMeta.SnapshotHeight > 0 && accountBlockMeta.SnapshotHeight < needDeleteSnapshotBlockHeight) {
					needDeleteSnapshotBlockHeight = accountBlockMeta.SnapshotHeight
				}

				if accountBlock.IsSendBlock() {
					receiveBlockHeight := accountBlockMeta.ReceiveBlockHeight
					if receiveBlockHeight > 0 {
						receiveBlockMeta, getFromBlockMetaErr := ac.GetBlockMeta(&accountBlock.FromBlockHash)
						if getFromBlockMetaErr != nil {
							iter.Release()
							return nil, nil, 0, getFromBlockMetaErr
						}
						receiveAccountId := receiveBlockMeta.AccountId

						if currentDeleteHeight, nextDeleteHeight := currentNeedDelete[receiveAccountId], nextNeedDelete[receiveAccountId]; !(currentDeleteHeight != 0 && currentDeleteHeight <= receiveBlockHeight ||
							nextDeleteHeight != 0 && nextDeleteHeight <= receiveBlockHeight) {

							if needDeleteSnapshotBlockHeight == 0 ||
								receiveBlockHeight < needDeleteSnapshotBlockHeight {
								needDeleteSnapshotBlockHeight = receiveBlockHeight
							}

							nextNeedDelete[receiveAccountId] = receiveBlockHeight
						}
					}
				} else if accountBlock.IsReceiveBlock() {
					reopenList = append(reopenList, &accountBlock.FromBlockHash)
				}
			}
			if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
				iter.Release()
				return nil, nil, 0, err
			}
			iter.Release()
		}

		currentNeedDelete = nextNeedDelete
	}

	return deleteMap, reopenList, needDeleteSnapshotBlockHeight, nil
}

// TODO: cache
func (ac *AccountChain) GetPlanToDelete(maxAccountId uint64, snapshotBlockHeight uint64) (map[uint64]uint64, error) {
	planToDelete := make(map[uint64]uint64)

	for i := uint64(1); i <= maxAccountId; i++ {
		blockKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, i)

		iter := ac.db.NewIterator(util.BytesPrefix(blockKey), nil)
		if !iter.Last() {
			iter.Release()
			return nil, nil
		}

		for iter.Prev() {
			blockHash := getAccountBlockHash(iter.Key())
			blockMeta, getBmErr := ac.GetBlockMeta(blockHash)
			if getBmErr != nil {
				iter.Release()
				return nil, getBmErr
			}

			if blockMeta == nil {
				break
			}

			if blockMeta.RefSnapshotHeight >= snapshotBlockHeight {
				planToDelete[i] = blockMeta.Height
			} else {
				break
			}
		}

		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			iter.Release()
			return nil, err
		}
		iter.Release()
	}

	return planToDelete, nil
}

func (ac *AccountChain) GetConfirmAccountBlock(snapshotHeight uint64, accountId uint64) (*ledger.AccountBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	iter.Last()
	for iter.Prev() {

		accountBlockHash := getAccountBlockHash(iter.Key())
		accountBlockMeta, getMetaErr := ac.GetBlockMeta(accountBlockHash)
		if getMetaErr != nil {
			return nil, getMetaErr
		}
		if accountBlockMeta.SnapshotHeight > 0 && accountBlockMeta.SnapshotHeight <= snapshotHeight {
			accountBlock := &ledger.AccountBlock{}
			if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr != nil {
				return nil, dsErr
			}

			accountBlock.Hash = *accountBlockHash
			accountBlock.Meta = accountBlockMeta

			return accountBlock, nil
		}
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return nil, nil
}

func (ac *AccountChain) GetUnConfirmAccountBlocks(accountId uint64, beforeHeight uint64) ([]*ledger.AccountBlock, error) {
	accountBlocks := make([]*ledger.AccountBlock, 0)

	var iter iterator.Iterator
	if beforeHeight > 0 {
		startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, 1)
		endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, beforeHeight)
		iter = ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)
	} else {
		key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)
		iter = ac.db.NewIterator(util.BytesPrefix(key), nil)
	}

	defer iter.Release()

	iter.Last()
	for iter.Prev() {
		accountBlockHash := getAccountBlockHash(iter.Key())
		accountBlockMeta, getMetaErr := ac.GetBlockMeta(accountBlockHash)
		if getMetaErr != nil {
			return nil, getMetaErr
		}
		if accountBlockMeta.SnapshotHeight <= 0 {
			accountBlock := &ledger.AccountBlock{}
			if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr != nil {
				return nil, dsErr
			}

			accountBlock.Hash = *accountBlockHash
			accountBlock.Meta = accountBlockMeta

			accountBlocks = append(accountBlocks, accountBlock)
		} else {
			return accountBlocks, nil
		}
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return accountBlocks, nil
}
