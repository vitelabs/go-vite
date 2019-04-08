package vm_db

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDb) Address() *types.Address {
	return db.address
}

func (db *vmDb) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	if db.latestSnapshotBlock == nil {
		if db.latestSnapshotBlockHash == nil {
			return nil, errors.New("No context, db.latestSnapshotBlockHash is nil")
		}
		var err error
		db.latestSnapshotBlock, err = db.chain.GetSnapshotHeaderByHash(*db.latestSnapshotBlockHash)
		if err != nil {
			return nil, err
		} else if db.latestSnapshotBlock == nil {
			return nil, errors.New(fmt.Sprintf("the returned snapshotHeader of db.chain.GetSnapshotHeaderByHash is nil, db.latestSnapshotBlockHash is %s", db.latestSnapshotBlockHash))
		}
	}
	return db.latestSnapshotBlock, nil
}

func (db *vmDb) PrevAccountBlock() (*ledger.AccountBlock, error) {
	if db.prevAccountBlock == nil {
		if db.prevAccountBlockHash == nil {
			return nil, errors.New("No context, db.prevAccountBlockHash is nil")
		}
		if db.prevAccountBlockHash.IsZero() {
			return nil, nil
		}
		var err error
		db.prevAccountBlock, err = db.chain.GetAccountBlockByHash(*db.prevAccountBlockHash)
		if err != nil {
			return nil, err
		} else if db.prevAccountBlock == nil {
			return nil, errors.New(fmt.Sprintf("the returned accountBlock of db.chain.GetAccountBlockByHash is nil, db.prevAccountBlockHash is %s", db.prevAccountBlockHash))
		}
	}
	return db.prevAccountBlock, nil
}

func (db *vmDb) IsContractAccount() (bool, error) {
	if db.address == nil {
		return false, nil
	}

	return db.chain.IsContractAccount(*db.address)
}

func (db *vmDb) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	return db.chain.GetCallDepth(*sendBlockHash)
}

func (db *vmDb) SetCallDepth(callDepth uint16) {
	db.unsaved.SetCallDepth(callDepth)
}
func (db *vmDb) GetUnsavedCallDepth() uint16 {
	return db.unsaved.GetCallDepth()
}

func (db *vmDb) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return db.chain.GetQuotaUsed(*db.address)
}
