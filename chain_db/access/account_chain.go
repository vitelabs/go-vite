package access

import (
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
)

func getAccountBlockHash(dbKey []byte) *types.Hash {
	hashBytes := dbKey[17:]
	hash, _ := types.BytesToHash(hashBytes)
	return &hash
}

func getAccountBlockHeight(dbKey []byte) uint64 {
	heightBytes := dbKey[9:17]
	return binary.BigEndian.Uint64(heightBytes)
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
	buf, err := blockMeta.Serialize()
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
	if ddsErr := block.DbDeserialize(iter.Value()); ddsErr != nil {
		return nil, ddsErr
	}

	block.Hash = *getAccountBlockHash(iter.Key())
	return block, nil
}

func (ac *AccountChain) GetBlockListByAccountId(accountId, startHeight, endHeight uint64, forward bool) ([]*ledger.AccountBlock, error) {
	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, startHeight)
	limitKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, endHeight+1)

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	// cap
	cap := uint64(0)
	if endHeight >= startHeight {
		cap = endHeight - startHeight + 1
	} else {
		return nil, errors.New("endHeight is less than startHeight")
	}

	blockList := make([]*ledger.AccountBlock, 0, cap)

	for iter.Next() {
		block := &ledger.AccountBlock{}
		err := block.DbDeserialize(iter.Value())

		if err != nil {
			return nil, err
		}

		block.Hash = *getAccountBlockHash(iter.Key())
		if forward {
			blockList = append(blockList, block)
		} else {
			// prepend, less garbage
			blockList = append(blockList, nil)
			copy(blockList[1:], blockList)
			blockList[0] = block
		}
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
	if blockMeta == nil {
		return nil, nil
	}

	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, blockMeta.AccountId, blockMeta.Height, blockHash.Bytes())

	data, err := ac.db.Get(key, nil)

	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	accountBlock := &ledger.AccountBlock{}
	if dsErr := accountBlock.DbDeserialize(data); dsErr != nil {
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
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	blockMeta := &ledger.AccountBlockMeta{}
	if err := blockMeta.Deserialize(blockMetaBytes); err != nil {
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

	vmLogList, dErr := ledger.VmLogListDeserialize(data)
	if dErr != nil {
		return nil, err
	}

	return vmLogList, err
}

func (ac *AccountChain) getConfirmHeight(accountBlockHash *types.Hash) (uint64, *ledger.AccountBlockMeta, error) {

	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, accountBlockHash.Bytes())
	data, err := ac.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, nil, err
		}
		return 0, nil, nil
	}

	accountBlockMeta := &ledger.AccountBlockMeta{}
	if dsErr := accountBlockMeta.Deserialize(data); dsErr != nil {
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

	if accountBlockMeta == nil {
		return 0, nil
	}

	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountBlockMeta.AccountId, accountBlockMeta.Height+1)
	endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountBlockMeta.AccountId, helper.MaxUint64)

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
	key, _ := database.EncodeKey(database.DBKP_LOG_LIST, logList.Hash().Bytes())

	buf, err := logList.Serialize()
	if err != nil {
		return err
	}

	batch.Put(key, buf)

	return nil
}

func (ac *AccountChain) DeleteVmLogList(batch *leveldb.Batch, logListHash *types.Hash) {
	if logListHash == nil {
		return
	}

	key, _ := database.EncodeKey(database.DBKP_LOG_LIST, logListHash.Bytes())
	batch.Delete(key)
}

