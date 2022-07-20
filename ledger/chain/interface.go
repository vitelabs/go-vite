package chain

import (
	"math/big"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_block "github.com/vitelabs/go-vite/v2/ledger/chain/block"
	chain_flusher "github.com/vitelabs/go-vite/v2/ledger/chain/flusher"
	chain_index "github.com/vitelabs/go-vite/v2/ledger/chain/index"
	chain_plugins "github.com/vitelabs/go-vite/v2/ledger/chain/plugins"
	chain_state "github.com/vitelabs/go-vite/v2/ledger/chain/state"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
)

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	SBPReader() core.SBPStatReader
	VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error)
}

type Chain interface {
	/*
	 *	Lifecycle
	 */
	Init() error

	Start() error

	Stop() error

	Destroy() error

	/*
	*	Event Manager
	 */
	Register(listener interfaces.EventListener)
	UnRegister(listener interfaces.EventListener)

	/*
	 *	C(Create)
	 */

	// vmAccountBlocks must have the same address
	InsertAccountBlock(vmAccountBlocks *interfaces.VmAccountBlock) error

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error)

	/*
	 *	D(Delete)
	 */

	DeleteAccountBlocks(addr types.Address, toHash types.Hash) ([]*ledger.AccountBlock, error)

	DeleteAccountBlocksToHeight(addr types.Address, toHeight uint64) ([]*ledger.AccountBlock, error)

	// contain the snapshot block of toHash, delete all blocks higher than snapshot line
	DeleteSnapshotBlocks(toHash types.Hash) ([]*ledger.SnapshotChunk, error)

	// contain the snapshot block of toHeight`, delete all blocks higher than snapshot line
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error)

	/*
	 *	R(Retrieve)
	 */

	// ====== Query account block ======
	IsGenesisAccountBlock(hash types.Hash) bool

	IsAccountBlockExisted(hash types.Hash) (bool, error) // ok

	GetAccountBlockByHeight(addr types.Address, height uint64) (*ledger.AccountBlock, error)

	GetAccountBlockHashByHeight(addr types.Address, height uint64) (*types.Hash, error)

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	// query receive block of send block
	GetReceiveAbBySendAb(sendBlockHash types.Hash) (*ledger.AccountBlock, error)

	// is received
	IsReceived(sendBlockHash types.Hash) (bool, error)

	// high to low, contains the block that has the blockHash
	GetAccountBlocks(blockHash types.Hash, count uint64) ([]*ledger.AccountBlock, error)

	GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	GetAccountBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*ledger.AccountBlock, error)

	// [start,end]
	GetAccountBlocksByRange(addr types.Address, start uint64, end uint64) ([]*ledger.AccountBlock, error)

	// get call depth
	GetCallDepth(sendBlock types.Hash) (uint16, error)

	// judge the account block is confirmed by the N or more than N snapshot blocks with seed
	IsSeedConfirmedNTimes(blockHash types.Hash, n uint64) (bool, error)

	// get confirmed times
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)

	// get latest account block
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)

	GetLatestAccountHeight(addr types.Address) (uint64, error)

	// ====== Query snapshot block ======
	IsGenesisSnapshotBlock(hash types.Hash) bool

	IsSnapshotBlockExisted(hash types.Hash) (bool, error) // ok

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	// get height
	GetSnapshotHeightByHash(hash types.Hash) (uint64, error)

	// header without snapshot content
	GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error)

	// return snapshot hash
	GetSnapshotHashByHeight(height uint64) (*types.Hash, error)

	// block contains header、snapshot content
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSnapshotHeaderByHash(hash types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error)

	// contains start snapshot block and end snapshot block
	GetRangeSnapshotHeaders(startHash types.Hash, endHash types.Hash) ([]*ledger.SnapshotBlock, error)

	GetRangeSnapshotBlocks(startHash types.Hash, endHash types.Hash) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot header that has the blockHash
	GetSnapshotHeaders(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot block that has the blockHash
	GetSnapshotBlocks(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot block that has the blockHash
	GetSnapshotHeadersByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot block that has the blockHash
	GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	GetConfirmSnapshotHeaderByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error)

	GetConfirmSnapshotBlockByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)

	GetLastUnpublishedSeedSnapshotHeader(producer types.Address, beforeTime time.Time) (*ledger.SnapshotBlock, error)

	GetRandomSeed(snapshotHash types.Hash, n int) uint64

	GetSnapshotBlockByContractMeta(addr types.Address, fromHash types.Hash) (*ledger.SnapshotBlock, error)

	GetSeedConfirmedSnapshotBlock(addr types.Address, fromHash types.Hash) (*ledger.SnapshotBlock, error)

	GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error)

	GetSubLedger(startHeight, endHeight uint64) ([]*ledger.SnapshotChunk, error)

	GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error)

	// ====== Query unconfirmed pool ======
	GetAllUnconfirmedBlocks() []*ledger.AccountBlock

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	//Deprecated:
	GetContentNeedSnapshot() ledger.SnapshotContent

	GetContentNeedSnapshotRange() map[types.Address]*ledger.HeightRange

	// ====== Query account ======

	IterateContracts(iterateFunc func(addr types.Address, meta *ledger.ContractMeta, err error) bool)

	IterateAccounts(iterateFunc func(addr types.Address, accountId uint64, err error) bool)

	// ===== Query state ======
	// get Balance
	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error)

	// get Balance map
	GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error)

	// get confirmed snapshot Balance, if history is too old, failed
	GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error)

	// get contract code
	GetContractCode(contractAddr types.Address) ([]byte, error)

	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)

	GetContractMetaInSnapshot(contractAddress types.Address, snapshotHeight uint64) (meta *ledger.ContractMeta, err error)

	GetContractList(gid types.Gid) ([]types.Address, error)

	GetQuotaUnused(address types.Address) (uint64, error)

	GetGlobalQuota() types.QuotaInfo

	GetQuotaUsedList(address types.Address) []types.QuotaInfo

	GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(address types.Address, key []byte) ([]byte, error)

	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)

	GetVMLogListByAddress(address types.Address, start uint64, end uint64, id *types.Hash) (ledger.VmLogList, error)

	// ====== Query built-in contract storage ======

	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)

	GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)

	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)

	GetConsensusGroup(snapshotHash types.Hash, gid types.Gid) (*types.ConsensusGroupInfo, error)

	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)

	GetStakeBeneficialAmount(addr types.Address) (*big.Int, error)

	// total
	GetStakeQuota(addr types.Address) (*big.Int, *types.Quota, error)

	// total
	GetStakeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error)

	GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error)

	GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error)

	CalVoteDetails(gid types.Gid, info *core.GroupInfo, snapshotBlock ledger.HashHeight) ([]*interfaces.VoteDetails, error)

	GetStakeListByPage(snapshotHash types.Hash, lastKey []byte, count uint64) ([]*types.StakeInfo, []byte, error)

	GetDexFundsByPage(snapshotHash types.Hash, lastAddress types.Address, count int) ([]*dex.Fund, error)

	GetDexStakeListByPage(snapshotHash types.Hash, lastKey []byte, count int) ([]*dex.DelegateStakeInfo, []byte, error)

	// ====== Sync ledger ======
	GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error)

	GetSyncCache() interfaces.SyncCache

	// ====== OnRoad ======
	LoadOnRoadRange(gid types.Gid, fn interfaces.LoadOnroadFn) error

	DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash)

	GetOnRoadBlocksByAddr(addr types.Address, pageNum, pageSize int) ([]*ledger.AccountBlock, error)

	LoadAllOnRoad() (map[types.Address][]types.Hash, error)

	GetAccountOnRoadInfo(addr types.Address) (*ledger.AccountInfo, error)

	GetOnRoadInfoUnconfirmedHashList(addr types.Address) ([]*types.Hash, error)

	UpdateOnRoadInfo(addr types.Address, tkId types.TokenTypeId, number uint64, amount big.Int) error

	ClearOnRoadUnconfirmedCache(addr types.Address, hashList []*types.Hash) error

	// ====== Other ======
	SetCacheLevelForConsensus(level uint32) // affect `GetVoteList` and `GetConfirmedBalanceList`. 0 means no cache, 1 means cache

	NewDb(dirName string) (*leveldb.DB, error)

	PrepareOnroadDb() (*leveldb.DB, error)

	Plugins() *chain_plugins.Plugins

	SetConsensus(verifier interfaces.ConsensusVerifier, periodTimeIndex interfaces.TimeIndex)

	DBs() (*chain_index.IndexDB, *chain_block.BlockDB, *chain_state.StateDB)

	Flusher() *chain_flusher.Flusher

	StopWrite()

	RecoverWrite()

	WriteGenesisCheckSum(hash types.Hash) error

	QueryGenesisCheckSum() (*types.Hash, error)

	// ====== Check ======
	CheckRedo() error

	CheckRecentBlocks() error

	CheckOnRoad() error

	GetStatus() []interfaces.DBStatus
}
