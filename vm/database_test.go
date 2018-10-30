package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
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

func (db *testDatabase) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
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
func (db *testDatabase) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	if height < uint64(len(db.snapshotBlockList)) {
		return db.snapshotBlockList[height-1]
	}
	return nil
}

// forward=true return [startHeight, startHeight+count), forward=false return (startHeight-count, startHeight]
func (db *testDatabase) GetSnapshotBlocks(startHeight uint64, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	blockList := make([]*ledger.SnapshotBlock, 0)
	var start, end uint64
	if forward {
		start = startHeight
		end = start + count
	} else {
		end = startHeight + 1
		start = end - count
	}
	for _, block := range db.snapshotBlockList {
		if block.Height >= start && block.Height < end {
			blockList = append(blockList, block)
		} else if block.Height >= end {
			break
		}
	}
	return blockList
}
func (db *testDatabase) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	for len := len(db.snapshotBlockList) - 1; len >= 0; len = len - 1 {
		block := db.snapshotBlockList[len]
		if block.Hash == *hash {
			return block
		}
	}
	return nil
}

func (db *testDatabase) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	if block, ok := db.accountBlockMap[db.addr][*hash]; ok {
		return block
	} else {
		return nil
	}
}
func (db *testDatabase) Reset() {}
func (db *testDatabase) IsAddressExisted(addr *types.Address) bool {
	_, ok := db.accountBlockMap[*addr]
	return ok
}
func (db *testDatabase) SetContractGid(gid *types.Gid, addr *types.Address) {
	db.contractGidMap[db.addr] = gid
}
func (db *testDatabase) SetContractCode(code []byte) {
	db.codeMap[db.addr] = code
}
func (db *testDatabase) GetContractCode(addr *types.Address) []byte {
	if code, ok := db.codeMap[*addr]; ok {
		return code
	} else {
		return nil
	}
}
func (db *testDatabase) GetStorage(addr *types.Address, key []byte) []byte {
	if data, ok := db.storageMap[*addr][string(key)]; ok {
		return data
	} else {
		return []byte{}
	}
}
func (db *testDatabase) SetStorage(key []byte, value []byte) {
	if _, ok := db.storageMap[db.addr]; !ok {
		db.storageMap[db.addr] = make(map[string][]byte)
	}
	db.storageMap[db.addr][string(key)] = value
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
func (db *testDatabase) GetStorageHash() *types.Hash {
	return &types.Hash{}
}
func (db *testDatabase) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *testDatabase) GetLogListHash() *types.Hash {
	return &types.Hash{}
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

func (db *testDatabase) NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator {
	storageMap := db.storageMap[*addr]
	items := make([]testIteratorItem, 0)
	for key, value := range storageMap {
		if len(prefix) > 0 {
			if bytes.Equal([]byte(key)[:len(prefix)], prefix) {
				items = append(items, testIteratorItem{[]byte(key), value})
			}
		} else {
			items = append(items, testIteratorItem{[]byte(key), value})
		}
	}
	return &testIterator{0, items}
}

func (db *testDatabase) CopyAndFreeze() vmctxt_interface.VmDatabase {
	return db
}

func (db *testDatabase) GetGid() *types.Gid {
	return nil
}
func (db *testDatabase) Address() *types.Address {
	return nil
}
func (db *testDatabase) CurrentSnapshotBlock() *ledger.SnapshotBlock {
	return db.snapshotBlockList[len(db.snapshotBlockList)-1]
}
func (db *testDatabase) PrevAccountBlock() *ledger.AccountBlock {
	var prevBlock *ledger.AccountBlock
	for _, block := range db.accountBlockMap[db.addr] {
		if prevBlock == nil || prevBlock.Height < block.Height {
			prevBlock = block
		}
	}
	return prevBlock
}
func (db *testDatabase) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func prepareDb(viteTotalSupply *big.Int) (db *testDatabase, addr1 types.Address, privKey ed25519.PrivateKey, hash12 types.Hash, snapshot2 *ledger.SnapshotBlock, timestamp int64) {
	addr1, _ = types.BytesToAddress(helper.HexToBytes("6c1032417f80329f3abe0a024fa3a7aa0e952b0f"))
	privKey, _ = ed25519.HexToPrivateKey("44e9768b7d8320a282e75337df8fc1f12a4f000b9f9906ddb886c6823bb599addfda7318e7824d25aae3c749c1cbd4e72ce9401653c66479554a05a2e3cb4f88")
	db = NewNoDatabase()
	db.storageMap[contracts.AddressMintage] = make(map[string][]byte)
	viteTokenIdLoc, _ := types.BytesToHash(helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), 32))
	db.storageMap[contracts.AddressMintage][string(viteTokenIdLoc.Bytes())], _ = contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, "ViteToken", "ViteToken", viteTotalSupply, uint8(18), addr1, big.NewInt(0), uint64(0))

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
	}
	db.accountBlockMap[addr1][hash12] = block12

	db.balanceMap[addr1] = make(map[types.TokenTypeId]*big.Int)
	db.balanceMap[addr1][ledger.ViteTokenId] = new(big.Int).Set(viteTotalSupply)

	db.storageMap[contracts.AddressConsensusGroup] = make(map[string][]byte)
	consensusGroupKey, _ := types.BytesToHash(contracts.GetConsensusGroupKey(types.SNAPSHOT_GID))
	consensusGroupData, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
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
	db.storageMap[contracts.AddressConsensusGroup][string(consensusGroupKey.Bytes())] = consensusGroupData
	consensusGroupKey, _ = types.BytesToHash(contracts.GetConsensusGroupKey(types.DELEGATE_GID))
	consensusGroupData, _ = contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
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
	db.storageMap[contracts.AddressConsensusGroup][string(consensusGroupKey.Bytes())] = consensusGroupData
	db.storageMap[contracts.AddressPledge] = make(map[string][]byte)
	db.storageMap[contracts.AddressPledge][string(contracts.GetPledgeBeneficialKey(addr1))], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))

	db.storageMap[contracts.AddressRegister] = make(map[string][]byte)
	db.storageMap[contracts.AddressRegister][string(contracts.GetRegisterKey("s1", types.SNAPSHOT_GID))], _ = contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, "s1", addr1, addr1, helper.Big0, uint64(1), uint64(1), uint64(0), []types.Address{addr1})
	db.storageMap[contracts.AddressRegister][string(contracts.GetRegisterKey("s2", types.SNAPSHOT_GID))], _ = contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, "s2", addr1, addr1, helper.Big0, uint64(1), uint64(1), uint64(0), []types.Address{addr1})
	return
}
