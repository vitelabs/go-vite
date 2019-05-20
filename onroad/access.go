package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad/pool"
	"github.com/vitelabs/go-vite/vm_db"
)

// GetOnRoadTotalNumByAddr method returns the total num of the contract' OnRoad blocks.
func (manager *Manager) GetOnRoadTotalNumByAddr(gid types.Gid, addr types.Address) (uint64, error) {
	onRoadPool, ok := manager.onRoadPools.Load(gid)
	if !ok || onRoadPool == nil {
		manager.log.Error(onroad_pool.ErrOnRoadPoolNotAvailable.Error(), "gid", gid, "addr", addr)
		return 0, onroad_pool.ErrOnRoadPoolNotAvailable
	}
	num, err := onRoadPool.(onroad_pool.OnRoadPool).GetOnRoadTotalNumByAddr(addr)
	if err != nil {
		return 0, err
	}
	return num, nil
}

// GetAllCallersFrontOnRoad method returns all callers's front OnRoad blocks, those with the lowest height,
// in a contract OnRoad pool.
func (manager *Manager) GetAllCallersFrontOnRoad(gid types.Gid, addr types.Address) ([]*ledger.AccountBlock, error) {
	onRoadPool, ok := manager.onRoadPools.Load(gid)
	if !ok || onRoadPool == nil {
		manager.log.Error(onroad_pool.ErrOnRoadPoolNotAvailable.Error(), "gid", gid, "addr", addr)
		return nil, onroad_pool.ErrOnRoadPoolNotAvailable
	}
	blockList, err := onRoadPool.(onroad_pool.OnRoadPool).GetFrontOnRoadBlocksByAddr(addr)
	if err != nil {
		return nil, err
	}
	return blockList, nil
}

// IsFrontOnRoadOfCaller method judges whether is the front OnRoad of a caller in a contract OnRoad pool.
func (manager *Manager) IsFrontOnRoadOfCaller(gid types.Gid, orAddr, caller types.Address, hash types.Hash) (bool, error) {
	onRoadPool, ok := manager.onRoadPools.Load(gid)
	if !ok || onRoadPool == nil {
		manager.log.Error(onroad_pool.ErrOnRoadPoolNotAvailable.Error(), "gid", gid, "addr", orAddr)
		return false, onroad_pool.ErrOnRoadPoolNotAvailable
	}
	return onRoadPool.(onroad_pool.OnRoadPool).IsFrontOnRoadOfCaller(orAddr, caller, hash)
}

func (manager *Manager) deleteDirect(sendBlock *ledger.AccountBlock) {
	manager.chain.DeleteOnRoad(sendBlock.ToAddress, sendBlock.Hash)
}

func (manager *Manager) insertBlockToPool(block *vm_db.VmAccountBlock) error {
	return manager.pool.AddDirectAccountBlock(block.AccountBlock.AccountAddress, block)
}
