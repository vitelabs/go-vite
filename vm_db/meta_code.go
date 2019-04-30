package vm_db

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (vdb *vmDb) SetContractMeta(toAddress types.Address, meta *ledger.ContractMeta) {
	vdb.unsaved().SetContractMeta(toAddress, meta)
}

func (vdb *vmDb) GetContractMeta() (*ledger.ContractMeta, error) {
	if vdb.address == nil {
		return nil, errors.New("no self address")
	}
	meta := vdb.unsaved().GetContractMeta(*vdb.address)
	if meta != nil {
		return meta, nil
	}

	return vdb.chain.GetContractMeta(*vdb.address)
}

func (db *vmDb) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error) {
	return db.chain.GetContractMetaInSnapshot(contractAddress, snapshotBlock.Height)
}

func (db *vmDb) SetContractCode(code []byte) {
	db.unsaved().SetCode(code)
}

func (vdb *vmDb) GetContractCode() ([]byte, error) {
	if code := vdb.unsaved().GetCode(); len(code) > 0 {
		return code, nil
	}

	return vdb.chain.GetContractCode(*vdb.address)
}
func (vdb *vmDb) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}

func (vdb *vmDb) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return vdb.unsaved().contractMetaMap
}
func (vdb *vmDb) GetUnsavedContractCode() []byte {
	return vdb.unsaved().code
}
