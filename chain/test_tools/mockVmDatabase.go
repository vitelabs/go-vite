package test_tools

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type MockVmDatabase struct{}

func (context *MockVmDatabase) CopyAndFreeze() vmctxt_interface.VmDatabase {
	return nil
}

func (context *MockVmDatabase) Address() *types.Address {
	return nil
}

func (context *MockVmDatabase) CurrentSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (context *MockVmDatabase) PrevAccountBlock() *ledger.AccountBlock {
	return nil
}

func (context *MockVmDatabase) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func (context *MockVmDatabase) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	return nil
}

func (context *MockVmDatabase) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *MockVmDatabase) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *MockVmDatabase) GetSnapshotBlocks(startHeight, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	return nil
}

func (context *MockVmDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *MockVmDatabase) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (context *MockVmDatabase) Reset() {}

func (context *MockVmDatabase) SetContractGid(gid *types.Gid, addr *types.Address) {}

func (context *MockVmDatabase) SetContractCode(code []byte) {}

func (context *MockVmDatabase) GetContractCode(addr *types.Address) []byte {
	return nil
}

func (context *MockVmDatabase) SetStorage(key []byte, value []byte) {}

func (context *MockVmDatabase) GetStorage(addr *types.Address, key []byte) []byte {
	return nil
}

func (context *MockVmDatabase) GetStorageHash() *types.Hash {
	return nil
}

func (context *MockVmDatabase) GetGid() *types.Gid {
	return nil
}

func (context *MockVmDatabase) AddLog(log *ledger.VmLog) {
}

func (context *MockVmDatabase) GetLogListHash() *types.Hash {
	return nil
}

func (context *MockVmDatabase) IsAddressExisted(addr *types.Address) bool {
	return false
}

func (context *MockVmDatabase) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	return nil
}

func (context *MockVmDatabase) NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator {
	return nil
}

func (context *MockVmDatabase) getBalanceBySnapshotHash(addr *types.Address, tokenTypeId *types.TokenTypeId, snapshotHash *types.Hash) *big.Int {
	return nil
}
func (context *MockVmDatabase) GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte {
	return nil
}
func (context *MockVmDatabase) NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator {
	return nil
}

func (context *MockVmDatabase) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	return nil, nil
}
func (context *MockVmDatabase) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	return nil, nil
}
func (context *MockVmDatabase) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	return nil, nil
}
func (context *MockVmDatabase) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) {
	return nil, nil
}
func (context *MockVmDatabase) GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *MockVmDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

func (context *MockVmDatabase) DebugGetStorage() map[string][]byte {
	return nil
}

func (context *MockVmDatabase) GetOneHourQuota() uint64 {
	return 0
}

func (context *MockVmDatabase) GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error) {
	return nil, nil
}
