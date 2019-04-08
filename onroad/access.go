package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (manager *Manager) GetOnRoadBlockByAddr(addr *types.Address, pageNum, pageCount uint64) ([]*ledger.AccountBlock, error) {
	log := manager.log.New("method", "newSignalToWorker", "addr", addr)
	hasOnroads, err := manager.chain.HasOnRoadBlocks(*addr)
	if err != nil {
		return nil, err
	}
	if !hasOnroads {
		return nil, nil
	}
	blocks := make([]*ledger.AccountBlock, 0)
	hashList, err := manager.chain.GetOnRoadBlocksHashList(*addr, int(pageNum), int(pageCount))
	if err != nil {
		log.Error(fmt.Sprintf("GetOnRoadBlocksHashList failed, err:%v", err))
		return nil, err
	}
	if hashList == nil {
		return nil, nil
	}
	for _, v := range hashList {
		/*		isReceive, err := manager.chain.IsReceived(v)
				if err != nil {
					return nil, err
				}
				if isReceive {
					log.Error("get find err, already exist")
					continue
				}*/
		onroad, err := manager.chain.GetAccountBlockByHash(v)
		if err != nil {
			log.Error(fmt.Sprintf("GetAccountBlockByHash failed, err:%v", err))
		}
		blocks = append(blocks, onroad)
	}
	return blocks, nil
}

func (manager *Manager) addContractLis(gid types.Gid, f func(address types.Address)) {
	manager.contractListenerMutex.Lock()
	defer manager.contractListenerMutex.Unlock()
	manager.newContractListener[gid] = f
}

func (manager *Manager) removeContractLis(gid types.Gid) {
	manager.contractListenerMutex.Lock()
	defer manager.contractListenerMutex.Unlock()
	delete(manager.newContractListener, gid)
}

func (manager *Manager) insertBlockToPool(block *vm_db.VmAccountBlock) error {
	return manager.pool.AddDirectAccountBlock(block.AccountBlock.AccountAddress, block)
}

func (manager *Manager) hasOnRoadBlocks(addr *types.Address) (bool, error) {
	return manager.chain.HasOnRoadBlocks(*addr)
}

func (manager *Manager) deleteDirect(sendBlock *ledger.AccountBlock) error {
	return manager.chain.DeleteOnRoad(sendBlock.Hash)
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
		if f, ok := manager.newContractListener[meta.Gid]; ok {
			f(block.ToAddress)
		}
	}
}
