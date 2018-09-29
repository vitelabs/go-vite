package access

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func getSnapshotBlockHash(dbKey []byte) *types.Hash {
	hashBytes := dbKey[9:]
	hash, _ := types.BytesToHash(hashBytes)
	return &hash
}

type SnapshotChain struct {
	db *leveldb.DB
}

func NewSnapshotChain(db *leveldb.DB) *SnapshotChain {
	return &SnapshotChain{
		db: db,
	}
}

func (sc *SnapshotChain) WriteSnapshotHash(batch *leveldb.Batch, hash *types.Hash, height uint64) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCKHASH, hash.Bytes())
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	batch.Put(key, heightBytes)
}

func (sc *SnapshotChain) WriteSnapshotContent(batch *leveldb.Batch, snapshotHeight uint64, snapshotContent ledger.SnapshotContent) error {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTCONTENT, snapshotHeight)
	data, sErr := snapshotContent.Serialize()
	if sErr != nil {
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (sc *SnapshotChain) WriteSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock) error {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, snapshotBlock.Height, snapshotBlock.Hash.Bytes())
	data, sErr := snapshotBlock.DbSerialize()
	if sErr != nil {
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (sc *SnapshotChain) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK)

	iter := sc.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		if err := iter.Error(); err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	sb := &ledger.SnapshotBlock{}
	sdErr := sb.Deserialize(iter.Value())

	if sdErr != nil {
		return nil, sdErr
	}

	sb.Hash = *getSnapshotBlockHash(iter.Key())

	var getContentErr error
	sb.SnapshotContent, getContentErr = sc.GetSnapshotContent(sb.Height)
	if getContentErr != nil {
		return nil, getContentErr
	}

	return sb, nil
}

func (sc *SnapshotChain) GetSnapshotContent(snapshotBlockHeight uint64) (ledger.SnapshotContent, error) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTCONTENT, snapshotBlockHeight)
	data, err := sc.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	snapshotContent := ledger.SnapshotContent{}
	snapshotContent.Deserialize(data)

	return snapshotContent, nil
}
func (sc *SnapshotChain) GetSnapshotBlocks(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	var blocks []*ledger.SnapshotBlock
	var startHeight, endHeight = uint64(0), uint64(0)
	if forward {
		startHeight = height
		endHeight = height + count
	} else {
		if height > count {
			startHeight = height - count
		}
		startHeight += 1

		endHeight = height + 1
	}

	startKey, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, startHeight)
	endKey, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, endHeight)

	iter := sc.db.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)

	currentHeight := startHeight
	for i := uint64(0); i < count && iter.Next(); i++ {
		data := iter.Value()
		block := &ledger.SnapshotBlock{}
		if dsErr := block.Deserialize(data); dsErr != nil {
			return blocks, dsErr
		}

		if containSnapshotContent {
			snapshotContent, err := sc.GetSnapshotContent(block.Height)
			if err != nil {
				return blocks, err
			}
			block.SnapshotContent = snapshotContent
		}

		block.Hash = *getSnapshotBlockHash(iter.Key())
		blocks = append(blocks, block)
		currentHeight++
	}

	return blocks, nil
}

func (sc *SnapshotChain) GetSnapshotBlockHeight(snapshotHash *types.Hash) (uint64, error) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCKHASH, snapshotHash)
	data, err := sc.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}

	return binary.BigEndian.Uint64(data), nil

}

func (sc *SnapshotChain) GetSnapshotBlock(height uint64, containsSnapshotContent bool) (*ledger.SnapshotBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, height)

	iter := sc.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Next() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}

	snapshotBlock := &ledger.SnapshotBlock{}
	if dsErr := snapshotBlock.Deserialize(iter.Value()); dsErr != nil {
		return nil, dsErr
	}

	snapshotBlock.Hash = *getSnapshotBlockHash(iter.Key())

	if containsSnapshotContent {
		var getContentErr error
		snapshotBlock.SnapshotContent, getContentErr = sc.GetSnapshotContent(snapshotBlock.Height)
		if getContentErr != nil {
			return nil, getContentErr
		}
	}

	return snapshotBlock, nil
}

// Delete list contains the to height
func (sc *SnapshotChain) DeleteToHeight(batch *leveldb.Batch, toHeight uint64) ([]*ledger.SnapshotBlock, error) {

	deleteList := make([]*ledger.SnapshotBlock, 0)

	startBlockKey, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, toHeight)
	endBlockKey, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, helper.MaxUint64)

	iter := sc.db.NewIterator(&util.Range{Start: startBlockKey, Limit: endBlockKey}, nil)
	defer iter.Release()

	currentHeight := toHeight
	for iter.Next() {
		snapshotBlock := &ledger.SnapshotBlock{}
		if sdErr := snapshotBlock.Deserialize(iter.Value()); sdErr != nil {
			return nil, sdErr
		}

		var getContentErr error
		snapshotBlock.SnapshotContent, getContentErr = sc.GetSnapshotContent(currentHeight)
		if getContentErr != nil {
			return nil, getContentErr
		}

		hash := getSnapshotBlockHash(iter.Key())
		// Delete snapshot block
		batch.Delete(iter.Key())

		snapshotContentKey, _ := database.EncodeKey(database.DBKP_SNAPSHOTCONTENT, currentHeight)
		// Delete snapshot content
		batch.Delete(snapshotContentKey)

		snapshotBlockHashIndex, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCKHASH, hash.Bytes())
		// Delete snapshot hash index
		batch.Delete(snapshotBlockHashIndex)

		deleteList = append(deleteList, snapshotBlock)

		currentHeight++
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		iter.Release()
		return nil, err
	}

	return deleteList, nil
}
