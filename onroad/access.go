package onroad

import (
	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad/pool"
	"github.com/vitelabs/go-vite/vm_db"
)

func (manager *Manager) GetOnRoadBlocksByAddr(addr types.Address, pageNum, pageSize int) ([]*ledger.AccountBlock, error) {
	return manager.chain.GetOnRoadBlocksByAddr(addr, pageNum, pageSize)
}

func (manager *Manager) GetAccountOnRoadInfo(addr types.Address) (*ledger.AccountInfo, error) {
	return manager.chain.GetAccountOnRoadInfo(addr)
}

func (manager *Manager) GetOnRoadTotalNumByAddr(gid types.Gid, addr types.Address) (uint64, error) {
	onRoadPool, ok := manager.onRoadPools.Load(gid)
	if !ok || onRoadPool == nil {
		manager.log.Error("contractOnRoadPool is not available", "gid", gid, "addr", addr)
		return 0, errors.New("contractOnRoadPool is not available")
	}
	num, err := onRoadPool.(onroad_pool.OnRoadPool).GetOnRoadTotalNumByAddr(addr)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (manager *Manager) GetOnRoadFrontBlocks(gid types.Gid, addr types.Address) ([]*ledger.AccountBlock, error) {
	onRoadPool, ok := manager.onRoadPools.Load(gid)
	if !ok || onRoadPool == nil {
		manager.log.Error("contractOnRoadPool is not available", "gid", gid, "addr", addr)
		return nil, errors.New("contractOnRoadPool is not available")
	}
	blockList, err := onRoadPool.(onroad_pool.OnRoadPool).GetOnRoadFrontBlocks(addr)
	if err != nil {
		return nil, err
	}
	return blockList, nil
}

func (manager *Manager) deleteDirect(sendBlock *ledger.AccountBlock) error {
	manager.chain.DeleteOnRoad(sendBlock.ToAddress, sendBlock.Hash)
	return nil
}

func (manager *Manager) insertBlockToPool(block *vm_db.VmAccountBlock) error {
	return manager.pool.AddDirectAccountBlock(block.AccountBlock.AccountAddress, block)
}

type reactFunc func(address types.Address)

func (manager *Manager) addContractLis(gid types.Gid, f reactFunc) {
	manager.newContractListener.Store(gid, f)
}

func (manager *Manager) removeContractLis(gid types.Gid) {
	manager.newContractListener.Delete(gid)
}

func (manager *Manager) newSignalToWorker(gid types.Gid, contract types.Address) {
	if f, ok := manager.newContractListener.Load(gid); ok {
		f.(reactFunc)(contract)
	}
}
