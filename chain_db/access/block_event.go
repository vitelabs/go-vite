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
	AddAccountBlockEvent     = byte(1)
	DeleteAccountBlockEvent  = byte(2)
	AddSnapshotBlockEvent    = byte(3)
	DeleteSnapshotBlockEvent = byte(4)
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

func (be *BlockEvent) AddAccountBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {

	be.writeEvent(batch, AddAccountBlockEvent, blockHashList)
}

func (be *BlockEvent) DeleteAccountBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, DeleteAccountBlockEvent, blockHashList)
}

func (be *BlockEvent) AddSnapshotBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, AddSnapshotBlockEvent, blockHashList)
}

func (be *BlockEvent) DeleteSnapshotBlocks(batch *leveldb.Batch, blockHashList []types.Hash) {
	be.writeEvent(batch, DeleteSnapshotBlockEvent, blockHashList)
}
