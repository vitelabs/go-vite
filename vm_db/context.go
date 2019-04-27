package vm_db

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (vdb *vmDb) Address() *types.Address {
	return vdb.address
}

func (vdb *vmDb) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	if vdb.latestSnapshotBlock == nil {
		if vdb.latestSnapshotBlockHash == nil {
			return nil, errors.New("No context, vdb.latestSnapshotBlockHash is nil")
		}
		var err error
		vdb.latestSnapshotBlock, err = vdb.chain.GetSnapshotHeaderByHash(*vdb.latestSnapshotBlockHash)
		if err != nil {
			return nil, err
		} else if vdb.latestSnapshotBlock == nil {
			return nil, errors.New(fmt.Sprintf("the returned snapshotHeader of vdb.chain.GetSnapshotHeaderByHash is nil, vdb.latestSnapshotBlockHash is %s", vdb.latestSnapshotBlockHash))
		}
	}
	return vdb.latestSnapshotBlock, nil
}

func (vdb *vmDb) PrevAccountBlock() (*ledger.AccountBlock, error) {
	if vdb.prevAccountBlock == nil {
		if vdb.prevAccountBlockHash == nil {
			return nil, errors.New("No context, vdb.prevAccountBlockHash is nil")
		}
		if vdb.prevAccountBlockHash.IsZero() {
			return nil, nil
		}
		var err error
		vdb.prevAccountBlock, err = vdb.chain.GetAccountBlockByHash(*vdb.prevAccountBlockHash)
		if err != nil {
			return nil, err
		} else if vdb.prevAccountBlock == nil {
			return nil, errors.New(fmt.Sprintf("the returned accountBlock of vdb.chain.GetAccountBlockByHash is nil, vdb.prevAccountBlockHash is %s", vdb.prevAccountBlockHash))
		}
	}
	return vdb.prevAccountBlock, nil
}

func (vdb *vmDb) IsContractAccount() (bool, error) {
	if vdb.address == nil {
		return false, errors.New("No context, vdb.address is nil")
	}

	return vdb.chain.IsContractAccount(*vdb.address)
}

func (vdb *vmDb) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	if vdb.callDepth != nil {
		return *vdb.callDepth, nil
	}

	callDepth, err := vdb.chain.GetCallDepth(*sendBlockHash)
	if err != nil {
		return 0, err
	}
	vdb.callDepth = &callDepth

	return *vdb.callDepth, nil
}

func (vdb *vmDb) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return vdb.chain.GetQuotaUsed(*vdb.address)
}
