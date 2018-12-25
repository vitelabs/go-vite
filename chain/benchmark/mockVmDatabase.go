package chain_benchmark

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type mockVmDatabase struct{}

func (context *mockVmDatabase) CopyAndFreeze() vmctxt_interface.VmDatabase {
	return nil
}

func (context *mockVmDatabase) Address() *types.Address {
	return nil
}

func (context *mockVmDatabase) CurrentSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (context *mockVmDatabase) PrevAccountBlock() *ledger.AccountBlock {
	return nil
}

func (context *mockVmDatabase) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func (context *mockVmDatabase) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	return nil
}

func (context *mockVmDatabase) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *mockVmDatabase) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *mockVmDatabase) GetSnapshotBlocks(startHeight, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	return nil
}

func (context *mockVmDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *mockVmDatabase) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (context *mockVmDatabase) Reset() {}

func (context *mockVmDatabase) SetContractGid(gid *types.Gid, addr *types.Address) {}

func (context *mockVmDatabase) SetContractCode(code []byte) {}

func (context *mockVmDatabase) GetContractCode(addr *types.Address) []byte {
	return nil
}

func (context *mockVmDatabase) SetStorage(key []byte, value []byte) {}

func (context *mockVmDatabase) GetStorage(addr *types.Address, key []byte) []byte {
	return nil
}

func (context *mockVmDatabase) GetStorageHash() *types.Hash {
	return nil
}

func (context *mockVmDatabase) GetGid() *types.Gid {
	return nil
}

func (context *mockVmDatabase) AddLog(log *ledger.VmLog) {
}

func (context *mockVmDatabase) GetLogListHash() *types.Hash {
	return nil
}

func (context *mockVmDatabase) IsAddressExisted(addr *types.Address) bool {
	return false
}

func (context *mockVmDatabase) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	return nil
}

func (context *mockVmDatabase) NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator {
	return nil
}

func (context *mockVmDatabase) getBalanceBySnapshotHash(addr *types.Address, tokenTypeId *types.TokenTypeId, snapshotHash *types.Hash) *big.Int {
	return nil
}
func (context *mockVmDatabase) GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte {
	return nil
}
func (context *mockVmDatabase) NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator {
	return nil
}

func (context *mockVmDatabase) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	return nil, nil
}
func (context *mockVmDatabase) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	return nil, nil
}
func (context *mockVmDatabase) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	return nil, nil
}
func (context *mockVmDatabase) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) {
	return nil, nil
}
func (context *mockVmDatabase) GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *mockVmDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

func (context *mockVmDatabase) DebugGetStorage() map[string][]byte {
	return nil
}
