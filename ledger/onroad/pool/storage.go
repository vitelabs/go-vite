package onroad_pool

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/v2/common/types"
	chain_utils "github.com/vitelabs/go-vite/v2/ledger/chain/utils"
)

type OnroadTx struct {
	FromAddr types.Address
	ToAddr   types.Address

	FromHeight uint64 // if the block is r-s block, the height means receive block height.
	FromHash   types.Hash
	FromIndex  *uint32
}

func (tx OnroadTx) String() string {

	ii := int64(-1)
	if tx.FromIndex != nil {
		tmp := *tx.FromIndex
		ii = int64(tmp)
	}
	return fmt.Sprintf("fromAddr=%s,toAddr=%s,fromHeight=%d,fromHash=%s,fromIndex=%d",
		tx.FromAddr, tx.ToAddr, tx.FromHeight, tx.FromHash, ii)
}

type OnRoadDB interface {
	InsertOnroad(tx OnroadTx) error
	DeleteOnroad(tx OnroadTx) error
}

type onroadStorage struct {
	db *leveldb.DB

	mu      sync.RWMutex
	callers sync.Map
}

func newOnroadStorage(db *leveldb.DB) *onroadStorage {
	return &onroadStorage{
		db: db,
		mu: sync.RWMutex{},
	}
}

func (storage *onroadStorage) addCaller(caller types.Address) error {
	storage.callers.Store(caller, 0)
	return nil
}
func (storage *onroadStorage) removeCaller(caller types.Address) {
	storage.callers.Delete(caller)
}

func (storage *onroadStorage) insertOnRoadTx(tx OnroadTx) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	// fmt.Println("add on road tx", tx.String())

	storage.addCaller(tx.FromAddr)
	// batch.Put(tx.toOnroadHeightKey().Bytes(), tx.toOnroadHeightValue())
	return storage.db.Put(tx.toOnroadHeightKey().Bytes(), tx.toOnroadHeightValue(), nil)
}

// @todo opt removeCaller
func (storage *onroadStorage) deleteOnRoadTx(tx OnroadTx) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	// fmt.Println("remove on road tx", tx.String())

	err := storage.db.Delete(tx.toOnroadHeightKey().Bytes(), nil)
	if err != nil {
		return fmt.Errorf("delete onroad tx err:%s", err.Error())
	}
	txs, err := storage.getFirstOnroadTx(tx.ToAddr, tx.FromAddr)
	if err != nil {
		return err
	}
	if len(txs) == 0 {
		storage.removeCaller(tx.FromAddr)
	}
	return nil
}

func (storage *onroadStorage) updateFromIndex(tx OnroadTx) (bool, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	key := tx.toOnroadHeightKey().Bytes()
	exist, err := storage.db.Has(key, nil)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
	}
	return true, storage.db.Put(tx.toOnroadHeightKey().Bytes(), tx.toOnroadHeightValue(), nil)
}

func (storage *onroadStorage) GetAllFirstOnroadTx(addr types.Address) (map[types.Address][]OnroadTx, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	result := make(map[types.Address][]OnroadTx)
	var resultErr error

	storage.callers.Range(func(key interface{}, val interface{}) bool {
		caller := key.(types.Address)
		txs, err := storage.getFirstOnroadTx(addr, caller)
		if err != nil {
			resultErr = err
			return false
		}
		if len(txs) == 0 {
			return true
		}
		result[caller] = txs
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	return result, nil
}

func (storage *onroadStorage) getFirstOnroadTx(addr types.Address, caller types.Address) ([]OnroadTx, error) {
	key := chain_utils.NewOnRoadHeightKey()
	iter := storage.db.NewIterator(util.BytesPrefix(key.IteratorPrefix(addr, caller)), nil)
	defer iter.Release()

	result := make([]OnroadTx, 0)
	initHeight := uint64(0)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		tx, err := newOnroadTxFromBytes(key, value)
		if err != nil {
			return nil, err
		}
		if initHeight == 0 {
			initHeight = tx.FromHeight
		}
		if tx.FromHeight != initHeight {
			break
		}
		result = append(result, *tx)
	}
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	// if len(result) > 0 {
	// 	fmt.Println("get First onroad tx", addr, caller, result)
	// }
	return result, nil
}

func (storage *onroadStorage) GetFirstOnroadTx(addr types.Address, caller types.Address) ([]OnroadTx, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	return storage.getFirstOnroadTx(addr, caller)
}

func (tx OnroadTx) toOnroadHeightKey() chain_utils.OnRoadHeightKey {
	return chain_utils.CreateOnRoadAddressHeightKey(tx.ToAddr, tx.FromAddr, tx.FromHeight, tx.FromHash)
}

func (tx OnroadTx) toOnroadHeightValue() []byte {
	if tx.FromIndex == nil {
		return []byte{}
	} else {
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, *tx.FromIndex)
		return bs
	}
}

func newOnroadTxFromBytes(key []byte, value []byte) (*OnroadTx, error) {
	toAddress, err := types.BytesToAddress(key[1 : 1+types.AddressSize])
	if err != nil {
		return nil, err
	}

	fromAddress, err := types.BytesToAddress(key[1+types.AddressSize : 1+types.AddressSize+types.AddressSize])
	if err != nil {
		return nil, err
	}

	height := binary.BigEndian.Uint64(key[1+types.AddressSize+types.AddressSize : 1+types.AddressSize+types.AddressSize+types.HeightSize])

	hash, err := types.BytesToHash(key[1+types.AddressSize+types.AddressSize+types.HeightSize : 1+types.AddressSize+types.AddressSize+types.HeightSize+types.HashSize])
	if err != nil {
		return nil, err
	}
	result := &OnroadTx{
		FromAddr:   fromAddress,
		ToAddr:     toAddress,
		FromHeight: height,
		FromHash:   hash,
	}
	if len(value) == 4 {
		fromIndex := binary.BigEndian.Uint32(value)
		result.FromIndex = &fromIndex
	}
	return result, nil
}

func newOnroadTxFromOrHashHeight(fromAddr types.Address, toAddress types.Address, or orHashHeight) OnroadTx {
	return OnroadTx{
		FromAddr:   fromAddr,
		ToAddr:     toAddress,
		FromHeight: or.Height,
		FromHash:   or.Hash,
		FromIndex:  or.SubIndex,
	}
}
