package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDB) SetContractMeta(meta *ledger.ContractMeta) {
	db.unsaved.SetContractMeta(meta)
}

func (db *vmDB) SetContractCode(code []byte) {
	db.unsaved.SetCode(code)
}

func (db *vmDB) GetContractCode() ([]byte, error) {
	if code := db.unsaved.GetCode(); len(code) > 0 {
		return code, nil
	}

	prevStateSnapshot, err := db.getPrevStateSnapshot()
	if err != nil {
		return nil, err
	}

	return prevStateSnapshot.GetCode()
}
func (db *vmDB) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}
