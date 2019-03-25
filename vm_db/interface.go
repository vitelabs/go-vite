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

	GetBalance(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error)

	GetContractCode(contractAddr *types.Address) ([]byte, error)
	GetContractMeta(contractAddress *types.Address) (meta *ledger.ContractMeta, err error)

	GetSnapshotHeaderByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error)

	GetUnconfirmedBlocks(addr *types.Address) []*ledger.AccountBlock

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error)

	GetStateIterator(address *types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(addr *types.Address, key []byte) ([]byte, error)
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
	GetReceiptHash() *types.Hash

	Reset()

	// Release memory used in runtime.
	Finish()

	// ====== Storage ======
	GetValue(key []byte) ([]byte, error)

	GetOriginalValue(key []byte) ([]byte, error)

	SetValue(key []byte, value []byte) error

	NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error)

	GetUnsavedStorage() [][2][]byte

	// ====== Balance ======
	GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)

	GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int

	// ====== VMLog ======
	AddLog(log *ledger.VmLog)

	GetLogList() ledger.VmLogList
	GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error)
	GetLogListHash() *types.Hash

	// ====== AccountBlock ======
	GetUnconfirmedBlocks() []*ledger.AccountBlock

	// ====== SnapshotBlock ======
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	// ====== Meta & Code ======
	SetContractMeta(meta *ledger.ContractMeta)

	SetContractCode(code []byte)

	GetContractCode() ([]byte, error)

	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) // TODO

	GetUnsavedContractMeta() *ledger.ContractMeta

	GetUnsavedContractCode() []byte

	// ====== built-in contract ======
	GetPledgeAmount(addr *types.Address) (*big.Int, error)

	// ====== debug ======
	DebugGetStorage() (map[string][]byte, error)
}
