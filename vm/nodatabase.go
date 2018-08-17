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
func (db *NoDatabase) SnapshotTimestamp(snapshotHash types.Hash) uint64                     { return 0 }
func (db *NoDatabase) SnapshotHeight(snapshotHash types.Hash) *big.Int                      { return nil }
func (db *NoDatabase) AccountBlock(addr types.Address, blockHash types.Hash) VmBlock        { return nil }
func (db *NoDatabase) Revert()                                                              {}
func (db *NoDatabase) IsExistAddress(addr types.Address) bool                               { return false }
func (db *NoDatabase) CreateAccount(addr types.Address)                                     {}
func (db *NoDatabase) DeleteAccount(addr types.Address)                                     {}
func (db *NoDatabase) SetContractCode(addr types.Address, code []byte, codeHash types.Hash) {}
func (db *NoDatabase) ContractCode(addr types.Address) []byte                               { return nil }
func (db *NoDatabase) ContractCodeSize(addr types.Address) uint64                           { return 0 }
func (db *NoDatabase) ContractCodeHash(addr types.Address) types.Hash                       { return types.Hash{} }
func (db *NoDatabase) State(addr types.Address, loc types.Hash) types.Hash                  { return types.Hash{} }
func (db *NoDatabase) SetState(addr types.Address, loc types.Hash, value types.Hash)        {}
func (db *NoDatabase) StatesString(addr types.Address) string                               { return "" }
func (db *NoDatabase) StateHash(addr types.Address) types.Hash                              { return types.Hash{} }
func (db *NoDatabase) SnapshotHash(height *big.Int) types.Hash                              { return types.Hash{} }
