package vm

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type mockDB struct {
	currentAddr            *types.Address
	latestSnapshotBlock    *ledger.SnapshotBlock
	prevAccountBlock       *ledger.AccountBlock
	quotaInfo              []types.QuotaInfo
	pledgeBeneficialAmount *big.Int
	balanceMap             map[types.TokenTypeId]*big.Int
	contractMetaMap        map[types.Address]*ledger.ContractMeta
}

func NewMockDB(addr *types.Address,
	latestSnapshotBlock *ledger.SnapshotBlock,
	prevAccountBlock *ledger.AccountBlock,
	quotaInfo []types.QuotaInfo,
	pledgeBeneficialAmount *big.Int,
	balanceMap map[types.TokenTypeId]*big.Int) *mockDB {

	db := &mockDB{currentAddr: addr,
		latestSnapshotBlock:    latestSnapshotBlock,
		prevAccountBlock:       prevAccountBlock,
		quotaInfo:              quotaInfo,
		pledgeBeneficialAmount: new(big.Int).Set(pledgeBeneficialAmount),
		contractMetaMap:        make(map[types.Address]*ledger.ContractMeta),
	}
	balanceMapCopy := make(map[types.TokenTypeId]*big.Int)
	for tid, amount := range balanceMap {
		balanceMapCopy[tid] = new(big.Int).Set(amount)
	}
	db.balanceMap = balanceMapCopy
	return db
}

func (db *mockDB) Address() *types.Address {
	return db.currentAddr
}
func (db *mockDB) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	if b := db.latestSnapshotBlock; b == nil {
		return nil, errors.New("latest snapshot block not exist")
	} else {
		return b, nil
	}
}
func (db *mockDB) PrevAccountBlock() (*ledger.AccountBlock, error) {
	if b := db.prevAccountBlock; b == nil {
		return nil, errors.New("prev account block not exist")
	} else {
		return b, nil
	}
}
func (db *mockDB) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if addr != *db.currentAddr {
		return nil, errors.New("current account address not match")
	} else {
		return db.prevAccountBlock, nil
	}
}
func (db *mockDB) IsContractAccount() (bool, error) {
	return types.IsContractAddr(*db.currentAddr), nil
}
func (db *mockDB) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	return 0, nil
}
func (db *mockDB) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	if addr != *db.currentAddr {
		return nil
	} else {
		return db.quotaInfo
	}
}
func (db *mockDB) GetGlobalQuota() types.QuotaInfo {
	return types.QuotaInfo{}
}
func (db *mockDB) GetReceiptHash() *types.Hash {
	return nil
}
func (db *mockDB) Reset() {

}
func (db *mockDB) Finish() {

}
func (db *mockDB) GetValue(key []byte) ([]byte, error) {
	return nil, nil
}
func (db *mockDB) GetOriginalValue(key []byte) ([]byte, error) {
	return nil, nil
}
func (db *mockDB) SetValue(key []byte, value []byte) error {
	return nil
}
func (db *mockDB) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	return nil, nil
}
func (db *mockDB) GetUnsavedStorage() [][2][]byte {
	return nil
}
func (db *mockDB) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.balanceMap[*tokenTypeId]; !ok {
		return big.NewInt(0), nil
	} else {
		return balance, nil
	}
}
func (db *mockDB) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	db.balanceMap[*tokenTypeId] = amount
}
func (db *mockDB) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}
func (db *mockDB) AddLog(log *ledger.VmLog) {

}
func (db *mockDB) GetLogList() ledger.VmLogList {
	return nil
}
func (db *mockDB) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}
func (db *mockDB) GetLogListHash() *types.Hash {
	return nil
}
func (db *mockDB) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return nil
}
func (db *mockDB) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (db *mockDB) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}
func (db *mockDB) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return 0, nil
}
func (db *mockDB) SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta) {
	db.contractMetaMap[toAddr] = meta
}
func (db *mockDB) GetContractMeta() (*ledger.ContractMeta, error) {
	return db.contractMetaMap[*db.currentAddr], nil
}
func (db *mockDB) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error) {
	return db.contractMetaMap[contractAddress], nil
}
func (db *mockDB) SetContractCode(code []byte) {

}
func (db *mockDB) GetContractCode() ([]byte, error) {
	return nil, nil
}
func (db *mockDB) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}
func (db *mockDB) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return nil
}
func (db *mockDB) GetUnsavedContractCode() []byte {
	return nil
}
func (db *mockDB) GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	if *addr != *db.currentAddr {
		return nil, errors.New("current account address not match")
	} else {
		return db.pledgeBeneficialAmount, nil
	}
}
func (db *mockDB) DebugGetStorage() (map[string][]byte, error) {
	return nil, nil
}
func (db *mockDB) CanWrite() bool {
	return false
}
