package vitedb

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"strconv"
)

type AccountChain struct {
	db  *DataBase
	log log15.Logger
}

var _accountchain *AccountChain

func GetAccountChain() *AccountChain {
	db, err := GetLDBDataBase(DB_LEDGER)
	if err != nil {
		log15.Root().Crit(err.Error())
	}

	if _accountchain == nil {
		_accountchain = &AccountChain{
			db:  db,
			log: log15.New("module", "vitedb/account_chain"),
		}
	}

	return _accountchain
}

// Fixme
func (ac *AccountChain) CounterAdd(batch *leveldb.Batch) error {
	key, err := createKey(DBKP_ACCOUNTBLOCK_COUNTER)
	if err != nil {
		return err
	}

	currentCount, counterGetErr := ac.CounterGet()
	if counterGetErr != nil {
		if counterGetErr == leveldb.ErrNotFound {
			currentCount = big.NewInt(0)
		} else {
			return counterGetErr
		}
	}

	count := big.Int{}
	count.Add(currentCount, big.NewInt(1))

	batch.Put(key, count.Bytes())
	return nil
}

// Fixme
func (ac *AccountChain) CounterGet() (*big.Int, error) {
	key, ckErr := createKey(DBKP_ACCOUNTBLOCK_COUNTER)
	if ckErr != nil {
		return nil, ckErr
	}

	val, getErr := ac.db.Leveldb.Get(key, nil)
	if getErr != nil {
		return nil, getErr
	}

	count := big.NewInt(0)

	if val != nil {
		count.SetBytes(val)
	}
	return count, nil
}

func (ac *AccountChain) BatchWrite(batch *leveldb.Batch, writeFunc func(batch *leveldb.Batch) error) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		return writeFunc(context.Batch)
	})
}

func (ac *AccountChain) WriteBlock(batch *leveldb.Batch, accountId *big.Int, accountBlock *ledger.AccountBlock) error {
	buf, err := accountBlock.DbSerialize()
	if err != nil {
		return err
	}
	key, err := createKey(DBKP_ACCOUNTBLOCK, accountId, accountBlock.Meta.Height)
	batch.Put(key, buf)

	return nil
}

func (ac *AccountChain) WriteBlockMeta(batch *leveldb.Batch, accountBlockHash *types.Hash, accountBlockMeta *ledger.AccountBlockMeta) error {
	buf, err := accountBlockMeta.DbSerialize()
	if err != nil {
		return err
	}

	key, err := createKey(DBKP_ACCOUNTBLOCKMETA, accountBlockHash.Bytes())
	batch.Put(key, buf)
	return nil
}

func (ac *AccountChain) GetBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	accountBlockMeta, err := ac.GetBlockMeta(blockHash)
	if err != nil {
		return nil, err
	}

	return ac.GetBlockByHeight(accountBlockMeta.AccountId, accountBlockMeta.Height)
}

func (ac *AccountChain) GetBlockByHeight(accountId *big.Int, blockHeight *big.Int) (*ledger.AccountBlock, error) {
	key, err := createKey(DBKP_ACCOUNTBLOCK, accountId, blockHeight)
	if err != nil {
		return nil, err
	}

	block, err := ac.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, err
	}

	accountBlock := &ledger.AccountBlock{}
	accountBlock.DbDeserialize(block)

	accountBlockMeta, err := ac.GetBlockMeta(accountBlock.Hash)
	if err != nil {
		return nil, err
	}

	accountBlock.Meta = accountBlockMeta

	return accountBlock, nil
}

func (ac *AccountChain) GetLatestBlockByAccountId(accountId *big.Int) (*ledger.AccountBlock, error) {

	latestBlockHeight, err := ac.GetLatestBlockHeightByAccountId(accountId)

	if err != nil || latestBlockHeight == nil {
		return nil, err
	}

	return ac.GetBlockByHeight(accountId, latestBlockHeight)
}

