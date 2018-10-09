package access

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type BlockEvent struct {
	db *leveldb.DB

	log           log15.Logger
	eventIdLock   sync.RWMutex
	latestEventId uint64
}

func NewBlockEvent(db *leveldb.DB) *BlockEvent {
	blockEvent := &BlockEvent{
		db:  db,
		log: log15.New("module", "chain_db/block_event"),
	}

	latestEventId, err := blockEvent.getLatestEventId()
	if err != nil {
		blockEvent.log.Crit("GetLatestEventId failed, error is "+err.Error(), "method", "NewBlockEvent")
	}
	blockEvent.latestEventId = latestEventId
	return blockEvent
}

var (
	AddAccountBlocksEvent     = byte(1)
	DeleteAccountBlocksEvent  = byte(2)
	AddSnapshotBlocksEvent    = byte(3)
	DeleteSnapshotBlocksEvent = byte(4)
)

func (be *BlockEvent) newEventId() uint64 {
	be.eventIdLock.Lock()
	defer be.eventIdLock.Unlock()

	be.latestEventId++
	return be.latestEventId
}

func (be *BlockEvent) writeEvent(batch *leveldb.Batch, eventPrefix byte, blockHashList []types.Hash) {
	if len(blockHashList) <= 0 {
		return
	}
	eventId := be.newEventId()
	key, _ := database.EncodeKey(database.DBKP_BLOCK_EVENT, eventId)

	var buf []byte
	for _, blockHash := range blockHashList {
		buf = append(buf, blockHash.Bytes()...)
	}
	value := append([]byte{eventPrefix}, buf...)
	batch.Put(key, value)
}

func (be *BlockEvent) getLatestEventId() (uint64, error) {
	iter := be.db.NewIterator(util.BytesPrefix([]byte{database.DBKP_BLOCK_EVENT}), nil)
	iterOk := iter.Last()
	if !iterOk {
		if iterErr := iter.Error(); iterErr != nil && iterErr != leveldb.ErrNotFound {
			return 0, iterErr
		}
		return 0, nil
	}

	return binary.BigEndian.Uint64(iter.Value()), nil
}

func (be *BlockEvent) LatestEventId() uint64 {
	be.eventIdLock.RLock()
	defer be.eventIdLock.RUnlock()

	return be.latestEventId
}

func (be *BlockEvent) GetEvent(eventId uint64) (byte, []types.Hash, error) {
	key, _ := database.EncodeKey(database.DBKP_BLOCK_EVENT, eventId)
	value, err := be.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return byte(0), nil, err
		}
		return byte(0), nil, nil
	}

	eventType := value[0]
	value = value[1:]

	var blockHashList []types.Hash
	for i := 0; i < types.HashSize/len(value); i++ {
		var blockHash types.Hash
		copy(blockHash[:], value[i*types.HashSize:(i+1)*types.HashSize])
		blockHashList = append(blockHashList, blockHash)
	}
	return eventType, blockHashList, nil
}

func (be *BlockEvent) AddAccountBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, AddAccountBlocksEvent, blockHashList)
}

func (be *BlockEvent) DeleteAccountBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, DeleteAccountBlocksEvent, blockHashList)
}

func (be *BlockEvent) AddSnapshotBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, AddSnapshotBlocksEvent, blockHashList)
}

func (be *BlockEvent) DeleteSnapshotBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, DeleteSnapshotBlocksEvent, blockHashList)
}
