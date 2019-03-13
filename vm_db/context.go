package vm_db

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDB) Address() *types.Address {
	return db.address
}

func (db *vmDB) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	if db.latestSnapshotBlock == nil {
		if db.latestSnapshotBlockHash == nil {
			return nil, errors.New("No context, db.latestSnapshotBlockHash is nil")
		}
		var err error
		db.latestSnapshotBlock, err = db.chain.GetSnapshotHeaderByHash(db.latestSnapshotBlockHash)
		if err != nil {
			return nil, err
		}
	}
	return db.latestSnapshotBlock, nil
}

func (db *vmDB) PrevAccountBlock() (*ledger.AccountBlock, error) {
	if db.prevAccountBlock == nil {
		if db.prevAccountBlockHash == nil {
			return nil, errors.New("No context, db.prevAccountBlockHash is nil")
		}
		var err error
		db.prevAccountBlock, err = db.chain.GetAccountBlockByHash(db.latestSnapshotBlockHash)
		if err != nil {
			return nil, err
		}
	}
	return db.prevAccountBlock, nil
}

func (db *vmDB) IsContractAccount() (bool, error) {
	return db.chain.IsContractAccount(db.address)
}

func (db *vmDB) GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error) {
	return 0, nil
}

func (db *vmDB) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return db.chain.GetQuotaUsed(db.address)
}