func (ac *AccountChain) GetLatestBlockHeightByAccountId(accountId *big.Int) (*big.Int, error) {
	key, err := createKey(DBKP_ACCOUNTBLOCK, accountId, nil)
	if err != nil {
		return nil, err
	}

	iter := ac.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		ac.log.Info("GetLatestBlockHeightByAccountId failed, because account " + accountId.String() + " doesn't exist.")
		return nil, nil
	}

	lastKey := iter.Key()
	partionList := deserializeKey(lastKey)

	latestBlockHeight := &big.Int{}
	latestBlockHeight.SetBytes(partionList[1])
	return latestBlockHeight, nil
}
func (ac *AccountChain) GetBlocksFromOrigin(originBlockHash *types.Hash, count uint64, forward bool) (ledger.AccountBlockList, error) {
	originBlockMeta, err := ac.GetBlockMeta(originBlockHash)

	if err != nil {
		ac.log.Info("AccountChain.GetBlocksFromOrigin: Get OriginBlockMeta failed.")
		return nil, err
	}
	ac.log.Info("AccountChain.GetBlocksFromOrigin: Get OriginBlockMeta success.")

	accountDb := GetAccount()
	address, err := accountDb.GetAddressById(originBlockMeta.AccountId)
	if err != nil {
		ac.log.Info("AccountChain.GetBlocksFromOrigin: Get Address failed.")

		return nil, err
	}
	ac.log.Info("AccountChain.GetBlocksFromOrigin: Get Address success.")

	account, err := accountDb.GetAccountMetaByAddress(address)
	if err != nil {
		ac.log.Info("AccountChain.GetBlocksFromOrigin: Get Account failed.")
		return nil, err
	}
	ac.log.Info("AccountChain.GetBlocksFromOrigin: Get Account success.")

	var startHeight, endHeight, gap = &big.Int{}, &big.Int{}, &big.Int{}
	gap.SetUint64(count)

	if forward {
		startHeight = originBlockMeta.Height
		endHeight.Add(startHeight, gap)
	} else {
		endHeight = originBlockMeta.Height
		// Because leveldb iterator range is [a, b)
		endHeight.Add(endHeight, big.NewInt(1))
		startHeight.Sub(endHeight, gap)
	}

	ac.log.Info("AccountChain.GetBlocksFromOrigin:", "Start height", startHeight, "End height", endHeight)

	startKey, err := createKey(DBKP_ACCOUNTBLOCK, originBlockMeta.AccountId, startHeight)
	if err != nil {
		return nil, err
	}

	limitKey, err := createKey(DBKP_ACCOUNTBLOCK, originBlockMeta.AccountId, endHeight)
	if err != nil {
		return nil, err
	}

	iter := ac.db.Leveldb.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	var blockList ledger.AccountBlockList

	for count := int64(0); iter.Next(); count++ {
		block := &ledger.AccountBlock{}

		err := block.DbDeserialize(iter.Value())
		if err != nil {
			ac.log.Error("AccountChain.GetBlocksFromOrigin: get failed", "error", err)
			return nil, err
		}

		currentHeight := &big.Int{}
		block.Meta = &ledger.AccountBlockMeta{
			Height: currentHeight.Add(startHeight, big.NewInt(count)),
		}

		block.AccountAddress = address
		block.PublicKey = account.PublicKey
		blockList = append(blockList, block)
	}

	ac.log.Info("AccountChain.GetBlocksFromOrigin: return " + strconv.Itoa(len(blockList)) + " blocks.")
	return blockList, nil
}

func (ac *AccountChain) GetBlockListByAccountMeta(index int, num int, count int, meta *ledger.AccountMeta) ([]*ledger.AccountBlock, error) {
	latestBlockHeight, err := ac.GetLatestBlockHeightByAccountId(meta.AccountId)
	if err != nil {
		return nil, err
	}

	limitIndex := latestBlockHeight
	limitIndex.Add(limitIndex, big.NewInt(1))

	limitKey, err := createKey(DBKP_ACCOUNTBLOCK, meta.AccountId, limitIndex)
	if err != nil {
		return nil, err
	}

	startKey, err := createKey(DBKP_ACCOUNTBLOCK, meta.AccountId, big.NewInt(1))
	if err != nil {
		return nil, err
	}

	iter := ac.db.Leveldb.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, nil
	}

	for i := 0; i < index*count; i++ {
		if !iter.Prev() {
			return nil, nil
		}
	}

	var blockList []*ledger.AccountBlock

	for i := 0; i < num*count; i++ {
		block := &ledger.AccountBlock{}

		err := block.DbDeserialize(iter.Value())
		if err != nil {
			return nil, err
		}

		blockList = append(blockList, block)

		if !iter.Prev() {
			break
		}
	}

	return blockList, nil
}

func (ac *AccountChain) IsBlockExist(blockHash *types.Hash) bool {
	key, err := createKey(DBKP_ACCOUNTBLOCKMETA, blockHash.Bytes())
	if err != nil {
		return false
	}

	blockMetaBytes, err := ac.db.Leveldb.Get(key, nil)
	if err != nil || blockMetaBytes == nil {
		return false
	}

	return true
}

func (ac *AccountChain) GetBlockMeta(blockHash *types.Hash) (*ledger.AccountBlockMeta, error) {
	key, err := createKey(DBKP_ACCOUNTBLOCKMETA, blockHash.Bytes())
	if err != nil {
		return nil, err
	}
	blockMetaBytes, err := ac.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, err
	}

	blockMeta := &ledger.AccountBlockMeta{}
	if err := blockMeta.DbDeserialize(blockMetaBytes); err != nil {
		return nil, err
	}

	return blockMeta, nil
}

// st == SnapshotTimestamp
func (ac *AccountChain) GetLastIdByStHeight(stHeight *big.Int) (*big.Int, error) {
	key, err := createKey(DBKP_SNAPSHOTTIMESTAMP_INDEX, stHeight)
	if err != nil {
		return nil, err
	}

	iter := ac.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	if !iter.Last() {
		return nil, nil
	}

	lastId := &big.Int{}
	lastId.SetBytes(iter.Value())
	return lastId, nil
}

func (ac *AccountChain) GetBlockHashList(index, num, count int) ([]*types.Hash, error) {
	key, err := createKey(DBKP_SNAPSHOTTIMESTAMP_INDEX, nil)
	if err != nil {
		return nil, err
	}

	iter := ac.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, nil
	}

	for i := 0; i < index*count; i++ {
		if !iter.Prev() {
			return nil, nil
		}
	}

	var blocHashList []*types.Hash
	for i := 0; i < num*count; i++ {
		blockHash, err := types.BytesToHash(iter.Value())
		if err != nil {
			return nil, err
		}

		blocHashList = append(blocHashList, &blockHash)

		if !iter.Prev() {
			break
		}
	}

	return blocHashList, nil
}
