package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type VmAccountBlock struct {
	AccountBlock *ledger.AccountBlock
	VmContext    VmDatabase
}

type VmDatabase interface {
	GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int
	AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	GetSnapshotBlock(hash *types.Hash) *ledger.SnapshotBlock
	GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock
	// forward=true return (startHeight, startHeight+count], forward=false return [startHeight-count, start)
	GetSnapshotBlocks(startHeight uint64, count uint64, forward bool) []*ledger.SnapshotBlock

	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock

	UnsavedCache() *UnsavedCache
	Reset()

	IsAddressExisted(addr *types.Address) bool

	SetContractGid(gid *types.Gid, addr *types.Address, open bool)
	SetContractCode(code []byte)
	GetContractCode(addr *types.Address) []byte

	GetStorage(addr *types.Address, key []byte) []byte
	SetStorage(key []byte, value []byte)
	GetStorageHash() *types.Hash

	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash

	NewStorageIterator(prefix []byte) *StorageIterator

	CopyAndFreeze() VmDatabase
	Address() *types.Address
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock
}
