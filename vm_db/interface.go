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
	IsContractAccount(address types.Address) (bool, error)

	GetQuotaUsedList(address types.Address) []types.QuotaInfo

	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error)

	GetContractCode(contractAddr types.Address) ([]byte, error)

	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)

	GetConfirmSnapshotHeaderByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error)

	GetContractMetaInSnapshot(contractAddress types.Address, snapshotHeight uint64) (meta *ledger.ContractMeta, err error)

	GetSnapshotHeaderByHash(hash types.Hash) (*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error)

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetPledgeBeneficialAmount(addr types.Address) (*big.Int, error)

	GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(addr types.Address, key []byte) ([]byte, error)

	GetCallDepth(sendBlockHash types.Hash) (uint16, error)

	GetSnapshotBlockByContractMeta(addr *types.Address, fromHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error)
}

type VmDb interface {
	// ====== Context ======
	Address() *types.Address

	LatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	PrevAccountBlock() (*ledger.AccountBlock, error)

	IsContractAccount() (bool, error)

	GetCallDepth(sendBlockHash *types.Hash) (uint16, error)

	GetQuotaUsedList(addr types.Address) []types.QuotaInfo

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
	GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock

	// ====== SnapshotBlock ======
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error)

	// ====== Meta & Code ======
	SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta)

	GetContractMeta() (*ledger.ContractMeta, error)

	GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error)

	SetContractCode(code []byte)

	GetContractCode() ([]byte, error)

	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) // TODO

	GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta

	GetUnsavedContractCode() []byte

	// ====== built-in contract ======
	GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error)

	// ====== debug ======
	DebugGetStorage() (map[string][]byte, error)
}
