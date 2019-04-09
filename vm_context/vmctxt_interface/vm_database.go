package vmctxt_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"time"
)

type VmDatabase interface {
	GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int
	AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock
	GetOneHourQuota() (uint64, error)

	// forward=true return [startHeight, startHeight+count), forward=false return (startHeight-count, startHeight]
	GetSnapshotBlocks(startHeight uint64, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock

	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	GetSelfAccountBlockByHeight(height uint64) *ledger.AccountBlock

	UnsavedCache() UnsavedCache
	Reset()

	IsAddressExisted(addr *types.Address) bool
	SetContractGid(gid *types.Gid, addr *types.Address)
	SetContractCode(code []byte)
	GetContractCode(addr *types.Address) []byte

	GetStorage(addr *types.Address, key []byte) []byte
	GetOriginalStorage(key []byte) []byte
	SetStorage(key []byte, value []byte)
	GetStorageHash() *types.Hash

	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash
	GetLogList() ledger.VmLogList

	NewStorageIterator(addr *types.Address, prefix []byte) StorageIterator

	CopyAndFreeze() VmDatabase

	GetGid() *types.Gid
	Address() *types.Address
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock

	GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte
	NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) StorageIterator
	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error)
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	DebugGetStorage() map[string][]byte

	GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error)
}
