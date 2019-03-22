package vmctxt_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"time"
)

type VmDatabase interface {
	GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int // remove addr
	AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) // remove
	GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock         // remove
	GetOneHourQuota() (uint64, error)                                      // remove

	// forward=true return [startHeight, startHeight+count), forward=false return (startHeight-count, startHeight]
	GetSnapshotBlocks(startHeight uint64, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock // remove

	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	GetSelfAccountBlockByHeight(height uint64) *ledger.AccountBlock // remove

	UnsavedCache() UnsavedCache // remove
	Reset()                     // remove

	IsAddressExisted(addr *types.Address) bool          // remove
	SetContractGid(gid *types.Gid, addr *types.Address) // modify
	SetContractCode(code []byte)                        // modify
	GetContractCode(addr *types.Address) []byte         // modify

	GetStorage(addr *types.Address, key []byte) []byte // modify
	GetOriginalStorage(key []byte) []byte              // ?
	SetStorage(key []byte, value []byte)               // modify
	GetStorageHash() *types.Hash                       // modify

	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash

	NewStorageIterator(addr *types.Address, prefix []byte) StorageIterator // remove addr

	CopyAndFreeze() VmDatabase // remove

	GetGid() *types.Gid                          // modify
	Address() *types.Address                     // modify
	CurrentSnapshotBlock() *ledger.SnapshotBlock // modify
	PrevAccountBlock() *ledger.AccountBlock      // modify

	GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte                     // remove
	NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) StorageIterator // remove

	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                     // ?
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                                  // ?
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                           // ?
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) // ?
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)                                                         // ?

	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	DebugGetStorage() map[string][]byte

	GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error)
}
