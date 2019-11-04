package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sort"
	"time"
)

type mockDB struct {
	currentAddr            *types.Address
	latestSnapshotBlock    *ledger.SnapshotBlock
	forkSnapshotBlockMap   map[uint64]*ledger.SnapshotBlock
	prevAccountBlock       *ledger.AccountBlock
	quotaInfo              []types.QuotaInfo
	pledgeBeneficialAmount *big.Int
	balanceMap             map[types.TokenTypeId]*big.Int
	balanceMapOrigin       map[types.TokenTypeId]*big.Int
	storageMap             map[string]string
	storageMapOrigin       map[string]string
	contractMetaMap        map[types.Address]*ledger.ContractMeta
	contractMetaMapOrigin  map[types.Address]*ledger.ContractMeta
	logList                []*ledger.VmLog
	code                   []byte
	genesisBlock           *ledger.SnapshotBlock
}

func NewMockDB(addr *types.Address,
	latestSnapshotBlock *ledger.SnapshotBlock,
	prevAccountBlock *ledger.AccountBlock,
	quotaInfo []types.QuotaInfo,
	pledgeBeneficialAmount *big.Int,
	balanceMap map[types.TokenTypeId]string,
	storage map[string]string,
	contractMetaMap map[types.Address]*ledger.ContractMeta,
	code []byte,
	genesisTimestamp int64,
	snapshotBlockMap map[uint64]*ledger.SnapshotBlock) (*mockDB, error) {
	db := &mockDB{currentAddr: addr,
		latestSnapshotBlock:    latestSnapshotBlock,
		prevAccountBlock:       prevAccountBlock,
		quotaInfo:              quotaInfo,
		pledgeBeneficialAmount: new(big.Int).Set(pledgeBeneficialAmount),
		logList:                make([]*ledger.VmLog, 0),
		balanceMap:             make(map[types.TokenTypeId]*big.Int),
		storageMap:             make(map[string]string),
		contractMetaMap:        make(map[types.Address]*ledger.ContractMeta),
		code:                   code,
		forkSnapshotBlockMap:   snapshotBlockMap,
	}
	balanceMapCopy := make(map[types.TokenTypeId]*big.Int)
	for tid, amount := range balanceMap {
		var ok bool
		balanceMapCopy[tid], ok = new(big.Int).SetString(amount, 16)
		if !ok {
			return nil, errors.New("invalid balance amount " + amount)
		}
	}
	db.balanceMapOrigin = balanceMapCopy

	storageMapCopy := make(map[string]string)
	for k, v := range storage {
		storageMapCopy[k] = v
	}
	db.storageMapOrigin = storageMapCopy

	contractMetaMapCopy := make(map[types.Address]*ledger.ContractMeta)
	for k, v := range contractMetaMap {
		contractMetaMapCopy[k] = v
	}
	db.contractMetaMapOrigin = contractMetaMapCopy
	genesisTime := time.Unix(genesisTimestamp, 0)
	db.genesisBlock = &ledger.SnapshotBlock{
		Height:    1,
		Timestamp: &genesisTime,
	}
	return db, nil
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
	return db.prevAccountBlock, nil
}
func (db *mockDB) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if addr != *db.currentAddr {
		return nil, errors.New("current account address not match")
	} else {
		return db.prevAccountBlock, nil
	}
}
func (db *mockDB) IsContractAccount() (bool, error) {
	if !types.IsContractAddr(*db.currentAddr) {
		return false, nil
	}
	if meta, err := db.GetContractMeta(); err != nil {
		return false, err
	} else {
		return meta != nil, nil
	}
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

type mockDBStorageKv struct {
	k string
	v string
}
type byKey []*mockDBStorageKv

func (a byKey) Len() int      { return len(a) }
func (a byKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byKey) Less(i, j int) bool {
	return a[i].k < a[j].k
}
func (db *mockDB) GetReceiptHash() *types.Hash {
	list := make([]*mockDBStorageKv, 0, len(db.storageMap))
	for k, v := range db.storageMap {
		list = append(list, &mockDBStorageKv{k, v})
	}
	sort.Sort(byKey(list))
	var source []byte
	for _, kv := range list {
		source = append(source, stringToBytes(kv.k)...)
		source = append(source, stringToBytes(kv.v)...)
	}
	if len(source) == 0 {
		return &types.Hash{}
	}
	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash
}
func (db *mockDB) Reset() {
	db.balanceMap = make(map[types.TokenTypeId]*big.Int)
	db.storageMap = make(map[string]string)
	db.contractMetaMap = make(map[types.Address]*ledger.ContractMeta)
	db.logList = make([]*ledger.VmLog, 0)
}
func (db *mockDB) Finish() {

}
func (db *mockDB) GetValue(key []byte) ([]byte, error) {
	keyStr := bytesToString(key)
	if v, ok := db.storageMap[keyStr]; ok {
		return stringToBytes(v), nil
	}
	if v, ok := db.storageMapOrigin[keyStr]; ok {
		return stringToBytes(v), nil
	}
	return nil, nil
}
func (db *mockDB) GetOriginalValue(key []byte) ([]byte, error) {
	if v, ok := db.storageMapOrigin[bytesToString(key)]; ok {
		return stringToBytes(v), nil
	}
	return nil, nil
}
func (db *mockDB) SetValue(key []byte, value []byte) error {
	db.storageMap[bytesToString(key)] = hex.EncodeToString(value)
	return nil
}
func (db *mockDB) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	items := make([]mockIteratorItem, 0)
	for key, value := range db.storageMap {
		if len(value) == 0 {
			continue
		}
		keyBytes := stringToBytes(key)
		prefixLen := len(prefix)
		if prefixLen > 0 {
			if len(keyBytes) >= prefixLen && bytes.Equal(keyBytes[:prefixLen], prefix) {
				items = append(items, mockIteratorItem{keyBytes, stringToBytes(value)})
			}
		} else {
			items = append(items, mockIteratorItem{keyBytes, stringToBytes(value)})
		}
	}
	for key, value := range db.storageMapOrigin {
		if _, ok := db.storageMap[key]; ok {
			continue
		}
		keyBytes := stringToBytes(key)
		prefixLen := len(prefix)
		if prefixLen > 0 {
			if len(keyBytes) >= prefixLen && bytes.Equal(keyBytes[:prefixLen], prefix) {
				items = append(items, mockIteratorItem{keyBytes, stringToBytes(value)})
			}
		} else {
			items = append(items, mockIteratorItem{keyBytes, stringToBytes(value)})
		}
	}
	sort.Sort(mockIteratorSorter(items))
	return &mockIterator{-1, items}, nil
	return nil, nil
}

