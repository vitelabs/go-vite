package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/vm_db"
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
	contractGidMap    map[types.Address]*types.Gid
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
		contractGidMap:    make(map[types.Address]*types.Gid),
		logList:           make([]*ledger.VmLog, 0),
		snapshotBlockList: make([]*ledger.SnapshotBlock, 0),
		accountBlockMap:   make(map[types.Address]map[types.Hash]*ledger.AccountBlock),
	}
}

func (db *testDatabase) Address() *types.Address {
	return &db.addr
}
func (db *testDatabase) LatestSnapshotBlock() *ledger.SnapshotBlock {
	return db.snapshotBlockList[len(db.snapshotBlockList)-1]
}

func (db *testDatabase) PrevAccountBlock() *ledger.AccountBlock {
	m := db.accountBlockMap[db.addr]
	var prevBlock *ledger.AccountBlock
	for _, b := range m {
		if prevBlock == nil || b.Height > prevBlock.Height {
			prevBlock = b
		}
	}
	return prevBlock
}
func (db *testDatabase) IsContractAccount() (bool, error) {
	return len(db.codeMap[db.addr]) > 0, nil
}

func (db *testDatabase) GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error) {
	return 0, nil
}

func (db *testDatabase) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return 0, 0
}
func (db *testDatabase) GetReceiptHash() *types.Hash {
	return &types.Hash{}
}
func (db *testDatabase) Reset() {
}

func (db *testDatabase) GetBalance(tokenTypeId *types.TokenTypeId) *big.Int {
	if balance, ok := db.balanceMap[db.addr][*tokenTypeId]; ok {
		return new(big.Int).Set(balance)
	} else {
		return big.NewInt(0)
	}
}
func (db *testDatabase) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	balance, ok := db.balanceMap[db.addr][*tokenTypeId]
	if ok && balance.Cmp(amount) >= 0 {
		db.balanceMap[db.addr][*tokenTypeId] = new(big.Int).Sub(balance, amount)
	}
}
func (db *testDatabase) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	if balance, ok := db.balanceMap[db.addr][*tokenTypeId]; ok {
		db.balanceMap[db.addr][*tokenTypeId] = new(big.Int).Add(balance, amount)
	} else {
		if _, ok := db.balanceMap[db.addr]; !ok {
			db.balanceMap[db.addr] = make(map[types.TokenTypeId]*big.Int)
		}
		db.balanceMap[db.addr][*tokenTypeId] = amount
	}

}
func (db *testDatabase) SetContractMeta(meta *ledger.ContractMeta) {
	db.contractGidMap[db.addr] = meta.Gid
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

func (db *testDatabase) GetValue(key []byte) []byte {
	if data, ok := db.storageMap[db.addr][ToKey(key)]; ok {
		return data
	} else {
		return []byte{}
	}
}
func (db *testDatabase) SetValue(key []byte, value []byte) {
	if _, ok := db.storageMap[db.addr]; !ok {
		db.storageMap[db.addr] = make(map[string][]byte)
	}
	db.storageMap[db.addr][ToKey(key)] = value
}

func (db *testDatabase) GetOriginalValue(key []byte) []byte {
	return nil
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

func (i *testIterator) Next() (key, value []byte, ok bool) {
	if i.index < len(i.items) {
		item := i.items[i.index]
		i.index = i.index + 1
		return item.key, item.value, true
	}
	return []byte{}, []byte{}, false
}

func (db *testDatabase) NewStorageIterator(prefix []byte) vm_db.StorageIterator {
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
	return &testIterator{0, items}
}

func (db *testDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.snapshotBlockList[0]
}

func (db *testDatabase) GetUnconfirmedBlocks() ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *testDatabase) GetPledgeAmount(addr *types.Address) (*big.Int, error) {
	data := db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(*addr))]
	if len(data) > 0 {
		amount := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(amount, abi.VariableNamePledgeBeneficial, data)
		return amount.Amount, nil
	}
	return big.NewInt(0), nil
}

func (db *testDatabase) DebugGetStorage() map[string][]byte {
	return db.storageMap[db.addr]
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
		SnapshotHash:   snapshot1.Hash,
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
		SnapshotHash:   snapshot1.Hash,
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
	if db.GetBalance(&ledger.ViteTokenId).Cmp(totalSupply) != 0 {
		t.Fatalf("invalid account balance")
	}
	db.addr = types.AddressConsensusGroup
	groupList, _ := abi.GetActiveConsensusGroupList(db)
	if len(groupList) != 2 {
		t.Fatalf("invalid consensus group info")
	}
	db.addr = addr1
	if pledgeAmount, _ := db.GetPledgeAmount(&addr1); pledgeAmount == nil || pledgeAmount.Sign() < 0 {
		t.Fatalf("invalid pledge amount")
	}
	db.addr = types.AddressConsensusGroup
	registrationList, _ := abi.GetRegistrationList(db, types.SNAPSHOT_GID, addr1)
	if len(registrationList) != 2 {
		t.Fatalf("invalid registration list")
	}
}
