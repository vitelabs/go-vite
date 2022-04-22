package vm

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"sort"
	"time"

	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/util"
)

type TestDatabase struct {
	BalanceMap map[types.Address]map[types.TokenTypeId]*big.Int
	StorageMap map[types.Address]map[string][]byte
	CodeMap    map[types.Address][]byte
	ContractMetaMap map[types.Address]*ledger.ContractMeta
	LogList           []*ledger.VmLog
	SnapshotBlockList []*ledger.SnapshotBlock
	AccountBlockMap     map[types.Address]map[types.Hash]*ledger.AccountBlock
	Addr                types.Address
	ExecutionContextMap map[types.Hash]*ledger.ExecutionContext
}

func NewNoDatabase() *TestDatabase {
	return &TestDatabase{
		BalanceMap:          make(map[types.Address]map[types.TokenTypeId]*big.Int),
		StorageMap:          make(map[types.Address]map[string][]byte),
		CodeMap:             make(map[types.Address][]byte),
		ContractMetaMap:     make(map[types.Address]*ledger.ContractMeta),
		LogList:             make([]*ledger.VmLog, 0),
		SnapshotBlockList:   make([]*ledger.SnapshotBlock, 0),
		AccountBlockMap:     make(map[types.Address]map[types.Hash]*ledger.AccountBlock),
		ExecutionContextMap: make(map[types.Hash]*ledger.ExecutionContext),
	}
}

func (db *TestDatabase) Address() *types.Address {
	return &db.Addr
}
func (db *TestDatabase) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	return db.SnapshotBlockList[len(db.SnapshotBlockList)-1], nil
}

func (db *TestDatabase) PrevAccountBlock() (*ledger.AccountBlock, error) {
	m := db.AccountBlockMap[db.Addr]
	var prevBlock *ledger.AccountBlock
	for _, b := range m {
		if prevBlock == nil || b.Height > prevBlock.Height {
			prevBlock = b
		}
	}
	return prevBlock, nil
}

func (db *TestDatabase) GetCallDepth(hash *types.Hash) (uint16, error) {
	return 0, nil
}
func (db *TestDatabase) SetCallDepth(uint16) {
}

func (db *TestDatabase) GetUnsavedCallDepth() uint16 {
	return 0
}

func (db *TestDatabase) GetReceiptHash() *types.Hash {
	return &types.Hash{}
}
func (db *TestDatabase) Reset() {
}
func (db *TestDatabase) Finish() {

}

func (db *TestDatabase) GetBalance(tokenTypeID *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.BalanceMap[db.Addr][*tokenTypeID]; ok {
		return new(big.Int).Set(balance), nil
	}
	return big.NewInt(0), nil
}
func (db *TestDatabase) SetBalance(tokenTypeID *types.TokenTypeId, amount *big.Int) {
	if amount == nil {
		delete(db.BalanceMap[db.Addr], *tokenTypeID)
	} else {
		if _, ok := db.BalanceMap[db.Addr]; !ok {
			db.BalanceMap[db.Addr] = make(map[types.TokenTypeId]*big.Int)
		}
		db.BalanceMap[db.Addr][*tokenTypeID] = new(big.Int).Set(amount)

	}
}
func (db *TestDatabase) SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta) {
	db.ContractMetaMap[toAddr] = meta
}
func (db *TestDatabase) GetContractMeta() (*ledger.ContractMeta, error) {
	if types.IsBuiltinContractAddrInUse(db.Addr) {
		return &ledger.ContractMeta{QuotaRatio: 10}, nil
	}
	return db.ContractMetaMap[db.Addr], nil
}
func (db *TestDatabase) SetContractCode(code []byte) {
	db.CodeMap[db.Addr] = code
}
func (db *TestDatabase) GetContractCode() ([]byte, error) {
	if code, ok := db.CodeMap[db.Addr]; ok {
		return code, nil
	}
	return nil, nil
}
func (db *TestDatabase) GetDeployedContractCode(deployedContractAddr types.Address, callerAddr types.Address) ([]byte, error) {
	return db.GetContractCode()
}
func (db *TestDatabase) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	if code, ok := db.CodeMap[*addr]; ok {
		return code, nil
	}
	return nil, nil
}
func (db *TestDatabase) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return nil
}
func (db *TestDatabase) GetUnsavedContractCode() []byte {
	return nil
}
func (db *TestDatabase) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}

func (db *TestDatabase) GetValue(key []byte) ([]byte, error) {
	if data, ok := db.StorageMap[db.Addr][ToKey(key)]; ok {
		return data, nil
	}
	return []byte{}, nil
}
func (db *TestDatabase) SetValue(key []byte, value []byte) error {
	if _, ok := db.StorageMap[db.Addr]; !ok {
		db.StorageMap[db.Addr] = make(map[string][]byte)
	}
	if len(value) == 0 {
		delete(db.StorageMap[db.Addr], ToKey(key))
	} else {
		db.StorageMap[db.Addr][ToKey(key)] = value
	}
	return nil
}

func (db *TestDatabase) GetOriginalValue(key []byte) ([]byte, error) {
	return nil, nil
}

