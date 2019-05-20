package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad/pool"
	"github.com/vitelabs/go-vite/vm_db"
)

// PrepareInsertAccountBlocks method implements and listens to chain trigger event.
func (manager *Manager) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

// InsertAccountBlocks method implements and listens to chain trigger event.
func (manager *Manager) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	sendCreateGidCache := make(map[types.Address]types.Gid)
	blockList := make([]*ledger.AccountBlock, 0)
	for _, v := range blocks {
		if v.AccountBlock.BlockType == ledger.BlockTypeSendCreate {
			newContract := v.AccountBlock.ToAddress
			unsavedMetas := v.VmDb.GetUnsavedContractMeta()
			meta, ok := unsavedMetas[newContract]
			if !ok || meta == nil {
				metaNilErr := errors.New("meta nil in SendCreate vmDb, skip and ignore the block")
				manager.log.Error(metaNilErr.Error(), "contract", newContract)
				return metaNilErr
			}
			sendCreateGidCache[newContract] = meta.Gid
		}
		blockList = append(blockList, v.AccountBlock)
	}

	cutMap := ExcludePairTrades(manager.chain, blockList)
	for addr, list := range cutMap {
		// handle contract onroad
		if !types.IsContractAddr(addr) {
			continue
		}
		var gid types.Gid
		meta, err := manager.chain.GetContractMeta(addr)
		if err != nil {
			panic(fmt.Sprintf("find contract meta err, orAddr %v  err %v ", addr, err))
		}
		if meta != nil {
			gid = meta.Gid
		} else {
			if _, ok := sendCreateGidCache[addr]; !ok {
				panic(fmt.Sprintf("find contract meta nil, orAddr %v", addr))
			}
			gid = sendCreateGidCache[addr]
		}
		orPool, exist := manager.onRoadPools.Load(gid)
		if !exist || orPool == nil {
			return nil
		}
		// insert into OnRoadPool
		if err := orPool.(onroad_pool.OnRoadPool).InsertAccountBlocks(addr, list); err != nil {
			panic(err.Error())
		}

		for _, v := range list {
			// new signal to worker
			if v.IsSendBlock() {
				manager.newContractSignalToWorker(gid, addr)
				break
			}
		}
	}
	return nil
}

// PrepareDeleteAccountBlocks method implements and listens to chain trigger event.
func (manager *Manager) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	cutMap := ExcludePairTrades(manager.chain, blocks)
	for addr, list := range cutMap {
		if !types.IsContractAddr(addr) {
			continue
		}
		meta, err := manager.chain.GetContractMeta(addr)
		if err != nil || meta == nil {
			panic(fmt.Sprintf("find contract meta nil, orAddr %v  err %v ", addr, err))
		}
		orPool, exist := manager.onRoadPools.Load(meta.Gid)
		if !exist || orPool == nil {
			return nil
		}
		// delete from OnRoadPool
		if err := orPool.(onroad_pool.OnRoadPool).DeleteAccountBlocks(addr, list); err != nil {
			panic(err.Error())
		}

		for _, v := range list {
			// new signal to worker
			if v.IsReceiveBlock() {
				manager.newContractSignalToWorker(meta.Gid, addr)
				break
			}
		}
	}
	return nil
}

// DeleteAccountBlocks method implements and listens to chain trigger event.
func (manager *Manager) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

// PrepareInsertSnapshotBlocks method implements and listens to chain trigger event.
func (manager *Manager) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

// InsertSnapshotBlocks method implements and listens to chain trigger event.
func (manager *Manager) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	var latestHeight uint64
	for _, v := range chunks {
		if v.SnapshotBlock.Height > latestHeight {
			latestHeight = v.SnapshotBlock.Height
		}
	}
	manager.log.Debug(fmt.Sprintf("InsertSnapshotBlocks latestHeight %v", latestHeight), "event", "snapshotEvent")
	manager.newSnapshotListener.Range(func(key interface{}, value interface{}) bool {
		manager.log.Info(fmt.Sprintf("snapshot line changed, latestHeight %v gid %v", latestHeight, key.(types.Gid)), "event", "snapshotEvent")
		value.(snapshotEventReactFunc)(latestHeight)
		return true
	})
	return nil
}

// PrepareDeleteSnapshotBlocks method implements and listens to chain trigger event.
func (manager *Manager) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	blocks := make([]*ledger.AccountBlock, 0)
	for _, v := range chunks {
		blocks = append(blocks, v.AccountBlocks...)
	}

	cutMap := ExcludePairTrades(manager.chain, blocks)
	for addr, list := range cutMap {
		if !types.IsContractAddr(addr) {
			continue
		}
		meta, err := manager.chain.GetContractMeta(addr)
		if err != nil || meta == nil {
			panic(fmt.Sprintf("find contract meta nil, orAddr %v  err %v ", addr, err))
		}
		orPool, exist := manager.onRoadPools.Load(meta.Gid)
		if !exist || orPool == nil {
			return nil
		}
		// delete from OnRoadPool
		if err := orPool.(onroad_pool.OnRoadPool).DeleteAccountBlocks(addr, list); err != nil {
			panic(err.Error())
		}

		for _, v := range list {
			// new signal to worker
			if v.IsReceiveBlock() {
				manager.newContractSignalToWorker(meta.Gid, addr)
				break
			}
		}
	}
	return nil
}

// DeleteSnapshotBlocks method implements and listens to chain trigger event.
func (manager *Manager) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

type contractReactFunc func(address types.Address)

func (manager *Manager) addContractLis(gid types.Gid, f contractReactFunc) {
	manager.newContractListener.Store(gid, f)
}

func (manager *Manager) removeContractLis(gid types.Gid) {
	manager.newContractListener.Delete(gid)
}

func (manager *Manager) newContractSignalToWorker(gid types.Gid, contract types.Address) {
	if f, ok := manager.newContractListener.Load(gid); ok {
		f.(contractReactFunc)(contract)
	}
}

type snapshotEventReactFunc func(height uint64)

func (manager *Manager) addSnapshotEventLis(gid types.Gid, f snapshotEventReactFunc) {
	manager.newSnapshotListener.Store(gid, f)
}

func (manager *Manager) removeSnapshotEventLis(gid types.Gid) {
	manager.newSnapshotListener.Delete(gid)
}

func (manager *Manager) snapshotEventSignalToWorker(gid types.Gid, height uint64) {
	//manager.snapshotMutex.RLock()
	if f, ok := manager.newSnapshotListener.Load(gid); ok {
		manager.log.Info(fmt.Sprintf("snapshot line changed, latestHeight %v gid %v", height, gid), "event", "snapshotEvent")
		f.(snapshotEventReactFunc)(height)
	}
	// manager.snapshotMutex.RUnlock()
}
