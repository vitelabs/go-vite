package vitedb

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

type SnapshotChain struct {
	db  *DataBase
	log log15.Logger
}

var _snapshotChain *SnapshotChain

func GetSnapshotChain() *SnapshotChain {
	if _snapshotChain == nil {
		db, err1 := GetLDBDataBase(DB_LEDGER)
		if err1 != nil {
			log15.Root().Crit(err1.Error())
		}

		_snapshotChain = &SnapshotChain{
			db:  db,
			log: log15.New("module", "vitedb/snapshot_chain"),
		}
	}

	return _snapshotChain
}

func (spc *SnapshotChain) DbBatchWrite(batch *leveldb.Batch) {
	spc.db.Leveldb.Write(batch, nil)
}

func (spc *SnapshotChain) DeleteBlocks(batch *leveldb.Batch, blockHash *types.Hash, count uint64) error {
	height, ghErr := spc.GetHeightByHash(blockHash)
	if ghErr != nil {
		return ghErr
	}

	currentHeight := height
	processCount := uint64(0)

	for currentHeight.Cmp(big.NewInt(0)) > 0 && processCount < count {
		currentBlock, gbbhErr := spc.GetBLockByHeight(currentHeight)
		if gbbhErr != nil {
			return gbbhErr
		}

		heightKey, ckheightErr := createKey(DBKP_SNAPSHOTBLOCK, height)

		if ckheightErr != nil {
			return ckheightErr
		}

		hashKey, ckhashErr := createKey(DBKP_SNAPSHOTBLOCKHASH, currentBlock.Hash.Bytes())

		if ckhashErr != nil {
			return ckhashErr
		}

		batch.Delete(hashKey)
		batch.Delete(heightKey)

		currentHeight = currentHeight.Sub(currentHeight, big.NewInt(1))
		processCount++
	}
	return nil
}

func (spc *SnapshotChain) GetHeightByHash(blockHash *types.Hash) (*big.Int, error) {
	key, err := createKey(DBKP_SNAPSHOTBLOCKHASH, blockHash.Bytes())
	heightBytes, err := spc.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, err
	}

	height := &big.Int{}
	height.SetBytes(heightBytes)
	return height, nil
}

func (spc *SnapshotChain) GetBlockByHash(blockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	blockHeight, ghErr := spc.GetHeightByHash(blockHash)
	if ghErr != nil {
		return nil, ghErr
	}
	snapshotBlcok, gbErr := spc.GetBLockByHeight(blockHeight)
	if gbErr != nil {
		return nil, gbErr
	}
	return snapshotBlcok, nil
}

func (spc *SnapshotChain) GetBlockList(index, num, count int) ([]*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := spc.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	var blockList []*ledger.SnapshotBlock
	if !iter.Last() {
		return nil, errors.New("GetBlockList failed. Cause the SnapshotChain has no block" + iter.Error().Error())
	}

	for i := 0; i < (index+num)*count; i++ {
		if i >= index*count {
			snapshotBLock := &ledger.SnapshotBlock{}
			dsErr := snapshotBLock.DbDeserialize(iter.Value())
			if dsErr != nil {
				return nil, dsErr
			}
			blockList = append(blockList, snapshotBLock)
		}
		if !iter.Prev() {
			if err := iter.Error(); err != nil {
				return nil, err
			}
			break
		}
	}
	return blockList, nil
}

func (spc *SnapshotChain) GetBLockByHeight(blockHeight *big.Int) (*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, blockHeight)
	if ckErr != nil {
		return nil, ckErr
	}
	data, dbErr := spc.db.Leveldb.Get(key, nil)
	if dbErr != nil {
		return nil, dbErr
	}
	sb := &ledger.SnapshotBlock{}
	dsErr := sb.DbDeserialize(data)
	if dsErr != nil {
		return nil, dsErr
	}
	return sb, nil
}

func (spc *SnapshotChain) GetBlocksFromOrigin(originBlockHash *types.Hash, count uint64, forward bool) ([]*ledger.SnapshotBlock, error) {
	originBlock, err := spc.GetBlockByHash(originBlockHash)

	if err != nil {
		return nil, err
	}

	var startHeight, endHeight, gap = &big.Int{}, &big.Int{}, &big.Int{}
	gap.SetUint64(count)

	if forward {
		startHeight = originBlock.Height
		startHeight = startHeight.Add(startHeight, big.NewInt(1))

		endHeight.Add(startHeight, gap)
	} else {
		endHeight = originBlock.Height
		endHeight = endHeight.Add(endHeight, big.NewInt(1))
		startHeight.Sub(endHeight, gap)
	}

	startKey, err := createKey(DBKP_SNAPSHOTBLOCK, startHeight)
	if err != nil {
		return nil, err
	}

	limitKey, err := createKey(DBKP_SNAPSHOTBLOCK, endHeight)
	if err != nil {
		return nil, err
	}

	iter := spc.db.Leveldb.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	var sbList []*ledger.SnapshotBlock

	for iter.Next() {
		sb := &ledger.SnapshotBlock{}
		sdErr := sb.DbDeserialize(iter.Value())
		if sdErr != nil {
			return nil, sdErr
		}

		accountDb := GetAccount()

		account, err := accountDb.GetAccountMetaByAddress(sb.Producer)
		if err != nil {
			return nil, err
		}

		sb.PublicKey = account.PublicKey
		sbList = append(sbList, sb)
	}

	return sbList, nil
}

func (spc *SnapshotChain) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, nil)
	if ckErr != nil {
		return nil, ckErr
	}

	iter := spc.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, errors.New("GetLatestBlock failed. Cause the SnapshotChain has no block")
	}
	sb := &ledger.SnapshotBlock{}
	sdErr := sb.DbDeserialize(iter.Value())
	if sdErr != nil {
		return nil, sdErr
	}

	spc.log.Info("SnapshotChain.GetLatestBlock:", "latest block height", sb.Height)
	return sb, nil
}

func (spc *SnapshotChain) Iterate(iterateFunc func(snapshotBlock *ledger.SnapshotBlock) bool, startBlockHash *types.Hash) error {
	startHeight, err := spc.GetHeightByHash(startBlockHash)
	if err != nil {
		return err
	}

	if startHeight == nil {
		return nil
	}

	startKey, err := createKey(DBKP_SNAPSHOTBLOCK, startHeight)

	iter := spc.db.Leveldb.NewIterator(&util.Range{Start: startKey}, nil)
	defer iter.Release()

	for iter.Next() {
		value := iter.Value()
		if value != nil {
			snapshotBlock := &ledger.SnapshotBlock{}
			snapshotBlock.DbDeserialize(value)
			if !iterateFunc(snapshotBlock) {
				return nil
			}
		}

	}

	return nil
}

func (spc *SnapshotChain) WriteBlock(batch *leveldb.Batch, block *ledger.SnapshotBlock) error {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, block.Height)
	if ckErr != nil {
		return ckErr
	}
	data, sErr := block.DbSerialize()
	if sErr != nil {
		spc.log.Info("SnapshotBlock DbSerialize error.")
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (spc *SnapshotChain) WriteBlockHeight(batch *leveldb.Batch, block *ledger.SnapshotBlock) error {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCKHASH, block.Hash.Bytes())
	if ckErr != nil {
		return ckErr
	}
	batch.Put(key, block.Height.Bytes())
	return nil
}

func (spc *SnapshotChain) BatchWrite(batch *leveldb.Batch, writeFunc func(batch *leveldb.Batch) error) error {
	return batchWrite(batch, spc.db.Leveldb, func(context *batchContext) error {
		err := writeFunc(context.Batch)
		return err
	})
}
