package chain_state

import (
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/memdb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"time"
)

type EventListener interface {
	PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error
	InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error

	PrepareInsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error
	InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error

	PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error
	DeleteAccountBlocks(blocks []*ledger.AccountBlock) error

	PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
}

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	SBPReader() core.SBPStatReader
}

type TimeIndex interface {
	Index2Time(index uint64) (time.Time, time.Time)
	Time2Index(t time.Time) uint64
}

type Chain interface {
	IterateAccounts(iterateFunc func(addr types.Address, accountId uint64, err error) bool)

	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSnapshotHeightByHash(hash types.Hash) (uint64, error)

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)

	// header without snapshot content
	GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error)

	StopWrite()

	RecoverWrite()
}

type RoundCacheInterface interface {
	Init(timeIndex core.TimeIndex) (returnErr error)
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, snapshotLog SnapshotLog) (returnErr error)
	DeleteSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error
	GetSnapshotViteBalanceList(snapshotHash types.Hash, addrList []types.Address) (map[types.Address]*big.Int, []types.Address, error)
	StorageIterator(snapshotHash types.Hash) interfaces.StorageIterator
	getCurrentData(snapshotHash types.Hash) *memdb.DB
	initRounds(startRoundIndex, endRoundIndex uint64) ([]*RedoCacheData, error)
	queryCurrentData(roundIndex uint64) (*memdb.DB, *ledger.SnapshotBlock, error)
	queryRedoLogs(roundIndex uint64) (returnRedoLogs *RoundCacheRedoLogs, isStoreRedoLogs bool, returnErr error)
	buildCurrentData(prevCurrentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *memdb.DB
	roundToLastSnapshotBlock(roundIndex uint64) (*ledger.SnapshotBlock, error)
	getRoundSnapshotBlocks(roundIndex uint64) ([]*ledger.SnapshotBlock, error)
	setAllBalanceToCache(roundData *memdb.DB, snapshotHash types.Hash) error
	setBalanceToCache(roundData *memdb.DB, snapshotHash types.Hash, addressList []types.Address) error
	setStorageToCache(roundData *memdb.DB, contractAddress types.Address, snapshotHash types.Hash) error
}

type StateDBInterface interface {
	NewStorageIterator(addr types.Address, prefix []byte) interfaces.StorageIterator
	NewSnapshotStorageIteratorByHeight(snapshotHeight uint64, addr types.Address, prefix []byte) (interfaces.StorageIterator, error)
	NewSnapshotStorageIterator(snapshotHash types.Hash, addr types.Address, prefix []byte) (interfaces.StorageIterator, error)
	NewRawSnapshotStorageIteratorByHeight(snapshotHeight uint64, addr types.Address, prefix []byte) interfaces.StorageIterator
	RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk, newUnconfirmedBlocks []*ledger.AccountBlock) error
	RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error
	rollbackByRedo(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, redoLogMap map[types.Address][]LogItem,
		rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{}) error
	recoverLatestIndexToSnapshot(batch *leveldb.Batch, hashHeight ledger.HashHeight, keySetMap map[types.Address]map[string]struct{}, tokenSetMap map[types.Address]map[types.TokenTypeId]struct{}) error
	recoverLatestIndexByRedo(batch *leveldb.Batch, addrMap map[types.Address]struct{}, redoLogMap map[types.Address][]LogItem, rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{})
	rollbackAccountBlock(batch *leveldb.Batch, accountBlock *ledger.AccountBlock)
	recoverToSnapshot(batch *leveldb.Batch, snapshotHeight uint64, unconfirmedLog map[types.Address][]LogItem, addrMap map[types.Address]struct{}) error
	recoverStorageToSnapshot(batch *leveldb.Batch, height uint64, addr types.Address, keySet map[string][]byte) error
	recoverBalanceToSnapshot(batch *leveldb.Batch, height uint64, addr types.Address, tokenSet map[types.TokenTypeId]*big.Int) error
	compactHistoryStorage()
	deleteContractMeta(batch interfaces.Batch, key []byte)
	deleteBalance(batch interfaces.Batch, key []byte)
	deleteHistoryKey(batch interfaces.Batch, key []byte)
	rollbackRoundCache(deletedSnapshotSegments []*ledger.SnapshotChunk) error
	Init() error
	Close() error
	SetConsensus(cs Consensus) error
	GetStorageValue(addr *types.Address, key []byte) ([]byte, error)
	GetBalance(addr types.Address, tokenTypeId types.TokenTypeId) (*big.Int, error)
	GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error)
	GetCode(addr types.Address) ([]byte, error)
	GetContractMeta(addr types.Address) (*ledger.ContractMeta, error)
	IterateContracts(iterateFunc func(addr types.Address, meta *ledger.ContractMeta, err error) bool)
	HasContractMeta(addr types.Address) (bool, error)
	GetContractList(gid *types.Gid) ([]types.Address, error)
	GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error)
	GetCallDepth(sendBlockHash *types.Hash) (uint16, error)
	GetSnapshotBalanceList(balanceMap map[types.Address]*big.Int, snapshotBlockHash types.Hash, addrList []types.Address, tokenId types.TokenTypeId) error
	GetSnapshotValue(snapshotBlockHeight uint64, addr types.Address, key []byte) ([]byte, error)
	SetCacheLevelForConsensus(level uint32)
	Store() *chain_db.Store
	RedoStore() *chain_db.Store
	Redo() RedoInterface
	GetStatus() []interfaces.DBStatus
	getSnapshotBalanceList(balanceMap map[types.Address]*big.Int, snapshotBlockHash types.Hash, addrList []types.Address, tokenId types.TokenTypeId) error
	NewStorageDatabase(snapshotHash types.Hash, addr types.Address) (StorageDatabaseInterface, error)
	newCache() error
	initCache() error
	disableCache()
	enableCache()
	initSnapshotValueCache() error
	initContractMetaCache() error
	getValue(key []byte, cachePrefix string) ([]byte, error)
	getValueInCache(key []byte, cachePrefix string) ([]byte, error)
	parseStorageKey(key []byte) []byte
	copyValue(value []byte) []byte
	Write(block *vm_db.VmAccountBlock) error
	WriteByRedo(blockHash types.Hash, addr types.Address, redoLog LogItem)
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error
	writeContractMeta(batch interfaces.Batch, key, value []byte)
	writeBalance(batch interfaces.Batch, key, value []byte)
	writeHistoryKey(batch interfaces.Batch, key, value []byte)
	canWriteVmLog(addr types.Address) bool
}

type StorageDatabaseInterface interface {
	GetValue(key []byte) ([]byte, error)
	NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error)
	Address() *types.Address
}
type RedoInterface interface {
	initCache() error
	Close() error
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock)
	HasRedo(snapshotHeight uint64) (bool, error)
	QueryLog(snapshotHeight uint64) (SnapshotLog, bool, error)
	SetCurrentSnapshot(snapshotHeight uint64, logMap SnapshotLog)
	AddLog(addr types.Address, log LogItem)
	Rollback(chunks []*ledger.SnapshotChunk)
}
