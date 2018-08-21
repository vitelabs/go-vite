package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type NoDatabase struct{}

func (db *NoDatabase) Balance(addr types.Address, tokenTypeId types.TokenTypeId) *big.Int {
	return big.NewInt(1000)
}
func (db *NoDatabase) SubBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int) {
}
func (db *NoDatabase) AddBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int) {
}
func (db *NoDatabase) SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock { return nil }
func (db *NoDatabase) SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock { return nil }
func (db *NoDatabase) SnapshotHeight(snapshotHash types.Hash) *big.Int       { return nil }
func (db *NoDatabase) AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock {
	return nil
}
func (db *NoDatabase) Rollback()                                               {}
func (db *NoDatabase) IsExistAddress(addr types.Address) bool                  { return false }
func (db *NoDatabase) CreateAccount(addr types.Address)                        {}
func (db *NoDatabase) SetContractCode(addr types.Address, code []byte)         {}
func (db *NoDatabase) ContractCode(addr types.Address) []byte                  { return nil }
func (db *NoDatabase) Storage(addr types.Address, loc []byte) []byte           { return nil }
func (db *NoDatabase) SetStorage(addr types.Address, loc []byte, value []byte) {}
func (db *NoDatabase) StorageString(addr types.Address) string                 { return "" }
func (db *NoDatabase) StorageHash(addr types.Address) types.Hash               { return types.Hash{} }
