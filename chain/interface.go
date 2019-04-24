package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"time"
)

type EventListener interface {
	PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error
	InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error

	PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error

	PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error
	DeleteAccountBlocks(blocks []*ledger.AccountBlock) error

	PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
}

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
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
	Register(listener EventListener)
	UnRegister(listener EventListener)

	/*
	 *	C(Create)
	 */

	// vmAccountBlocks must have the same address
	InsertAccountBlock(vmAccountBlocks *vm_db.VmAccountBlock) error

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

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	// query receive block of send block
	GetReceiveAbBySendAb(sendBlockHash types.Hash) (*ledger.AccountBlock, error)

	// is received
	IsReceived(sendBlockHash types.Hash) (bool, error)

	// high to low, contains the block that has the blockHash
	GetAccountBlocks(blockHash types.Hash, count uint64) ([]*ledger.AccountBlock, error)

	GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	GetAccountBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*ledger.AccountBlock, error)

	// get call depth

	GetCallDepth(sendBlock types.Hash) (uint16, error)

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

	GetLastSeedSnapshotHeader(producer types.Address) (*ledger.SnapshotBlock, error)

	GetRandomSeed(snapshotHash types.Hash, n int) uint64

	GetSnapshotBlockByContractMeta(addr *types.Address, fromHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error)

	GetSubLedger(startHeight, endHeight uint64) ([]*ledger.SnapshotChunk, error)

	GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error)

	// ====== Query unconfirmed pool ======
	GetAllUnconfirmedBlocks() []*ledger.AccountBlock

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetContentNeedSnapshot() ledger.SnapshotContent

	// ====== Query account ======

	// The address is contract address when it's first receive block inserted into the chain.
	// In others words, The first receive block of the address is not contract address when the block has not yet been inserted into the chain
	IsContractAccount(address types.Address) (bool, error)

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

	GetContractList(gid types.Gid) ([]types.Address, error)

	GetQuotaUnused(address types.Address) (uint64, error)

	GetQuotaUsed(address types.Address) (quotaUsed uint64, blockCount uint64)

	GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(address types.Address, key []byte) ([]byte, error)

	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)

	// ====== Query built-in contract storage ======

	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)

	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)

	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)

	GetPledgeBeneficialAmount(addr types.Address) (*big.Int, error)

	// total
	GetPledgeQuota(addr types.Address) (*types.Quota, error)

	// total
	GetPledgeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error)

	GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error)

	GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error)

	// ====== Sync ledger ======
	GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error)

	GetSyncCache() interfaces.SyncCache

	// ====== OnRoad ======
	LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error)

	DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash)

	GetAccountOnRoadInfo(addr types.Address) (*ledger.AccountInfo, error)

	GetOnRoadBlocksByAddr(addr types.Address, pageNum, pageSize int) ([]*ledger.AccountBlock, error)

	// ====== Other ======
	NewDb(dirName string) (*leveldb.DB, error)

	Plugins() *chain_plugins.Plugins

	SetConsensus(cs Consensus)
}
