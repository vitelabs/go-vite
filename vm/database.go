package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Database interface {
	Balance(addr types.Address, tokenTypeId types.TokenTypeId) *big.Int
	SubBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int)
	AddBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int)

	SnapshotTimestamp(snapshotHash types.Hash) uint64
	SnapshotHeight(snapshotHash types.Hash) *big.Int

	AccountBlock(addr types.Address, blockHash types.Hash) VmBlock

	Revert()

	IsExistAddress(addr types.Address) bool

	CreateAccount(addr types.Address)
	DeleteAccount(addr types.Address)

	SetContractCode(addr types.Address, code []byte, codeHash types.Hash)
	ContractCode(addr types.Address) []byte
	ContractCodeSize(addr types.Address) uint64
	ContractCodeHash(addr types.Address) types.Hash

	State(addr types.Address, loc types.Hash) types.Hash
	SetState(addr types.Address, loc types.Hash, value types.Hash)
	StatesString(addr types.Address) string
	StateHash(addr types.Address) types.Hash

	SnapshotHash(height *big.Int) types.Hash
}
