package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/state"
	"github.com/vitelabs/go-vite/vm_context"
	"io"
	"math/big"
	"time"
)

type PrepareInsertAccountBlocksListener func(blocks []*vm_context.VmAccountBlock) error
type InsertAccountBlocksListener func(blocks []*vm_context.VmAccountBlock) error

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
	InsertAccountBlock(vmAccountBlocks *vm_context.VmAccountBlock) error

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (invalidSubLedger map[types.Address][]*ledger.AccountBlock, err error)

	/*
	 *	D(Delete)
	 */

	// contain the snapshot block of toHash, delete all blocks higher than snapshot line
	DeleteSnapshotBlocks(toHash *types.Hash) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	// contain the snapshot block of toHeight`, delete all blocks higher than snapshot line
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	/*
	 *	R(Retrieve)
	 */

	// ====== Query account block ======
	IsAccountBlockExisted(hash *types.Hash) (bool, error)

	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	// query receive block of send block
	GetReceiveAbBySendAb(sendBlockHash *types.Hash) (*ledger.AccountBlock, error)

	// is received
	IsReceived(sendBlockHash *types.Hash) (bool, error)

	// high to low, contains the block that has the blockHash
	GetAccountBlocks(blockHash *types.Hash, count uint64) ([]*ledger.AccountBlock, error)

	// get call depth
	GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error)

	// get confirmed times
	GetConfirmedTimes(blockHash *types.Hash) (uint64, error)

	// ====== Query snapshot block ======

	IsSnapshotBlockExisted(hash *types.Hash) (bool, error)

	// is valid
	IsSnapshotContentValid(snapshotContent *ledger.SnapshotContent) (invalidMap map[types.Address]*ledger.HashHeight, err error)

	GetGenesisSnapshotHeader() *ledger.SnapshotBlock

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	GetLatestSnapshotHeader() *ledger.SnapshotBlock

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	// header without snapshot content
	GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error)

	// block contains header、snapshot content
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSnapshotHeaderByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	// contains start snapshot block and end snapshot block
	GetRangeSnapshotHeaders(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error)

	GetRangeSnapshotBlocks(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot header that has the blockHash
	GetSnapshotHeaders(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	// contains the snapshot block that has the blockHash
	GetSnapshotBlocks(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	GetConfirmSnapshotHeaderByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error)

	GetConfirmSnapshotBlockByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)

	GetSnapshotContentBySbHash(hash *types.Hash) (ledger.SnapshotContent, error)

	// ====== Query unconfirmed pool ======
	GetUnconfirmedBlocks(addr *types.Address) ([]*ledger.AccountBlock, error)

	GetContentNeedSnapshot() ledger.SnapshotContent

	// ====== Query account ======

	// AccountType(address *types.Address) (byte, error)

	// The address is contract address when it's first receive block inserted into the chain.
	// In others words, The first receive block of the address is not contract address when the block has not yet been inserted into the chain
	IsContractAccount(address *types.Address) (bool, error)

	// ===== Query state ======
	// get balance
	GetBalance(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error)

	// get history balance, if history is too old, failed
	GetHistoryBalance(addr *types.Address, tokenId *types.TokenTypeId, accountBlockHash *types.Hash) (*big.Int, error)

	// get confirmed snapshot balance, if history is too old, failed
	GetConfirmedBalance(addr *types.Address, tokenId *types.TokenTypeId, sbHash *types.Hash) (*big.Int, error)

	// get contract code
	GetContractCode(contractAddr *types.Address) ([]byte, error)

	GetContractMeta(contractAddress *types.Address) (meta *ledger.ContractMeta, err error)

	GetContractList(gid *types.Gid) (map[types.Address]*ledger.ContractMeta, error)

	GetStateSnapshot(blockHash *types.Hash) (stateSnapshot chain_state.StateSnapshot, err error)
	// ====== Query built-in contract storage ======

	GetRegisterList(snapshotHash *types.Hash, gid *types.Gid) ([]*types.Registration, error)

	GetVoteMap(snapshotHash *types.Hash, gid *types.Gid) ([]*types.VoteInfo, error)

	GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error)

	// total
	GetPledgeQuota(snapshotHash *types.Hash, addr *types.Address) (uint64, error)

	// total
	GetPledgeQuotas(snapshotHash *types.Hash, addrList []*types.Address) (map[types.Address]uint64, error)

	GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error)

	GetAllTokenInfo() ([]*types.TokenInfo, error)

	// ====== Query used quota ======

	GetQuotaUnused(address *types.Address) uint64

	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)

	// ====== Query vm log list ======
	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)

	// ====== Sync ledger ======
	GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr LedgerReader, err error)

	// TODO insert syncCache ledger
	// TODO query syncCache state

	// ====== OnRoad ======
	HasOnRoadBlocks(address *types.Address) (bool, error)

	GetOnRoadBlocksHashList(address *types.Address, pageNum, countPerPage int) ([]*types.Hash, error)
}

type LedgerReader interface {
	Bound() (from, to uint64)
	Size() int
	Stream() io.ReadCloser
}

type SyncCache interface {
	NewWriter(from, to uint64) io.WriteCloser
	Chunks() [][2]uint64
	NewReader() (interface {
		Read(from, to uint64, fn func(ablock *ledger.AccountBlock, sblock *ledger.SnapshotBlock, err error))
		Close() error
	}, error)
}