func (db *TestDatabase) DeleteValue(key []byte) {
	delete(db.StorageMap[db.Addr], ToKey(key))
}

func (db *TestDatabase) PrintStorage(addr types.Address) string {
	if storage, ok := db.StorageMap[addr]; ok {
		var str string
		for key, value := range storage {
			str += key + "=>" + hex.EncodeToString(value) + ", "
		}
		return str
	}
	return ""
}

func (db *TestDatabase) AddLog(log *ledger.VmLog) {
	db.LogList = append(db.LogList, log)
}
func (db *TestDatabase) GetLogListHash() *types.Hash {
	return &types.Hash{}
}
func (db *TestDatabase) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}

func (db *TestDatabase) GetLogList() ledger.VmLogList {
	return db.LogList
}

func (db *TestDatabase) GetExecutionContext(blockHash *types.Hash) (*ledger.ExecutionContext, error) {
	return db.ExecutionContextMap[*blockHash], nil
}

func (db *TestDatabase) SetExecutionContext(blockHash *types.Hash, context *ledger.ExecutionContext) {
	db.ExecutionContextMap[*blockHash] = context
}

func (db *TestDatabase) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return db.LatestSnapshotBlock()
}
func (db *TestDatabase) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (*ledger.ContractMeta, error) {
	meta := db.ContractMetaMap[contractAddress]
	return meta, nil
}
func (db *TestDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	for _, sb := range db.SnapshotBlockList {
		if sb.Height == height {
			return sb, nil
		}
	}
	return nil, nil
}
func (db *TestDatabase) CanWrite() bool {
	return false
}

type testIteratorItem struct {
	key, value []byte
}
type testIterator struct {
	index int
	items []testIteratorItem
}

func (i *testIterator) Next() (ok bool) {
	if i.index < len(i.items)-1 {
		i.index = i.index + 1
		return true
	}
	return false
}
func (i *testIterator) Prev() bool {
	return i.index <= 0
}
func (i *testIterator) Last() bool {
	return i.index == len(i.items)-1
}
func (i *testIterator) Key() []byte {
	return i.items[i.index].key
}
func (i *testIterator) Value() []byte {
	return i.items[i.index].value
}
func (i *testIterator) Error() error {
	return nil
}
func (i *testIterator) Release() {

}
func (i *testIterator) Seek(key []byte) bool {
	for index, item := range i.items {
		if bytes.Equal(item.key, key) {
			i.index = index
			return true
		}
	}
	return false
}

func (db *TestDatabase) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	storageMap := db.StorageMap[db.Addr]
	items := make([]testIteratorItem, 0)
	for key, value := range storageMap {
		keyBytes := ToBytes(key)
		prefixLen := len(prefix)
		if prefixLen > 0 {
			if len(keyBytes) >= prefixLen && bytes.Equal(keyBytes[:prefixLen], prefix) {
				items = append(items, testIteratorItem{keyBytes, value})
			}
		} else {
			items = append(items, testIteratorItem{keyBytes, value})
		}
	}
	sort.Sort(testIteratorSorter(items))
	return &testIterator{-1, items}, nil
}

type testIteratorSorter []testIteratorItem

func (st testIteratorSorter) Len() int {
	return len(st)
}

func (st testIteratorSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st testIteratorSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].key, st[j].key)
	if tkCmp < 0 {
		return true
	}
	return false
}

func (db *TestDatabase) GetUnsavedStorage() [][2][]byte {
	return nil
}

func (db *TestDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.SnapshotBlockList[0]
}

func (db *TestDatabase) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return nil
}
func (db *TestDatabase) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	list := make([]types.QuotaInfo, 75)
	for i := range list {
		list[i] = types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0}
	}
	return list
}

func (db *TestDatabase) GetGlobalQuota() types.QuotaInfo {
	return types.QuotaInfo{}
}

func (db *TestDatabase) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *TestDatabase) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *TestDatabase) GetStakeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	data, _ := db.StorageMap[types.AddressQuota][ToKey(abi.GetStakeBeneficialKey(*addr))]
	if len(data) > 0 {
		amount := new(abi.VariableStakeBeneficial)
		abi.ABIQuota.UnpackVariable(amount, abi.VariableNameStakeBeneficial, data)
		return amount.Amount, nil
	}
	return big.NewInt(0), nil
}

func (db *TestDatabase) DebugGetStorage() (map[string][]byte, error) {
	return db.StorageMap[db.Addr], nil
}
func (db *TestDatabase) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return 1, nil
}
func (db *TestDatabase) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if m, ok := db.AccountBlockMap[addr]; ok {
		var block *ledger.AccountBlock
		for _, b := range m {
			if block == nil || block.Height < b.Height {
				block = b
			}
		}
		return block, nil
	}
	return nil, nil
}

