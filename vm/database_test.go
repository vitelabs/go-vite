package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/interfaces"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

type testDatabase struct {
	balanceMap        map[types.Address]map[types.TokenTypeId]*big.Int
	storageMap        map[types.Address]map[string][]byte
	codeMap           map[types.Address][]byte
	contractMetaMap   map[types.Address]*ledger.ContractMeta
	logList           []*ledger.VmLog
	snapshotBlockList []*ledger.SnapshotBlock
	accountBlockMap   map[types.Address]map[types.Hash]*ledger.AccountBlock
	addr              types.Address
}

func NewNoDatabase() *testDatabase {
	return &testDatabase{
		balanceMap:        make(map[types.Address]map[types.TokenTypeId]*big.Int),
		storageMap:        make(map[types.Address]map[string][]byte),
		codeMap:           make(map[types.Address][]byte),
		contractMetaMap:   make(map[types.Address]*ledger.ContractMeta),
		logList:           make([]*ledger.VmLog, 0),
		snapshotBlockList: make([]*ledger.SnapshotBlock, 0),
		accountBlockMap:   make(map[types.Address]map[types.Hash]*ledger.AccountBlock),
	}
}

func (db *testDatabase) Address() *types.Address {
	return &db.addr
}
func (db *testDatabase) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	return db.snapshotBlockList[len(db.snapshotBlockList)-1], nil
}

func (db *testDatabase) PrevAccountBlock() (*ledger.AccountBlock, error) {
	m := db.accountBlockMap[db.addr]
	var prevBlock *ledger.AccountBlock
	for _, b := range m {
		if prevBlock == nil || b.Height > prevBlock.Height {
			prevBlock = b
		}
	}
	return prevBlock, nil
}
func (db *testDatabase) IsContractAccount() (bool, error) {
	return len(db.codeMap[db.addr]) > 0, nil
}

func (db *testDatabase) GetCallDepth(hash *types.Hash) (uint16, error) {
	return 0, nil
}
func (db *testDatabase) SetCallDepth(uint16) {
}

func (db *testDatabase) GetUnsavedCallDepth() uint16 {
	return 0
}

func (db *testDatabase) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return 0, 0
}
func (db *testDatabase) GetReceiptHash() *types.Hash {
	return &types.Hash{}
}
func (db *testDatabase) Reset() {
}
func (db *testDatabase) Finish() {

}

func (db *testDatabase) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.balanceMap[db.addr][*tokenTypeId]; ok {
		return new(big.Int).Set(balance), nil
	} else {
		return big.NewInt(0), nil
	}
}
func (db *testDatabase) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	if amount == nil {
		delete(db.balanceMap[db.addr], *tokenTypeId)
	} else {
		if _, ok := db.balanceMap[db.addr]; !ok {
			db.balanceMap[db.addr] = make(map[types.TokenTypeId]*big.Int)
		}
		db.balanceMap[db.addr][*tokenTypeId] = new(big.Int).Set(amount)

	}
}
func (db *testDatabase) SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta) {
	db.contractMetaMap[toAddr] = meta
}
func (db *testDatabase) GetContractMeta() (*ledger.ContractMeta, error) {
	return db.contractMetaMap[db.addr], nil
}
func (db *testDatabase) SetContractCode(code []byte) {
	db.codeMap[db.addr] = code
}
func (db *testDatabase) GetContractCode() ([]byte, error) {
	if code, ok := db.codeMap[db.addr]; ok {
		return code, nil
	} else {
		return nil, nil
	}
}
func (db *testDatabase) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	if code, ok := db.codeMap[*addr]; ok {
		return code, nil
	} else {
		return nil, nil
	}
}
func (db *testDatabase) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return nil
}
func (db *testDatabase) GetUnsavedContractCode() []byte {
	return nil
}
func (db *testDatabase) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}

func (db *testDatabase) GetValue(key []byte) ([]byte, error) {
	if data, ok := db.storageMap[db.addr][ToKey(key)]; ok {
		return data, nil
	} else {
		return []byte{}, nil
	}
}
func (db *testDatabase) SetValue(key []byte, value []byte) error {
	if len(value) == 0 {
		delete(db.storageMap[db.addr], ToKey(key))
	}
	if _, ok := db.storageMap[db.addr]; !ok {
		db.storageMap[db.addr] = make(map[string][]byte)
	}
	db.storageMap[db.addr][ToKey(key)] = value
	return nil
}

func (db *testDatabase) GetOriginalValue(key []byte) ([]byte, error) {
	return nil, nil
}

func (db *testDatabase) DeleteValue(key []byte) {
	delete(db.storageMap[db.addr], ToKey(key))
}

func (db *testDatabase) PrintStorage(addr types.Address) string {
	if storage, ok := db.storageMap[addr]; ok {
		var str string
		for key, value := range storage {
			str += key + "=>" + hex.EncodeToString(value) + ", "
		}
		return str
	} else {
		return ""
	}
}

func (db *testDatabase) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *testDatabase) GetLogListHash() *types.Hash {
	return &types.Hash{}
}
func (db *testDatabase) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}

