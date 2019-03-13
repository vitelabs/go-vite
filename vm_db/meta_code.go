package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDB) SetContractMeta(meta *ledger.ContractMeta) {

}

func (db *vmDB) SetContractCode(code []byte) {}

func (db *vmDB) GetContractCode() ([]byte, error) {
	return nil, nil
}
func (db *vmDB) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}
