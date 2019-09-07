package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

var (
	balanceKey = "$BALANCE"
	codeKey    = "$CODE"
)

func getBalanceKey(tokenID *types.TokenTypeId) string {
	return balanceKey + tokenID.String()
}

func getCodeKey(addr types.Address) string {
	return codeKey + addr.String()
}

// test database for single call
type memoryDatabase struct {
	addr            types.Address
	storage         map[string][]byte
	originalStorage map[string][]byte
	logList         []*ledger.VmLog
	sb              *ledger.SnapshotBlock
}

func newMemoryDatabase(addr types.Address, sb *ledger.SnapshotBlock) *memoryDatabase {
	return &memoryDatabase{
		addr:            addr,
		storage:         make(map[string][]byte),
		originalStorage: make(map[string][]byte),
		logList:         make([]*ledger.VmLog, 0),
		sb:              sb,
	}
}
func (db *memoryDatabase) GetBalance(tokenTypeID *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.storage[getBalanceKey(tokenTypeID)]; ok {
		return new(big.Int).SetBytes(balance), nil
	}
	return big.NewInt(0), nil
}
func (db *memoryDatabase) SetBalance(tokenTypeID *types.TokenTypeId, amount *big.Int) {
	if amount == nil {
		delete(db.storage, getBalanceKey(tokenTypeID))
	} else {
		db.storage[getBalanceKey(tokenTypeID)] = amount.Bytes()
	}
}
func (db *memoryDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (db *memoryDatabase) Reset()  {}
func (db *memoryDatabase) Finish() {}

func (db *memoryDatabase) SetContractCode(code []byte) {
	db.storage[getCodeKey(db.addr)] = code
}
func (db *memoryDatabase) GetContractCode() ([]byte, error) {
	if code, ok := db.storage[getCodeKey(db.addr)]; ok {
		return code, nil
	}
	return nil, nil
}

func (db *memoryDatabase) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	if code, ok := db.storage[getCodeKey(*addr)]; ok {
		return code, nil
	}
	return nil, nil
}

func (db *memoryDatabase) GetOriginalValue(key []byte) ([]byte, error) {
	if data, ok := db.originalStorage[hex.EncodeToString(key)]; ok {
		return data, nil
	}
	return nil, nil
}

func (db *memoryDatabase) GetValue(key []byte) ([]byte, error) {
	if data, ok := db.storage[hex.EncodeToString(key)]; ok {
		return data, nil
	}
	return nil, nil
}
func (db *memoryDatabase) SetValue(key []byte, value []byte) error {
	if len(value) == 0 {
		delete(db.storage, hex.EncodeToString(key))
	} else {
		db.storage[hex.EncodeToString(key)] = value
	}
	return nil
}
func (db *memoryDatabase) PrintStorage() string {
	str := "["
	for key, value := range db.storage {
		str += key + "=>" + hex.EncodeToString(value) + ", "
	}
	str += "]"
	return str
}
func (db *memoryDatabase) GetReceiptHash() *types.Hash {
	return &types.Hash{}
}
func (db *memoryDatabase) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *memoryDatabase) GetLogListHash() *types.Hash {
	if len(db.logList) == 0 {
		return nil
	}
	var source []byte
	for _, vmLog := range db.logList {
		for _, topic := range vmLog.Topics {
			source = append(source, topic.Bytes()...)
		}
		source = append(source, vmLog.Data...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash
}

func (db *memoryDatabase) GetLogList() ledger.VmLogList {
	return db.logList
}
func (db *memoryDatabase) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}

func (db *memoryDatabase) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	return nil, nil
}

func (db *memoryDatabase) Address() *types.Address {
	return &db.addr
}
func (db *memoryDatabase) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	return db.sb, nil
}
func (db *memoryDatabase) PrevAccountBlock() (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *memoryDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	sb, _ := db.LatestSnapshotBlock()
	return sb
}

func (db *memoryDatabase) GetUnsavedStorage() [][2][]byte {
	return nil
}

func (db *memoryDatabase) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}
func (db *memoryDatabase) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return nil
}
func (db *memoryDatabase) GetUnsavedContractCode() []byte {
	return nil
}

func (db *memoryDatabase) DebugGetStorage() (map[string][]byte, error) {
	return db.storage, nil
}

func (db *memoryDatabase) IsContractAccount() (bool, error) {
	return len(db.storage[getCodeKey(db.addr)]) > 0, nil
}

func (db *memoryDatabase) GetCallDepth(hash *types.Hash) (uint16, error) {
	return 0, nil
}
func (db *memoryDatabase) SetCallDepth(uint16) {
}

func (db *memoryDatabase) GetUnsavedCallDepth() uint16 {
	return 0
}

func (db *memoryDatabase) DeleteValue(key []byte) {
	delete(db.storage, hex.EncodeToString(key))
}

func (db *memoryDatabase) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return nil
}
func (db *memoryDatabase) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	list := make([]types.QuotaInfo, 75)
	for i := range list {
		list[i] = types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0}
	}
	return list
}

func (db *memoryDatabase) GetGlobalQuota() types.QuotaInfo {
	return types.QuotaInfo{}
}

func (db *memoryDatabase) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *memoryDatabase) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *memoryDatabase) SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta) {
}

func (db *memoryDatabase) GetContractMeta() (*ledger.ContractMeta, error) {
	return &ledger.ContractMeta{Gid: types.DELEGATE_GID, SendConfirmedTimes: 0, QuotaRatio: 10}, nil
}

func (db *memoryDatabase) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return db.LatestSnapshotBlock()
}
func (db *memoryDatabase) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (*ledger.ContractMeta, error) {
	return &ledger.ContractMeta{Gid: types.DELEGATE_GID, SendConfirmedTimes: 0, QuotaRatio: 10}, nil
}

func (db *memoryDatabase) GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (db *memoryDatabase) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return 0, nil
}
func (db *memoryDatabase) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}
func (db *memoryDatabase) CanWrite() bool {
	return false
}
