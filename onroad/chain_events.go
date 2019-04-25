package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad/pool"
	"github.com/vitelabs/go-vite/vm_db"
)

func (manager *Manager) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

func (manager *Manager) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	for _, block := range blocks {
		if manager.chain.IsGenesisAccountBlock(block.AccountBlock.Hash) {
			continue
		}

		var addr *types.Address
		if block.AccountBlock.IsSendBlock() {
			addr = &block.AccountBlock.ToAddress
		} else {
			addr = &block.AccountBlock.AccountAddress
		}
		meta, err := manager.chain.GetContractMeta(*addr)
		if err != nil {
			return err
		}
		if meta == nil {
			return nil
		}

		// handle contract addr
		orPool, exist := manager.onRoadPools.Load(meta.Gid)
		if !exist || orPool == nil {
			return nil
		}
		if err := orPool.(onroad_pool.OnRoadPool).WriteAccountBlock(block.AccountBlock); err != nil {
			return err
		}
		if block.AccountBlock.IsSendBlock() {
			manager.newSignalToWorker(meta.Gid, block.AccountBlock.ToAddress)
		} else {
			for _, subSend := range block.AccountBlock.SendBlockList {
				sm, err := manager.chain.GetContractMeta(subSend.ToAddress)
				if err != nil {
					return err
				}
				if sm == nil {
					continue
				}
				manager.newSignalToWorker(sm.Gid, subSend.ToAddress)
			}
		}
	}
	return nil
}

func (manager *Manager) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

func (manager *Manager) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {

	for _, v := range blocks {
		if manager.chain.IsGenesisAccountBlock(v.Hash) {
			continue
		}

		var addr *types.Address
		if v.IsSendBlock() {
			addr = &v.ToAddress
		} else {
			addr = &v.AccountAddress
		}
		meta, err := manager.chain.GetContractMeta(*addr)
		if err != nil {
			return err
		}
		if meta == nil {
			return nil
		}
		orPool, exist := manager.onRoadPools.Load(meta.Gid)
		if !exist || orPool == nil {
			return nil
		}
		return orPool.(onroad_pool.OnRoadPool).DeleteAccountBlock(v)
	}
	return nil
}

func (manager *Manager) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

func (manager *Manager) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

func (manager *Manager) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

func (manager *Manager) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
