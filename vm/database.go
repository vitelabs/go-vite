package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type VmDatabase interface {
	Balance(addr types.Address, tokenId types.TokenTypeId) *big.Int
	SubBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)
	AddBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)

	SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock
	SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock

	AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock

	Rollback()

	IsExistAddress(addr types.Address) bool

	CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool

	SetContractCode(addr types.Address, code []byte)
	ContractCode(addr types.Address) []byte

	Storage(addr types.Address, loc types.Hash) types.Hash
	SetStorage(addr types.Address, loc, value types.Hash)
	PrintStorage(addr types.Address) string
	StorageHash(addr types.Address) types.Hash

	AddLog(*Log)
	LogListHash() types.Hash
}
