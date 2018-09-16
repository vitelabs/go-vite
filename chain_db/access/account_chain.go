package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type AccountChain struct {
	db *leveldb.DB
}

func NewAccountChain(db *leveldb.DB) *AccountChain {
	return &AccountChain{
		db: db,
	}
}

func (ac *AccountChain) getBlockHash(dbKey []byte) *types.Hash {
	hashBytes := dbKey[17:]
	hash, _ := types.BytesToHash(hashBytes)
	return &hash
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
		hashList = append(hashList, ac.getBlockHash(iter.Key()))
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

	block.Hash = *ac.getBlockHash(iter.Key())
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

		block.Hash = *ac.getBlockHash(iter.Key())
		blockList = append(blockList, block)
	}

	return blockList, nil
}

func (ac *AccountChain) GetBlock(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	blockMeta, gbmErr := ac.GetBlockMeta(blockHash)
	if gbmErr != nil {
		return nil, gbmErr
	}

	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCK, blockMeta.AccountId, blockMeta.Height)

	block, err := ac.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	accountBlock := &ledger.AccountBlock{}
	accountBlock.DbDeserialize(block)

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

func (ac *AccountChain) GetConfirmHeight(accountBlock *ledger.AccountBlock) (uint64, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, accountBlock.Hash.Bytes())

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	for iter.Next() {
		accountBlockMeta := &ledger.AccountBlockMeta{}
		if dsErr := accountBlockMeta.DbDeserialize(iter.Value()); dsErr != nil {
			return 0, dsErr
		}

		if accountBlockMeta.SnapshotHeight > 0 {
			return accountBlockMeta.SnapshotHeight, nil
		}
	}
	return 0, nil
}

func (ac *AccountChain) WriteVmLogList(batch *leveldb.Batch, logList ledger.VmLogList) error {
	return nil
}

func (ac *AccountChain) WriteContractGid(batch *leveldb.Batch, gid *types.Gid, addr *types.Address, open bool) error {
	return nil
}