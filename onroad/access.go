package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (manager *Manager) insertBlockToPool(block *vm_db.VmAccountBlock) error {
	return manager.pool.AddDirectAccountBlock(block.AccountBlock.AccountAddress, block)
}

func (manager *Manager) checkExistInPool(addr types.Address, fromBlockHash types.Hash) bool {
	return manager.pool.ExistInPool(addr, fromBlockHash)
}

func (manager *Manager) getOnRoadBlockByAddr(addr *types.Address) (*ledger.AccountBlock, error) {
	hashList, err := manager.chain.GetOnRoadBlocksHashList(*addr, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(hashList) > 0 {
		onroad, err := manager.chain.GetAccountBlockByHash(hashList[0])
		if err != nil {
			return nil, err
		}
		return onroad, nil
	}
	return nil, nil
}

func (manager *Manager) hasOnRoadBlocks(addr *types.Address) (bool, error) {
	return manager.chain.HasOnRoadBlocks(*addr)
}

func (manager *Manager) DeleteDirect(sendBlock *ledger.AccountBlock) error {
	//p.dbAccess.store.DeleteMeta(nil, &sendBlock.ToAddress, &sendBlock.Hash)
	return nil
}

func (manager *Manager) AddContractLis(gid types.Gid, f func(address types.Address)) {
	manager.contractListenerMutex.Lock()
	defer manager.contractListenerMutex.Unlock()
	manager.newContractListener[gid] = f
}

func (manager *Manager) RemoveContractLis(gid types.Gid) {
	manager.contractListenerMutex.Lock()
	defer manager.contractListenerMutex.Unlock()
	delete(manager.newContractListener, gid)
}

func (manager *Manager) NewOnroad(blocks []*vm_db.VmAccountBlock) error {
	for _, v := range blocks {
		if v.AccountBlock.IsSendBlock() {
			manager.newSignalToWorker(v.AccountBlock)
		}
	}
	return nil
}

func (manager *Manager) newSignalToWorker(block *ledger.AccountBlock) {
	newLog := manager.log.New("method", "newSignalToWorker", "Hash", block.Hash)
	isContract, err := manager.chain.IsContractAccount(block.AccountAddress)
	if err != nil {
		newLog.Error(fmt.Sprintf("IsContractAccount, err:%v", err))
		return
	}
	if isContract {
		meta, err := manager.chain.GetContractMeta(block.AccountAddress)
		if err != nil {
			newLog.Error(fmt.Sprintf("GetContractMeta, err:%v", err))
			return
		}
		if meta == nil {
			return
		}
		manager.contractListenerMutex.RLock()
		defer manager.contractListenerMutex.RUnlock()
		if f, ok := manager.newContractListener[*meta.Gid]; ok {
			f(block.ToAddress)
		}
	}
}

func (manager *Manager) NewSnapshot(snapshotBlock []*ledger.SnapshotBlock) error {
	manager.interruptSignalToAllWorker()
	return nil
}

func (manager *Manager) interruptSignalToAllWorker() {
}