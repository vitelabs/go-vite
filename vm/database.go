package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Database interface {
	Balance(addr types.Address, tokenTypeId types.TokenTypeId) *big.Int
	SubBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int)
	AddBalance(addr types.Address, tokenTypeId types.TokenTypeId, amount *big.Int)

	SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock
	SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock

	AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock

	Rollback()

	IsExistAddress(addr types.Address) bool
	CreateAccount(addr types.Address)

	CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool

	SetContractCode(addr types.Address, code []byte)
	ContractCode(addr types.Address) []byte

	Storage(addr types.Address, loc []byte) []byte
	SetStorage(addr types.Address, loc []byte, value []byte)
	StorageString(addr types.Address) string
	StorageHash(addr types.Address) types.Hash

	AddLog(*Log)
	LogListHash() types.Hash
}