func PrepareDb(viteTotalSupply *big.Int) (db *TestDatabase, addr1 types.Address, privKey ed25519.PrivateKey, hash12 types.Hash, snapshot2 *ledger.SnapshotBlock, timestamp int64) {
	addr1, _ = types.BytesToAddress(helper.HexToBytes("6c1032417f80329f3abe0a024fa3a7aa0e952b0f00"))
	privKey, _ = ed25519.HexToPrivateKey("44e9768b7d8320a282e75337df8fc1f12a4f000b9f9906ddb886c6823bb599addfda7318e7824d25aae3c749c1cbd4e72ce9401653c66479554a05a2e3cb4f88")
	db = NewNoDatabase()
	db.StorageMap[types.AddressAsset] = make(map[string][]byte)
	viteTokenIDKey := abi.GetTokenInfoKey(ledger.ViteTokenId)
	var err error
	db.StorageMap[types.AddressAsset][ToKey(viteTokenIDKey)], err = abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, "ViteToken", "ViteToken", viteTotalSupply, uint8(18), addr1, true, helper.Tt256m1, false, uint16(1))
	if err != nil {
		panic(err)
	}

	timestamp = 1536214502
	t1 := time.Unix(timestamp-1, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 1, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot1)
	t2 := time.Unix(timestamp, 0)
	snapshot2 = &ledger.SnapshotBlock{Height: 2, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot2)

	hash11 := types.DataHash([]byte{1, 1})
	block11 := &ledger.AccountBlock{
		Height:         1,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash11,
	}
	db.AccountBlockMap[addr1] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr1][hash11] = block11
	hash12 = types.DataHash([]byte{1, 2})
	block12 := &ledger.AccountBlock{
		Height:         2,
		ToAddress:      addr1,
		AccountAddress: addr1,
		FromBlockHash:  hash11,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash11,
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash12,
	}
	db.AccountBlockMap[addr1][hash12] = block12

	db.BalanceMap[addr1] = make(map[types.TokenTypeId]*big.Int)
	db.BalanceMap[addr1][ledger.ViteTokenId] = new(big.Int).Set(viteTotalSupply)

	db.StorageMap[types.AddressGovernance] = make(map[string][]byte)
	consensusGroupKey := abi.GetConsensusGroupInfoKey(types.SNAPSHOT_GID)
	consensusGroupData, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(50),
		uint16(1),
		uint8(0),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(3600*24*90).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		addr1,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		panic(err)
	}
	db.StorageMap[types.AddressGovernance][ToKey(consensusGroupKey)] = consensusGroupData
	consensusGroupKey = abi.GetConsensusGroupInfoKey(types.DELEGATE_GID)
	consensusGroupData, err = abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		uint16(48),
		uint8(1),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(3600*24*90).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		addr1,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		panic(err)
	}
	db.StorageMap[types.AddressGovernance][ToKey(consensusGroupKey)] = consensusGroupData

	db.StorageMap[types.AddressQuota] = make(map[string][]byte)
	db.StorageMap[types.AddressQuota][ToKey(abi.GetStakeBeneficialKey(addr1))], err = abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	if err != nil {
		panic(err)
	}

	db.StorageMap[types.AddressGovernance][ToKey(abi.GetRegistrationInfoKey("s1", types.SNAPSHOT_GID))], err = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, "s1", addr1, addr1, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr1})
	if err != nil {
		panic(err)
	}
	db.StorageMap[types.AddressGovernance][ToKey(abi.GetRegistrationInfoKey("s2", types.SNAPSHOT_GID))], err = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, "s2", addr1, addr1, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr1})
	if err != nil {
		panic(err)
	}
	return
}

func ToKey(key []byte) string {
	return hex.EncodeToString(key)
}
func ToBytes(key string) []byte {
	bs, _ := hex.DecodeString(key)
	return bs
}

//func TestPrepareDb(t *testing.T) {
//	totalSupply := big.NewInt(1e18)
//	db, addr1, _, _, _, _ := PrepareDb(totalSupply)
//	db.addr = types.AddressAsset
//	tokenMap, _ := abi.GetTokenMap(db)
//	if len(tokenMap) != 1 || tokenMap[ledger.ViteTokenId] == nil || tokenMap[ledger.ViteTokenId].TotalSupply.Cmp(totalSupply) != 0 {
//		t.Fatalf("invalid token info")
//	}
//	if len(db.accountBlockMap) != 1 || len(db.accountBlockMap[addr1]) != 2 {
//		t.Fatalf("invalid account block info")
//	}
//	db.addr = addr1
//	balance, _ := db.GetBalance(&ledger.ViteTokenId)
//	if totalSupply.Cmp(balance) != 0 {
//		t.Fatalf("invalid account balance")
//	}
//	db.addr = types.AddressGovernance
//	groupList, _ := abi.GetConsensusGroupList(db)
//	if len(groupList) != 2 {
//		t.Fatalf("invalid consensus group info")
//	}
//	db.addr = addr1
//	if stakeAmount, _ := db.GetStakeBeneficialAmount(&addr1); stakeAmount == nil || stakeAmount.Sign() < 0 {
//		t.Fatalf("invalid stake amount")
//	}
//	db.addr = types.AddressGovernance
//	registrationList, _ := abi.GetRegistrationList(db, types.SNAPSHOT_GID, addr1)
//	if len(registrationList) != 2 {
//		t.Fatalf("invalid registration list")
//	}
//}
