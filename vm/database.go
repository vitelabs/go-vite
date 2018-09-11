package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Log struct {
	// list of topics provided by the contract
	Topics []types.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
}

type VmDatabase interface {
	Balance(addr types.Address, tokenId types.TokenTypeId) *big.Int
	SubBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)
	AddBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)

	SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock
	SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock
	// forward=true return (startHeight, startHeight+count], forward=false return [startHeight-count, start)
	SnapshotBlockList(startHeight *big.Int, count uint64, forward bool) []VmSnapshotBlock

	AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock

	Rollback()

	IsExistAddress(addr types.Address) bool

	IsExistToken(tokenId types.TokenTypeId) bool
	CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool

	SetContractGid(addr types.Address, gid types.Gid, open bool)
	SetContractCode(addr types.Address, gid types.Gid, code []byte)
	ContractCode(addr types.Address) []byte

	Storage(addr types.Address, loc types.Hash) []byte
	SetStorage(addr types.Address, loc types.Hash, value []byte)
	PrintStorage(addr types.Address) string
	StorageHash(addr types.Address) types.Hash

	AddLog(*Log)
	LogListHash() types.Hash

	GetPledgeAmount(beneficial types.Address) *big.Int

	GetDbIteratorByPrefix(prefix []byte) DbIterator
}

type DbIterator interface {
	HasNext() bool
	Next() (key, value []byte)
}
