package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type StorageIterator interface {
	Next() (key, value []byte, ok bool)
}

type Chain interface {
	IsContractAccount(address *types.Address) (bool, error)
	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)
}

type VMDB interface {
	// ====== Context ======
	Address() *types.Address
	LatestSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock

	IsContractAccount() (bool, error)

	GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error)

	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)

	// ====== State ======
	GetReceiptHash() *types.Hash
	Reset()

	// ====== Storage ======
	GetValue(key []byte) []byte
	GetOriginalValue(key []byte) []byte

	SetValue(key []byte, value []byte)
	DeleteValue(key []byte)

	NewStorageIterator(prefix []byte) StorageIterator

	// ====== Balance ======
	GetBalance(tokenTypeId *types.TokenTypeId) *big.Int
	AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)

	// ====== VMLog ======
	AddLog(log *ledger.VmLog)
	GetLogList() ledger.VmLogList
	GetLogListHash() *types.Hash

	// ====== AccountBlock ======
	GetUnconfirmedBlocks() ([]*ledger.AccountBlock, error)

	// ====== SnapshotBlock ======
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	// ====== Meta & Code ======
	SetContractMeta(meta *ledger.ContractMeta)

	SetContractCode(code []byte)

	GetContractCode() ([]byte, error)
	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error)

	// ====== built-in contract ======
	GetPledgeAmount(addr *types.Address) (*big.Int, error)

	// ====== debug ======
	DebugGetStorage() map[string][]byte
}
