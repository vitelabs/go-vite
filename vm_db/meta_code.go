package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDb) SetContractMeta(meta *ledger.ContractMeta) {
	db.unsaved.SetContractMeta(meta)
}

func (db *vmDb) SetContractCode(code []byte) {
	db.unsaved.SetCode(code)
}

func (db *vmDb) GetContractCode() ([]byte, error) {
	if code := db.unsaved.GetCode(); len(code) > 0 {
		return code, nil
	}

	return db.chain.GetContractCode(db.address)
}
func (db *vmDb) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}

func (db *vmDb) GetUnsavedContractMeta() *ledger.ContractMeta {
	return db.unsaved.contractMeta
}
func (db *vmDb) GetUnsavedContractCode() []byte {
	return db.unsaved.code
}