type mockIteratorItem struct {
	key, value []byte
}
type mockIterator struct {
	index int
	items []mockIteratorItem
}

func (i *mockIterator) Next() (ok bool) {
	if i.index < len(i.items)-1 {
		i.index = i.index + 1
		return true
	}
	return false
}
func (i *mockIterator) Prev() bool {
	return i.index <= 0
}
func (i *mockIterator) Last() bool {
	return i.index == len(i.items)-1
}
func (i *mockIterator) Key() []byte {
	return i.items[i.index].key
}
func (i *mockIterator) Value() []byte {
	return i.items[i.index].value
}
func (i *mockIterator) Error() error {
	return nil
}
func (i *mockIterator) Release() {

}
func (i *mockIterator) Seek(key []byte) bool {
	for index, item := range i.items {
		if bytes.Equal(item.key, key) {
			i.index = index
			return true
		}
	}
	return false
}

type mockIteratorSorter []mockIteratorItem

func (st mockIteratorSorter) Len() int {
	return len(st)
}
func (st mockIteratorSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}
func (st mockIteratorSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].key, st[j].key)
	if tkCmp < 0 {
		return true
	} else {
		return false
	}
}
func (db *mockDB) GetUnsavedStorage() [][2][]byte {
	return nil
}
func (db *mockDB) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.balanceMap[*tokenTypeId]; ok {
		return new(big.Int).Set(balance), nil
	}
	if balance, ok := db.balanceMapOrigin[*tokenTypeId]; ok {
		return new(big.Int).Set(balance), nil
	}
	return big.NewInt(0), nil
}
func (db *mockDB) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	db.balanceMap[*tokenTypeId] = amount
}
func (db *mockDB) GetBalanceMap() (map[types.TokenTypeId]*big.Int, error) {
	balanceMap := make(map[types.TokenTypeId]*big.Int)
	for tid, amount := range db.balanceMap {
		balanceMap[tid] = new(big.Int).Set(amount)
	}
	for tid, amount := range db.balanceMapOrigin {
		if _, ok := balanceMap[tid]; !ok {
			balanceMap[tid] = new(big.Int).Set(amount)
		}
	}
	return balanceMap, nil
}
func (db *mockDB) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}
func (db *mockDB) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *mockDB) GetLogList() ledger.VmLogList {
	return db.logList
}
func (db *mockDB) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}
func (db *mockDB) GetLogListHash() *types.Hash {
	if len(db.logList) == 0 {
		return nil
	}
	var source []byte
	for _, log := range db.logList {
		for _, topic := range log.Topics {
			source = append(source, topic.Bytes()...)
		}
		source = append(source, log.Data...)
	}
	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash
}
func (db *mockDB) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return nil
}
func (db *mockDB) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.genesisBlock
}
func (db *mockDB) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return db.forkSnapshotBlockMap[height], nil
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
	if meta := ledger.GetBuiltinContractMeta(*db.currentAddr); meta != nil {
		return meta, nil
	}
	if meta, ok := db.contractMetaMap[*db.currentAddr]; ok {
		return meta, nil
	}
	if meta, ok := db.contractMetaMapOrigin[*db.currentAddr]; ok {
		return meta, nil
	}
	return nil, nil
}
func (db *mockDB) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error) {
	if meta := ledger.GetBuiltinContractMeta(contractAddress); meta != nil {
		return meta, nil
	}
	if meta, ok := db.contractMetaMap[contractAddress]; ok {
		return meta, nil
	}
	if meta, ok := db.contractMetaMapOrigin[contractAddress]; ok {
		return meta, nil
	}
	return nil, nil
}
func (db *mockDB) getContractMetaMap() map[types.Address]*ledger.ContractMeta {
	metaMap := make(map[types.Address]*ledger.ContractMeta)
	for addr, meta := range db.contractMetaMap {
		metaMap[addr] = meta
	}
	for addr, meta := range db.contractMetaMapOrigin {
		if _, ok := metaMap[addr]; !ok {
			metaMap[addr] = meta
		}
	}
	return metaMap
}
func (db *mockDB) getStorageMap() map[string]string {
	storageMap := make(map[string]string)
	for k, v := range db.storageMap {
		storageMap[k] = v
	}
	for k, v := range db.storageMapOrigin {
		if _, ok := storageMap[k]; !ok {
			storageMap[k] = v
		}
	}
	return storageMap
}
func (db *mockDB) SetContractCode(code []byte) {
	db.code = code
}
func (db *mockDB) GetContractCode() ([]byte, error) {
	return db.code, nil
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
func (db *mockDB) GetStakeBeneficialAmount(addr *types.Address) (*big.Int, error) {
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
func bytesToString(v []byte) string {
	return hex.EncodeToString(v)
}
func stringToBytes(v string) []byte {
	vBytes, _ := hex.DecodeString(v)
	return vBytes
}
