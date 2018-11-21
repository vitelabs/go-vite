package chain_benchmark

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type mockVmDatabse struct{}

func (context *mockVmDatabse) CopyAndFreeze() vmctxt_interface.VmDatabase {
	return nil
}

func (context *mockVmDatabse) Address() *types.Address {
	return nil
}

func (context *mockVmDatabse) CurrentSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (context *mockVmDatabse) PrevAccountBlock() *ledger.AccountBlock {
	return nil
}

func (context *mockVmDatabse) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func (context *mockVmDatabse) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	return nil
}

func (context *mockVmDatabse) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *mockVmDatabse) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
}

func (context *mockVmDatabse) GetSnapshotBlocks(startHeight, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	return nil
}

func (context *mockVmDatabse) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *mockVmDatabse) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (context *mockVmDatabse) Reset() {}

func (context *mockVmDatabse) SetContractGid(gid *types.Gid, addr *types.Address) {}

func (context *mockVmDatabse) SetContractCode(code []byte) {}

func (context *mockVmDatabse) GetContractCode(addr *types.Address) []byte {
	return nil
}

func (context *mockVmDatabse) SetStorage(key []byte, value []byte) {}

func (context *mockVmDatabse) GetStorage(addr *types.Address, key []byte) []byte {
	return nil
}

func (context *mockVmDatabse) GetStorageHash() *types.Hash {
	return nil
}

func (context *mockVmDatabse) GetGid() *types.Gid {
	return nil
}

func (context *mockVmDatabse) AddLog(log *ledger.VmLog) {
}

func (context *mockVmDatabse) GetLogListHash() *types.Hash {
	return nil
}

func (context *mockVmDatabse) IsAddressExisted(addr *types.Address) bool {
	return false
}

func (context *mockVmDatabse) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	return nil
}

func (context *mockVmDatabse) NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator {
	return nil
}

func (context *mockVmDatabse) getBalanceBySnapshotHash(addr *types.Address, tokenTypeId *types.TokenTypeId, snapshotHash *types.Hash) *big.Int {
	return nil
}
func (context *mockVmDatabse) GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte {
	return nil
}
func (context *mockVmDatabse) NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator {
	return nil
}

func (context *mockVmDatabse) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	return nil, nil
}
func (context *mockVmDatabse) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	return nil, nil
}
func (context *mockVmDatabse) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	return nil, nil
}
func (context *mockVmDatabse) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) {
	return nil, nil
}
func (context *mockVmDatabse) GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (context *mockVmDatabse) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
