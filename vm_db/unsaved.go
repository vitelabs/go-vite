package vm_db

import (
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"math/rand"
)

type Unsaved struct {
	contractMetaMap map[types.Address]*ledger.ContractMeta

	code []byte

	logList ledger.VmLogList

	storage     *DB
	deletedKeys map[string]struct{}
	keys        map[string]struct{}

	storageDirty bool
	storageCache [][2][]byte

	balanceMap map[types.TokenTypeId]*big.Int
}

var rnd = rand.New(rand.NewSource(0xdeadbeef))

func NewUnsaved() *Unsaved {
	return &Unsaved{
		contractMetaMap: make(map[types.Address]*ledger.ContractMeta),
		logList:         make(ledger.VmLogList, 0),
		keys:            make(map[string]struct{}),
		deletedKeys:     make(map[string]struct{}),
		storage:         newMemDB(comparer.DefaultComparer, 0, rnd),
		storageDirty:    false,
		balanceMap:      make(map[types.TokenTypeId]*big.Int),
	}
}

func (unsaved *Unsaved) Reset() {
	unsaved.contractMetaMap = make(map[types.Address]*ledger.ContractMeta)
	unsaved.code = nil

	unsaved.logList = nil

	unsaved.storage.Reset()
	unsaved.deletedKeys = make(map[string]struct{})
	unsaved.storageDirty = false

	unsaved.storageCache = nil

	unsaved.balanceMap = make(map[types.TokenTypeId]*big.Int)
}

func (unsaved *Unsaved) GetStorage() [][2][]byte {
	if unsaved.storageDirty {
		iter := unsaved.storage.NewIterator(nil)
		defer iter.Release()

		unsaved.storageCache = make([][2][]byte, 0, len(unsaved.keys))
		for iter.Next() {
			unsaved.storageCache = append(unsaved.storageCache, [2][]byte{iter.Key(), iter.Value()})
		}
		unsaved.storageDirty = false
	}

	return unsaved.storageCache
}

func (unsaved *Unsaved) GetBalanceMap() map[types.TokenTypeId]*big.Int {
	return unsaved.balanceMap
}

func (unsaved *Unsaved) GetCode() []byte {
	return unsaved.code
}

func (unsaved *Unsaved) GetContractMeta(addr types.Address) *ledger.ContractMeta {
	return unsaved.contractMetaMap[addr]
}
func (unsaved *Unsaved) IsDelete(key []byte) bool {
	_, ok := unsaved.deletedKeys[string(key)]
	return ok
}

func (unsaved *Unsaved) SetValue(key []byte, value []byte) {
	unsaved.storageDirty = true

	keyStr := string(key)
	unsaved.keys[keyStr] = struct{}{}
	if len(value) <= 0 {
		unsaved.deletedKeys[keyStr] = struct{}{}
	} else if _, ok := unsaved.deletedKeys[keyStr]; ok {
		delete(unsaved.deletedKeys, keyStr)
	}

	unsaved.storage.Put(key, value)
}

func (unsaved *Unsaved) GetValue(key []byte) ([]byte, bool) {
	value, errNotFound := unsaved.storage.Get(key)
	if errNotFound != nil {
		if _, ok := unsaved.deletedKeys[string(key)]; ok {
			return nil, true
		}
		return nil, false
	}

	return value, true
}

func (unsaved *Unsaved) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, bool) {
	result, ok := unsaved.balanceMap[*tokenTypeId]
	return result, ok
}

func (unsaved *Unsaved) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	unsaved.balanceMap[*tokenTypeId] = amount
}

func (unsaved *Unsaved) AddLog(log *ledger.VmLog) {
	unsaved.logList = append(unsaved.logList, log)
}

func (unsaved *Unsaved) GetLogList() ledger.VmLogList {
	return unsaved.logList
}

func (unsaved *Unsaved) GetLogListHash() *types.Hash {
	return unsaved.logList.Hash()
}

func (unsaved *Unsaved) SetContractMeta(addr types.Address, contractMeta *ledger.ContractMeta) {
	unsaved.contractMetaMap[addr] = contractMeta
}

func (unsaved *Unsaved) SetCode(code []byte) {
	unsaved.code = code
}

func (unsaved *Unsaved) NewStorageIterator(prefix []byte) interfaces.StorageIterator {
	return unsaved.storage.NewIterator(util.BytesPrefix(prefix))
}

func (unsaved *Unsaved) ReleaseRuntime() {
	unsaved.GetStorage()
	unsaved.storage = nil
}
