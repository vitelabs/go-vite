package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"time"
)

type PrepareInsertAccountBlocksListener func(blocks []*vm_db.VmAccountBlock) error
type InsertAccountBlocksListener func(blocks []*vm_db.VmAccountBlock) error

type PrepareInsertSnapshotBlocksListener func(snapshotBlock []*ledger.SnapshotBlock) error
type InsertSnapshotBlocksListener func(snapshotBlock []*ledger.SnapshotBlock) error

type PrepareDeleteAccountBlocksListener func(subLedger map[types.Address][]*ledger.AccountBlock) error
type DeleteAccountBlocksListener func(subLedger map[types.Address][]*ledger.AccountBlock) error

type PrepareDeleteSnapshotBlocksListener func(snapshotBlockList []*ledger.SnapshotBlock, subLedger map[types.Address][]*ledger.AccountBlock) error
type DeleteSnapshotBlocksListener func(snapshotBlockList []*ledger.SnapshotBlock, subLedger map[types.Address][]*ledger.AccountBlock) error

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
	RegisterPrepareInsertAccountBlocks(listener PrepareInsertAccountBlocksListener) (eventHandler uint64)
	RegisterInsertAccountBlocks(listener InsertAccountBlocksListener) (eventHandler uint64)

	RegisterPrepareInsertSnapshotBlocks(listener PrepareInsertSnapshotBlocksListener) (eventHandler uint64)
	RegisterInsertSnapshotBlocks(listener InsertSnapshotBlocksListener) (eventHandler uint64)

	RegisterPrepareDeleteAccountBlocks(listener PrepareDeleteAccountBlocksListener) (eventHandler uint64)
	RegisterDeleteAccountBlocks(listener DeleteAccountBlocksListener) (eventHandler uint64)

	RegisterPrepareDeleteSnapshotBlocks(listener PrepareDeleteSnapshotBlocksListener) (eventHandler uint64)
	RegisterDeleteSnapshotBlocks(listener DeleteSnapshotBlocksListener) (eventHandler uint64)

	UnRegister(eventHandler uint64)

	/*
	 *	C(Create)
	 */

	// vmAccountBlocks must have the same address
	InsertAccountBlock(vmAccountBlocks *vm_db.VmAccountBlock) error

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error)

	/*
	 *	D(Delete)
	 */

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

	// is valid
	IsSnapshotContentValid(snapshotContent ledger.SnapshotContent) (invalidMap map[types.Address]*ledger.HashHeight, err error)

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

	GetConfirmSnapshotHeaderByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error)

	GetConfirmSnapshotBlockByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)

	GetLastSeedSnapshotHeader(producer types.Address) (*ledger.SnapshotBlock, error)

	GetRandomSeed(snapshotHash types.Hash, n int) uint64

	GetSubLedger(endHeight, startHeight uint64) ([]*chain_block.SnapshotSegment, error)

	GetSubLedgerAfterHeight(height uint64) ([]*chain_block.SnapshotSegment, error)

	// ====== Query unconfirmed pool ======
	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetContentNeedSnapshot() ledger.SnapshotContent

	// ====== Query account ======

	// The address is contract address when it's first receive block inserted into the chain.
	// In others words, The first receive block of the address is not contract address when the block has not yet been inserted into the chain
	IsContractAccount(address types.Address) (bool, error)

	// ===== Query state ======
	// get balance
	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error)

	// get balance map
	GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error)

	// get confirmed snapshot balance, if history is too old, failed
	GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error)

	// get contract code
	GetContractCode(contractAddr types.Address) ([]byte, error)

	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)

	GetContractList(gid types.Gid) ([]types.Address, error)

	GetQuotaUnused(address types.Address) (uint64, error)

	GetQuotaUsed(address types.Address) (quotaUsed uint64, blockCount uint64)

	GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(address types.Address, key []byte) ([]byte, error)

	// ====== Query built-in contract storage ======

	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)

	GetConsensusGroupList(snapshotHash types.Hash, addr types.Address) ([]types.Gid, error)

	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)

	GetPledgeAmount(addr types.Address) (*big.Int, error)

	// total
	GetPledgeQuota(addr types.Address) (*types.Quota, error)

	// total
	GetPledgeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error)

	GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error)

	GetAllTokenInfo() ([]*types.TokenInfo, error)

	// ====== Query vm log list ======`
	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)

	// ====== Sync ledger ======
	GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error)

	// TODO insert syncCache ledger
	// TODO query syncCache state

	// ====== OnRoad ======
	HasOnRoadBlocks(address types.Address) (bool, error)

	GetOnRoadBlocksHashList(address types.Address, pageNum, countPerPage int) ([]types.Hash, error)

	// ====== Other ======
	NewDb(dirName string) (*leveldb.DB, error)
	GetRandomGlobalStatus(addr *types.Address, fromHash *types.Hash) (*util.GlobalStatus, error)
}