func (ac *AccountChain) GetBlockByHeight(accountId uint64, height uint64) (*ledger.AccountBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, height)

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	if !iter.Last() {
		if err := iter.Error(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	accountBlock := &ledger.AccountBlock{}
	if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr != nil {
		return nil, dsErr
	}

	accountBlock.Hash = *getAccountBlockHash(iter.Key())

	return accountBlock, nil
}

func (ac *AccountChain) GetContractGid(accountId uint64) (*types.Gid, error) {
	genesisBlock, err := ac.GetBlockByHeight(accountId, 1)

	if err != nil {
		return nil, err
	}

	if genesisBlock == nil {
		return nil, nil

	}

	fromBlock, getBlockErr := ac.GetBlock(&genesisBlock.FromBlockHash)
	if getBlockErr == nil {
		return nil, getBlockErr
	}

	if fromBlock == nil {
		return nil, nil
	}

	if fromBlock.BlockType != ledger.BlockTypeSendCreate {
		return nil, nil
	}

	gid := contracts.GetGidFromCreateContractData(fromBlock.Data)
	return &gid, nil
}

func (ac *AccountChain) ReopenSendBlocks(batch *leveldb.Batch, reopenList []*ledger.HashHeight, deletedMap map[uint64]uint64) (map[types.Hash]*ledger.AccountBlockMeta, error) {
	var blockMetas = make(map[types.Hash]*ledger.AccountBlockMeta)
	for _, reopenItem := range reopenList {
		blockMeta, err := ac.GetBlockMeta(&reopenItem.Hash)
		if err != nil {
			return nil, err
		}
		if blockMeta == nil {
			continue
		}

		// The block will be deleted, don't need be write
		if deletedHeight := deletedMap[blockMeta.AccountId]; deletedHeight != 0 && blockMeta.Height >= deletedHeight {
			continue
		}

		newReceiveBlockHeights := blockMeta.ReceiveBlockHeights

		for index, receiveBlockHeight := range blockMeta.ReceiveBlockHeights {
			if receiveBlockHeight == reopenItem.Height {
				newReceiveBlockHeights = append(blockMeta.ReceiveBlockHeights[:index], blockMeta.ReceiveBlockHeights[index+1:]...)
				break
			}
		}
		blockMeta.ReceiveBlockHeights = newReceiveBlockHeights
		writeErr := ac.WriteBlockMeta(batch, &reopenItem.Hash, blockMeta)
		if writeErr != nil {
			return nil, err
		}
		blockMetas[reopenItem.Hash] = blockMeta
	}
	return blockMetas, nil
}

func (ac *AccountChain) deleteChain(batch *leveldb.Batch, accountId uint64, toHeight uint64) ([]*ledger.AccountBlock, error) {
	deletedChain := make([]*ledger.AccountBlock, 0)

	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, toHeight)
	endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, helper.MaxUint64)

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

func (ac *AccountChain) Delete(batch *leveldb.Batch, deleteMap map[uint64]uint64) (map[uint64][]*ledger.AccountBlock, error) {
	deleted := make(map[uint64][]*ledger.AccountBlock)
	for accountId, deleteHeight := range deleteMap {
		deletedChain, err := ac.deleteChain(batch, accountId, deleteHeight)
		if err != nil {
			return nil, err
		}

		if len(deletedChain) > 0 {
			deleted[accountId] = deletedChain
		}
	}

	return deleted, nil
}

func (ac *AccountChain) GetDeleteMapAndReopenList(planToDelete map[uint64]uint64, getAccountByAddress func(*types.Address) (*ledger.Account, error), needExtendDelete, needNoSnapshot bool) (map[uint64]uint64, []*ledger.HashHeight, error) {
	currentNeedDelete := planToDelete

	deleteMap := make(map[uint64]uint64)
	var reopenList []*ledger.HashHeight

	for len(currentNeedDelete) > 0 {
		nextNeedDelete := make(map[uint64]uint64)

		for accountId, needDeleteHeight := range currentNeedDelete {
			endHeight := helper.MaxUint64
			if deleteHeight := deleteMap[accountId]; deleteHeight != 0 {
				if deleteHeight <= needDeleteHeight {
					continue
				}
				endHeight = deleteHeight

				// Pre set
				deleteMap[accountId] = needDeleteHeight
			} else {
				deleteMap[accountId] = needDeleteHeight
			}

			startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, needDeleteHeight)
			endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, endHeight)

			iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)

			for iter.Next() {
				accountBlock := &ledger.AccountBlock{}

				if dsErr := accountBlock.DbDeserialize(iter.Value()); dsErr != nil {
					iter.Release()
					return nil, nil, dsErr
				}

				blockHash := getAccountBlockHash(iter.Key())
				accountBlockMeta, getBmErr := ac.GetBlockMeta(blockHash)
				if getBmErr != nil {
					iter.Release()
					return nil, nil, getBmErr
				}

				if needNoSnapshot && accountBlockMeta.SnapshotHeight > 0 {
					return nil, nil, errors.New("is snapshot")
				}

				if needExtendDelete && accountBlock.IsSendBlock() {

					receiveAccount, getAccountErr := getAccountByAddress(&accountBlock.ToAddress)
					if getAccountErr != nil {
						iter.Release()
						return nil, nil, getAccountErr
					}
					receiveAccountId := receiveAccount.AccountId

					for _, receiveBlockHeight := range accountBlockMeta.ReceiveBlockHeights {
						if receiveBlockHeight > 0 {
							if currentDeleteHeight, nextDeleteHeight := currentNeedDelete[receiveAccountId], nextNeedDelete[receiveAccountId]; !(currentDeleteHeight != 0 && currentDeleteHeight <= receiveBlockHeight ||
								nextDeleteHeight != 0 && nextDeleteHeight <= receiveBlockHeight) {
								nextNeedDelete[receiveAccountId] = receiveBlockHeight
							}
						}
					}
				} else if accountBlock.IsReceiveBlock() {
					reopenList = append(reopenList, &ledger.HashHeight{
						Hash:   accountBlock.FromBlockHash,
						Height: accountBlock.Height,
					})
				}
			}

			if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
				iter.Release()
				return nil, nil, err
			}
			iter.Release()
		}

		currentNeedDelete = nextNeedDelete
	}

	return deleteMap, reopenList, nil
}

