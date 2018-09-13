package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

type VmDatabase interface {
	GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int
	AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	GetSnapshotBlock(hash *types.Hash) *ledger.SnapshotBlock
	GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock
	// forward=true return (startHeight, startHeight+count], forward=false return [startHeight-count, start)
	GetSnapshotBlocks(startHeight uint64, count uint64, forward bool) []*ledger.SnapshotBlock

	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock

	Reset()

	IsAddressExisted(addr *types.Address) bool

	GetToken(id *types.TokenTypeId) *ledger.Token
	SetToken(token *ledger.Token)

	SetContractGid(gid *types.Gid, addr *types.Address, open bool)
	SetContractCode(code []byte)
	GetContractCode(addr *types.Address) []byte

	GetStorage(addr *types.Address, key []byte) []byte
	SetStorage(key []byte, value []byte)
	GetStorageHash() *types.Hash

	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash

	NewStorageIterator(prefix []byte) *vm_context.StorageIterator
}
