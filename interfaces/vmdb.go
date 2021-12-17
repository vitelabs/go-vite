package interfaces

import (
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

type VmAccountBlock struct {
	AccountBlock *core.AccountBlock
	VmDb         VmDb
}

type VmDb interface {
	// ====== Context ======
	CanWrite() bool

	Address() *types.Address

	LatestSnapshotBlock() (*core.SnapshotBlock, error)

	PrevAccountBlock() (*core.AccountBlock, error)

	GetLatestAccountBlock(addr types.Address) (*core.AccountBlock, error)

	GetCallDepth(sendBlockHash *types.Hash) (uint16, error)

	GetQuotaUsedList(addr types.Address) []types.QuotaInfo

	GetGlobalQuota() types.QuotaInfo

	// ====== State ======
	GetReceiptHash() *types.Hash

	Reset()

	// Release memory used in runtime.
	Finish()

	// ====== Storage ======
	GetValue(key []byte) ([]byte, error)

	GetOriginalValue(key []byte) ([]byte, error)

	SetValue(key []byte, value []byte) error

	NewStorageIterator(prefix []byte) (StorageIterator, error)

	GetUnsavedStorage() [][2][]byte

	// ====== Balance ======
	GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)

	GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int

	// ====== VMLog ======
	AddLog(log *core.VmLog)

	GetLogList() core.VmLogList
	GetHistoryLogList(logHash *types.Hash) (core.VmLogList, error)
	GetLogListHash() *types.Hash

	// ====== AccountBlock ======
	GetUnconfirmedBlocks(address types.Address) []*core.AccountBlock

	// ====== SnapshotBlock ======
	GetGenesisSnapshotBlock() *core.SnapshotBlock

	GetConfirmSnapshotHeader(blockHash types.Hash) (*core.SnapshotBlock, error)

	GetConfirmedTimes(blockHash types.Hash) (uint64, error)

	GetSnapshotBlockByHeight(height uint64) (*core.SnapshotBlock, error)

	// ====== Meta & Code ======
	SetContractMeta(toAddr types.Address, meta *core.ContractMeta)

	GetContractMeta() (*core.ContractMeta, error)

	GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *core.SnapshotBlock) (meta *core.ContractMeta, err error)

	SetContractCode(code []byte)

	GetContractCode() ([]byte, error)

	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *core.SnapshotBlock) ([]byte, error) // TODO

	GetUnsavedContractMeta() map[types.Address]*core.ContractMeta

	GetUnsavedContractCode() []byte

	// ====== built-in contract ======
	GetStakeBeneficialAmount(addr *types.Address) (*big.Int, error)

	// ====== debug ======
	DebugGetStorage() (map[string][]byte, error)
}