// TODO: cache
func (ac *AccountChain) GetPlanToDelete(maxAccountId uint64, snapshotBlockHeight uint64) (map[uint64]uint64, error) {
	planToDelete := make(map[uint64]uint64)

	for i := uint64(1); i <= maxAccountId; i++ {
		blockKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, i)

		iter := ac.db.NewIterator(util.BytesPrefix(blockKey), nil)
		iterOk := iter.Last()

		for iterOk {
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
			iterOk = iter.Prev()
		}

		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			iter.Release()
			return nil, err
		}
		iter.Release()
	}

	return planToDelete, nil
}

func (ac *AccountChain) GetUnConfirmedSubLedger(maxAccountId uint64) (map[uint64][]*ledger.AccountBlock, error) {
	unConfirmedAccountBlocks := make(map[uint64][]*ledger.AccountBlock)
	for i := uint64(1); i <= maxAccountId; i++ {
		blocks, err := ac.GetUnConfirmAccountBlocks(i, 0)
		if err != nil {
			return nil, err
		}
		if len(blocks) > 0 {
			unConfirmedAccountBlocks[i] = blocks
		}
	}
	return unConfirmedAccountBlocks, nil
}

// TODO Add cache, call frequently.
func (ac *AccountChain) GetConfirmAccountBlock(snapshotHeight uint64, accountId uint64) (*ledger.AccountBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId)

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	iterOk := iter.Last()
	for iterOk {
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
		iterOk = iter.Prev()
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return nil, nil
}

func (ac *AccountChain) GetFirstConfirmedBlockBeforeOrAtAbHeight(accountId, accountBlockHeight uint64) (*ledger.AccountBlock, error) {
	startKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, 1)
	endKey, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, accountBlockHeight+1)

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)
	defer iter.Release()

	iterOk := iter.Last()

	var accountBlock *ledger.AccountBlock
	for iterOk {
		tmpAccountBlockHash := getAccountBlockHash(iter.Key())
		tmpAccountBlockMeta, getMetaErr := ac.GetBlockMeta(tmpAccountBlockHash)
		if getMetaErr != nil {
			return nil, getMetaErr
		}

		tmpAccountBlock, err := ac.GetBlock(tmpAccountBlockHash)
		if err != nil {
			return nil, err
		}

		tmpAccountBlock.Hash = *tmpAccountBlockHash
		tmpAccountBlock.Meta = tmpAccountBlockMeta

		if tmpAccountBlock.Height != accountBlockHeight && tmpAccountBlock.Meta.SnapshotHeight > 0 {
			break
		}

		accountBlock = tmpAccountBlock
		iterOk = iter.Prev()
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return accountBlock, nil
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
	iterOk := iter.Last()

	for iterOk {
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

		iterOk = iter.Prev()
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return accountBlocks, nil
}
