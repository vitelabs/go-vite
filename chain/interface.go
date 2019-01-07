package chain

import (
	"math/big"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/chain/sender"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
)

type InsertProcessorFunc func(batch *leveldb.Batch, blocks []*vm_context.VmAccountBlock) error
type InsertProcessorFuncSuccess func(blocks []*vm_context.VmAccountBlock)
type DeleteProcessorFunc func(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error
type DeleteProcessorFuncSuccess func(subLedger map[types.Address][]*ledger.AccountBlock)

type InsertSnapshotBlocksSuccess func([]*ledger.SnapshotBlock)
type DeleteSnapshotBlocksSuccess func([]*ledger.SnapshotBlock)

type Chain interface {
	InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error
	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start uint64, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlockMap(queryParams map[types.Address]*BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error)
	GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error)
	GetAccountBlockHashByHeight(addr *types.Address, height uint64) (*types.Hash, error)

	GetAllLatestAccountBlock() ([]*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlocksByAddress(addr *types.Address, index int, num int, count int) ([]*ledger.AccountBlock, error)
	GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error)

	GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock
	GetUnConfirmedSubLedger() (map[types.Address][]*ledger.AccountBlock, error)
	GetUnConfirmedPartSubLedger(addrList []types.Address) (map[types.Address][]*ledger.AccountBlock, error)

	DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)
	Init()
	Compressor() *compress.Compressor
	TrieGc() trie_gc.Collector

	StopSaveTrie()
	StartSaveTrie()

	ChainDb() *chain_db.ChainDb
	SaList() *chain_cache.AdditionList

	Start()
	Destroy()
	Stop()
	GenStateTrie(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error)
	GetNeedSnapshotContent() ledger.SnapshotContent

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error
	GetAccountBlockMetaByHash(hash *types.Hash) (*ledger.AccountBlockMeta, error)
	GetSnapshotBlocksByHash(originBlockHash *types.Hash, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockHeadByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockHeadByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetConfirmBlock(accountBlockHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetConfirmTimes(accountBlockHash *types.Hash) (uint64, error)
	GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error)
	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)
	KafkaSender() *sender.KafkaSender

	// Pledge amount
	GetPledgeAmount(snapshotHash types.Hash, beneficial types.Address) (*big.Int, error)

	// Pledge quota
	GetPledgeQuota(snapshotHash types.Hash, beneficial types.Address) (uint64, error)
	GetPledgeQuotas(snapshotHash types.Hash, beneficialList []types.Address) (map[types.Address]uint64, error)

	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error)

	GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error)
	AccountType(address *types.Address) (uint64, error)
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetSubLedgerByHeight(startHeight uint64, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64)
	GetSubLedgerByHash(startBlockHash *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error)
	GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)
	UnRegister(listenerId uint64)
	TrieDb() *leveldb.DB
	CleanTrieNodePool()
	RegisterInsertAccountBlocks(processor InsertProcessorFunc) uint64
	RegisterInsertAccountBlocksSuccess(processor InsertProcessorFuncSuccess) uint64
	RegisterDeleteAccountBlocks(processor DeleteProcessorFunc) uint64
	RegisterDeleteAccountBlocksSuccess(processor DeleteProcessorFuncSuccess) uint64
	RegisterInsertSnapshotBlocksSuccess(processor InsertSnapshotBlocksSuccess) uint64
	RegisterDeleteSnapshotBlocksSuccess(processor DeleteSnapshotBlocksSuccess) uint64
	GetConfirmSubLedgerBySnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error)

	GetStateTrie(stateHash *types.Hash) *trie.Trie
	NewStateTrie() *trie.Trie

	IsGenesisSnapshotBlock(block *ledger.SnapshotBlock) bool

	// Be
	GetLatestBlockEventId() (uint64, error)
	GetEvent(eventId uint64) (byte, []types.Hash, error)

	// onroad
	IsSuccessReceived(addr *types.Address, hash *types.Hash) bool

	getChainRangeSet(snapshotBlocks []*ledger.SnapshotBlock) map[types.Address][2]*ledger.HashHeight

	// account block is existed
	IsAccountBlockExisted(hash types.Hash) (bool, error)

	// get receive block heights
	GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error)
}
