package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type testDatabase struct {
	balanceMap        map[types.Address]map[types.TokenTypeId]*big.Int
	storageMap        map[types.Address]map[types.Hash][]byte
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
		storageMap:        make(map[types.Address]map[types.Hash][]byte),
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
func (db *testDatabase) GetSnapshotBlock(hash *types.Hash) *ledger.SnapshotBlock {
	for len := len(db.snapshotBlockList) - 1; len >= 0; len = len - 1 {
		block := db.snapshotBlockList[len]
		if bytes.Equal(block.Hash.Bytes(), hash.Bytes()) {
			return block
		}
	}
	return nil

}
func (db *testDatabase) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	if height < uint64(len(db.snapshotBlockList)) {
		return db.snapshotBlockList[height-1]
	}
	return nil
}
func (db *testDatabase) GetSnapshotBlocks(startHeight uint64, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	if forward {
		start := startHeight
		end := start + count
		return db.snapshotBlockList[start:end]
	} else {
		end := startHeight - 1
		start := end - count
		return db.snapshotBlockList[start:end]
	}
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
	locHash, _ := types.BytesToHash(key)
	if data, ok := db.storageMap[*addr][locHash]; ok {
		return data
	} else {
		return []byte{}
	}
}
func (db *testDatabase) SetStorage(key []byte, value []byte) {
	locHash, _ := types.BytesToHash(key)
	if _, ok := db.storageMap[db.addr]; !ok {
		db.storageMap[db.addr] = make(map[types.Hash][]byte)
	}
	db.storageMap[db.addr][locHash] = value
}
func (db *testDatabase) PrintStorage(addr types.Address) string {
	if storage, ok := db.storageMap[addr]; ok {
		var str string
		for key, value := range storage {
			str += hex.EncodeToString(key.Bytes()) + "=>" + hex.EncodeToString(value) + ", "
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

func (db *testDatabase) NewStorageIterator(prefix []byte) vmctxt_interface.StorageIterator {
	storageMap := db.storageMap[db.addr]
	items := make([]testIteratorItem, 0)
	for key, value := range storageMap {
		if len(prefix) > 0 {
			if bytes.Equal(key.Bytes()[:len(prefix)], prefix) {
				items = append(items, testIteratorItem{key.Bytes(), value})
			}
		} else {
			items = append(items, testIteratorItem{key.Bytes(), value})
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
	return nil
}
func (db *testDatabase) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func prepareDb(viteTotalSupply *big.Int) (db *testDatabase, addr1 types.Address, hash12 types.Hash, snapshot2 *ledger.SnapshotBlock, timestamp int64) {
	addr1, _ = types.BytesToAddress(helper.HexToBytes("CA35B7D915458EF540ADE6068DFE2F44E8FA733C"))
	db = NewNoDatabase()
	db.storageMap[contracts.AddressMintage] = make(map[types.Hash][]byte)
	viteTokenIdLoc, _ := types.BytesToHash(helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), 32))
	db.storageMap[contracts.AddressMintage][viteTokenIdLoc], _ = contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, "ViteToken", "ViteToken", viteTotalSupply, uint8(18), addr1, big.NewInt(0), int64(0))

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

	db.storageMap[contracts.AddressConsensusGroup] = make(map[types.Hash][]byte)
	consensusGroupKey, _ := types.BytesToHash(contracts.GetConsensusGroupKey(ledger.CommonGid))
	db.storageMap[contracts.AddressConsensusGroup][consensusGroupKey], _ = contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		uint8(0),
		helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize),
		uint8(0),
		helper.JoinBytes(helper.LeftPadBytes(new(big.Int).Mul(big.NewInt(1e6), attovPerVite).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(3600*24*90).Bytes(), helper.WordSize)),
		uint8(0),
		[]byte{},
		addr1,
		big.NewInt(0),
		timestamp+createConsensusGroupPledgeTime)
	db.storageMap[contracts.AddressPledge] = make(map[types.Hash][]byte)
	db.storageMap[contracts.AddressPledge][types.DataHash(addr1.Bytes())], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, big.NewInt(1e18))
	return
}
