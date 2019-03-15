package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type VmAccountBlock struct {
	AccountBlock *ledger.AccountBlock
	VmDb         VmDb
}

type Chain interface {
	IsContractAccount(address *types.Address) (bool, error)
	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)

	GetSnapshotHeaderByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	GetStateSnapshot(blockHash *types.Hash) (interfaces.StateSnapshot, error)

	GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error)

	GetUnconfirmedBlocks(addr *types.Address) ([]*ledger.AccountBlock, error)

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error)
}

type VmDb interface {
	// ====== Context ======
	Address() *types.Address
	LatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	PrevAccountBlock() (*ledger.AccountBlock, error)

	IsContractAccount() (bool, error)

	GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error) // TODO

	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)

	// ====== State ======
	GetReceiptHash() *types.Hash // TODO
	Reset()

	// ====== Storage ======
	GetValue(key []byte) ([]byte, error)
	GetOriginalValue(key []byte) ([]byte, error)

	SetValue(key []byte, value []byte)

	NewStorageIterator(prefix []byte) interfaces.StorageIterator // TODO

	// ====== Balance ======
	GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)

	// ====== VMLog ======
	AddLog(log *ledger.VmLog)

	GetLogList() ledger.VmLogList
	GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error)
	GetLogListHash() *types.Hash

	// ====== AccountBlock ======
	GetUnconfirmedBlocks() ([]*ledger.AccountBlock, error)

	// ====== SnapshotBlock ======
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	// ====== Meta & Code ======
	SetContractMeta(meta *ledger.ContractMeta)

	SetContractCode(code []byte)

	GetContractCode() ([]byte, error)
	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) // TODO

	// ====== built-in contract ======
	GetPledgeAmount(addr *types.Address) (*big.Int, error)

	// ====== debug ======
	DebugGetStorage() map[string][]byte // TODO
}
