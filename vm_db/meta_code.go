package vm_db

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDb) SetContractMeta(toAddress types.Address, meta *ledger.ContractMeta) {
	db.unsaved.SetContractMeta(toAddress, meta)
}

func (db *vmDb) GetContractMeta() (*ledger.ContractMeta, error) {
	if db.address == nil {
		return nil, errors.New("no self address")
	}
	meta := db.unsaved.GetContractMeta(*db.address)
	if meta != nil {
		return meta, nil
	}

	return db.chain.GetContractMeta(*db.address)
}

func (db *vmDb) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error) {
	return db.chain.GetContractMetaInSnapshot(contractAddress, snapshotBlock.Height)
}

func (db *vmDb) SetContractCode(code []byte) {
	db.unsaved.SetCode(code)
}

func (db *vmDb) GetContractCode() ([]byte, error) {
	if code := db.unsaved.GetCode(); len(code) > 0 {
		return code, nil
	}

	return db.chain.GetContractCode(*db.address)
}
func (db *vmDb) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}

func (db *vmDb) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return db.unsaved.contractMetaMap
}
func (db *vmDb) GetUnsavedContractCode() []byte {
	return db.unsaved.code
}
