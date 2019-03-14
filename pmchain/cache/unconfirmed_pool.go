package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

type insertedItem struct {
	dataId  uint64
	eventId uint64
}

type UnconfirmedPool struct {
	ds *dataSet

	mu       sync.RWMutex
	memStore map[types.Address][]uint64

	latestEventId uint64
	insertedList  []*insertedItem
}

func NewUnconfirmedPool(ds *dataSet) *UnconfirmedPool {
	return &UnconfirmedPool{
		ds:       ds,
		memStore: make(map[types.Address][]uint64),
	}
}

//func (up *UnconfirmedPool) Clean() {
//	up.mu.Lock()
//	defer up.mu.Unlock()
//
//	up.memStore = make(map[types.Address][]uint64)
//}

func (up *UnconfirmedPool) InsertAccountBlock(dataId uint64) {
	up.mu.Lock()
	defer up.mu.Unlock()
	eventId := up.newEventId()
	up.insertedList = append(up.insertedList, &insertedItem{
		dataId:  dataId,
		eventId: eventId,
	})

	block := up.ds.GetAccountBlock(dataId)
	up.memStore[block.AccountAddress] = append(up.memStore[block.AccountAddress], dataId)

	up.ds.RefDataId(dataId)
}

// No lock
func (up *UnconfirmedPool) GetCurrentBlocks() []*ledger.AccountBlock {
	up.mu.RLock()
	currentLength := len(up.insertedList)
	currentEventId := up.insertedList[currentLength-1].eventId
	up.mu.RUnlock()

	blocks := make([]*ledger.AccountBlock, 0, len(up.insertedList))
	for _, inserted := range up.insertedList {
		if inserted.eventId > currentEventId {
			break
		}
		blocks = append(blocks, up.ds.GetAccountBlock(inserted.dataId))

	}

	return blocks
}

//func (up *UnconfirmedPool) Snapshot(sc ledger.SnapshotContent) {
//	up.mu.Lock()
//	defer up.mu.Unlock()
//
//	for addr, hashHeight := range sc {
//		index := findIndex(up.memStore[addr], hashHeight)
//		up.memStore[addr] = up.memStore[addr][index+1:]
//	}
//}
//
//// TODO
//func (up *UnconfirmedPool) GetContentNeedSnapshot() ledger.SnapshotContent {
//	up.mu.RLock()
//	defer up.mu.RUnlock()
//
//	sc := make(ledger.SnapshotContent, len(up.memStore))
//	for addr, blocks := range up.memStore {
//		block := blocks[len(blocks)-1]
//		sc[addr] = &ledger.HashHeight{
//			Hash:   block.Hash,
//			Height: block.Height,
//		}
//	}
//	return sc
//}

func (up *UnconfirmedPool) newEventId() uint64 {
	up.latestEventId++
	return up.latestEventId
}

func findIndex(blocks []*ledger.AccountBlock, hashHeight *ledger.HashHeight) int {
	return 0
}