func (db *testDatabase) GetLogList() ledger.VmLogList {
	return db.logList
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

func (db *testDatabase) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	storageMap := db.storageMap[db.addr]
	items := make([]testIteratorItem, 0)
	for key, value := range storageMap {
		if len(prefix) > 0 {
			if bytes.Equal(ToBytes(key)[:len(prefix)], prefix) {
				items = append(items, testIteratorItem{ToBytes(key), value})
			}
		} else {
			items = append(items, testIteratorItem{ToBytes(key), value})
		}
	}
	return &testIterator{-1, items}, nil
}

func (db *testDatabase) GetUnsavedStorage() [][2][]byte {
	return nil
}

func (db *testDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.snapshotBlockList[0]
}

func (db *testDatabase) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return nil
}

func (db *testDatabase) GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	data := db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(*addr))]
	if len(data) > 0 {
		amount := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(amount, abi.VariableNamePledgeBeneficial, data)
		return amount.Amount, nil
	}
	return big.NewInt(0), nil
}

func (db *testDatabase) DebugGetStorage() (map[string][]byte, error) {
	return db.storageMap[db.addr], nil
}

func prepareDb(viteTotalSupply *big.Int) (db *testDatabase, addr1 types.Address, privKey ed25519.PrivateKey, hash12 types.Hash, snapshot2 *ledger.SnapshotBlock, timestamp int64) {
	addr1, _ = types.BytesToAddress(helper.HexToBytes("6c1032417f80329f3abe0a024fa3a7aa0e952b0f"))
	privKey, _ = ed25519.HexToPrivateKey("44e9768b7d8320a282e75337df8fc1f12a4f000b9f9906ddb886c6823bb599addfda7318e7824d25aae3c749c1cbd4e72ce9401653c66479554a05a2e3cb4f88")
	db = NewNoDatabase()
	db.storageMap[types.AddressMintage] = make(map[string][]byte)
	viteTokenIdKey := abi.GetMintageKey(ledger.ViteTokenId)
	var err error
	db.storageMap[types.AddressMintage][ToKey(viteTokenIdKey)], err = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, "ViteToken", "ViteToken", viteTotalSupply, uint8(18), addr1, big.NewInt(0), uint64(0), addr1, true, helper.Tt256m1, false)
	if err != nil {
		panic(err)
	}

	timestamp = 1536214502
	t1 := time.Unix(timestamp-1, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 1, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
	t2 := time.Unix(timestamp, 0)
	snapshot2 = &ledger.SnapshotBlock{Height: 2, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot2)

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
	db.accountBlockMap[addr1] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr1][hash11] = block11
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
	db.accountBlockMap[addr1][hash12] = block12

	db.balanceMap[addr1] = make(map[types.TokenTypeId]*big.Int)
	db.balanceMap[addr1][ledger.ViteTokenId] = new(big.Int).Set(viteTotalSupply)

	db.storageMap[types.AddressConsensusGroup] = make(map[string][]byte)
	consensusGroupKey := abi.GetConsensusGroupKey(types.SNAPSHOT_GID)
	consensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	db.storageMap[types.AddressConsensusGroup][ToKey(consensusGroupKey)] = consensusGroupData
	consensusGroupKey = abi.GetConsensusGroupKey(types.DELEGATE_GID)
	consensusGroupData, err = abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	db.storageMap[types.AddressConsensusGroup][ToKey(consensusGroupKey)] = consensusGroupData

	db.storageMap[types.AddressPledge] = make(map[string][]byte)
	db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(addr1))], err = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	if err != nil {
		panic(err)
	}

	db.storageMap[types.AddressConsensusGroup][ToKey(abi.GetRegisterKey("s1", types.SNAPSHOT_GID))], err = abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, "s1", addr1, addr1, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr1})
	if err != nil {
		panic(err)
	}
	db.storageMap[types.AddressConsensusGroup][ToKey(abi.GetRegisterKey("s2", types.SNAPSHOT_GID))], err = abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, "s2", addr1, addr1, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr1})
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

func TestPrepareDb(t *testing.T) {
	totalSupply := big.NewInt(1e18)
	db, addr1, _, _, _, _ := prepareDb(totalSupply)
	db.addr = types.AddressMintage
	tokenMap, _ := abi.GetTokenMap(db)
	if len(tokenMap) != 1 || tokenMap[ledger.ViteTokenId] == nil || tokenMap[ledger.ViteTokenId].TotalSupply.Cmp(totalSupply) != 0 {
		t.Fatalf("invalid token info")
	}
	if len(db.accountBlockMap) != 1 || len(db.accountBlockMap[addr1]) != 2 {
		t.Fatalf("invalid account block info")
	}
	db.addr = addr1
	balance, _ := db.GetBalance(&ledger.ViteTokenId)
	if totalSupply.Cmp(balance) != 0 {
		t.Fatalf("invalid account balance")
	}
	db.addr = types.AddressConsensusGroup
	groupList, _ := abi.GetActiveConsensusGroupList(db)
	if len(groupList) != 2 {
		t.Fatalf("invalid consensus group info")
	}
	db.addr = addr1
	if pledgeAmount, _ := db.GetPledgeBeneficialAmount(&addr1); pledgeAmount == nil || pledgeAmount.Sign() < 0 {
		t.Fatalf("invalid pledge amount")
	}
	db.addr = types.AddressConsensusGroup
	registrationList, _ := abi.GetRegistrationList(db, types.SNAPSHOT_GID, addr1)
	if len(registrationList) != 2 {
		t.Fatalf("invalid registration list")
	}
}
