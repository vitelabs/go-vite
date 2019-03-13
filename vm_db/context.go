package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDB) Address() *types.Address {
	return db.address
}

func (db *vmDB) LatestSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

func (db *vmDB) PrevAccountBlock() *ledger.AccountBlock {
	return nil
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
