package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type RedoCache struct {
	snapshotLogMap map[uint64]SnapshotLog
	currentHeight  uint64

	retainHeightGap uint64
	mu              sync.RWMutex
}

func NewRedoCache() *RedoCache {
	return &RedoCache{
		retainHeightGap: 12,
	}
}

func (redoCache *RedoCache) Init(currentHeight uint64) {
	redoCache.mu.Lock()
	defer redoCache.mu.Unlock()

	redoCache.snapshotLogMap = make(map[uint64]SnapshotLog)
	redoCache.currentHeight = currentHeight
	redoCache.snapshotLogMap[currentHeight] = make(SnapshotLog)
}

func (redoCache *RedoCache) Current() SnapshotLog {
	redoCache.mu.RLock()
	defer redoCache.mu.RUnlock()

	return redoCache.snapshotLogMap[redoCache.currentHeight]
}

func (redoCache *RedoCache) Get(snapshotHeight uint64) (SnapshotLog, bool) {
	redoCache.mu.RLock()
	defer redoCache.mu.RUnlock()

	snapshotLog, ok := redoCache.snapshotLogMap[snapshotHeight]
	return snapshotLog, ok
}

func (redoCache *RedoCache) Delete(snapshotHeight uint64) {
	redoCache.mu.Lock()
	defer redoCache.mu.Unlock()

	delete(redoCache.snapshotLogMap, snapshotHeight)
	if redoCache.currentHeight >= snapshotHeight {
		redoCache.currentHeight -= 1
	}
}

func (redoCache *RedoCache) Set(snapshotHeight uint64, snapshotLog SnapshotLog) {
	redoCache.mu.Lock()
	defer redoCache.mu.Unlock()

	redoCache.snapshotLogMap[snapshotHeight] = snapshotLog
}

func (redoCache *RedoCache) SetCurrent(snapshotHeight uint64, snapshotLog SnapshotLog) {
	redoCache.mu.Lock()
	defer redoCache.mu.Unlock()

	redoCache.currentHeight = snapshotHeight
	redoCache.snapshotLogMap[snapshotHeight] = snapshotLog
	if uint64(len(redoCache.snapshotLogMap)) > redoCache.retainHeightGap &&
		snapshotHeight > redoCache.retainHeightGap {
		staleHeight := snapshotHeight - redoCache.retainHeightGap

		for height := range redoCache.snapshotLogMap {
			if height <= staleHeight {
				delete(redoCache.snapshotLogMap, height)
			}
		}
	}
}

func (redoCache *RedoCache) AddLog(addr types.Address, log LogItem) {
	redoCache.mu.Lock()
	defer redoCache.mu.Unlock()
	current := redoCache.snapshotLogMap[redoCache.currentHeight]
	current[addr] = append(current[addr], log)
}
