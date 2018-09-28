package chain

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"time"
)

type Chain interface {
	InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error
	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start uint64, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlockMap(queryParams map[types.Address]*BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error)
	GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error)
	GetAccountBlockHashByHeight(addr *types.Address, height uint64) (*types.Hash, error)

	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlocksByAddress(addr *types.Address, index int, num int, count int) ([]*ledger.AccountBlock, error)
	GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error)
	GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock
	DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)
	Init()
	Compressor() *compress.Compressor
	ChainDb() *chain_db.ChainDb
	Start()
	Stop()
	GenStateTrie(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) *trie.Trie
	GetNeedSnapshotContent() ledger.SnapshotContent
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error
	GetSnapshotBlocksByHash(originBlockHash *types.Hash, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetConfirmBlock(accountBlockHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetConfirmTimes(accountBlockHash *types.Hash) (uint64, error)
	GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error)
	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo
	GetPledgeAmount(snapshotHash types.Hash, beneficial types.Address) *big.Int
	GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) map[types.Address]*big.Int
	GetTokenInfoById(tokenId *types.TokenTypeId) *contracts.TokenInfo
	AccountType(address *types.Address) (uint64, error)
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetSubLedgerByHeight(startHeight uint64, count uint64, forward bool) ([]string, [][2]uint64)
	GetSubLedgerByHash(startBlockHash *types.Hash, count uint64, forward bool) ([]string, [][2]uint64, error)
	GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error)
	UnRegister(listenerId uint64)
	RegisterInsertAccountBlocks(processor insertProcessorFunc) uint64
	RegisterInsertAccountBlocksSuccess(processor insertProcessorFuncSuccess) uint64
	RegisterDeleteAccountBlocks(processor deleteProcessorFunc) uint64
	RegisterDeleteAccountBlocksSuccess(processor deleteProcessorFuncSuccess) uint64
	GetStateTrie(stateHash *types.Hash) *trie.Trie
	NewStateTrie() *trie.Trie
	GetPledgeQuota(snapshotHash types.Hash, beneficial types.Address) uint64
}
