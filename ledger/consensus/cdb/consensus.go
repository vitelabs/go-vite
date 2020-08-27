package cdb

import (
	"encoding/binary"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
)

const (
	// IndexElectionResult is store prefix for election result
	IndexElectionResult = byte(0)
	// IndexPointPeriod is store prefix for period sbp info
	IndexPointPeriod = byte(1)
	// IndexPointHour is store prefix for hour sbp info
	IndexPointHour = byte(2)
	// IndexPointDay is store prefix for day sbp info
	IndexPointDay = byte(3)
)

// AddrArr is slice of types.Address
type AddrArr []types.Address

// Bytes implement AddrArr to byte slice
func (addrs AddrArr) Bytes() []byte {
	arr := []types.Address(addrs)
	var result []byte
	for _, v := range arr {
		result = append(result, v.Bytes()...)
	}
	return result
}

// SetBytes implement byte slice to types.Address slice
func (addrs AddrArr) SetBytes(byt []byte) ([]types.Address, error) {
	size := len(byt) / types.AddressSize
	result := make([]types.Address, size)
	for i := 0; i < size; i++ {
		addr, err := types.BytesToAddress(byt[i*types.AddressSize : (i+1)*types.AddressSize])
		if err != nil {
			return nil, err
		}
		result[i] = addr
	}
	arr := AddrArr(result)
	return arr, nil
}

// ConsensusDB is leveldb for
type ConsensusDB struct {
	db *leveldb.DB
}

func NewConsensusDB(db *leveldb.DB) *ConsensusDB {
	return &ConsensusDB{
		db: db,
	}
}

func (self *ConsensusDB) GetPointByHeight(prefix byte, height uint64) (*Point, error) {
	key := CreatePointKey(prefix, height)
	value, err := self.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	result := &Point{}
	err = result.Unmarshal(value)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (self *ConsensusDB) DeletePointByHeight(prefix byte, height uint64) error {
	key := CreatePointKey(prefix, height)
	return self.db.Delete(key, nil)
}

func (self *ConsensusDB) StorePointByHeight(prefix byte, height uint64, p *Point) error {
	key := CreatePointKey(prefix, height)
	byt, err := p.Marshal()
	if err != nil {
		return err
	}
	return self.db.Put(key, byt, nil)
}

func (self *ConsensusDB) GetElectionResultByHash(hash types.Hash) ([]types.Address, error) {
	key := CreateElectionResultKey(hash)
	value, err := self.db.Get(key, nil)

	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	var result AddrArr
	result, err = result.SetBytes(value)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse fail. %s, %s\n", err.Error(), string(value)))
	}
	return []types.Address(result), nil
}

func (self *ConsensusDB) DeleteElectionResultByHash(hash types.Hash) error {
	key := CreateElectionResultKey(hash)
	err := self.db.Delete(key, nil)
	return err
}

func (self *ConsensusDB) StoreElectionResultByHash(hash types.Hash, addrArr []types.Address) error {
	data := AddrArr(addrArr).Bytes()
	key := CreateElectionResultKey(hash)
	return self.db.Put(key, data, nil)
}

func (self *ConsensusDB) Check() {
	db := self.db
	key := CreateElectionResultPrefixKey()
	iter := db.NewIterator(util.BytesPrefix(key), nil)
	i := uint64(0)
	for ; iter.Next(); i++ {
		bytes := iter.Key()
		//value := iter.Value()
		hash, err := types.BytesToHash(bytes[1:])
		if err != nil {
			panic(err)
		}
		fmt.Println(hash)
	}
}

func CreateElectionResultPrefixKey() []byte {
	key := make([]byte, 1)
	key[0] = IndexElectionResult
	return key
}

func CreateElectionResultKey(hash types.Hash) []byte {
	key := make([]byte, 1+types.HashSize)
	key[0] = IndexElectionResult

	copy(key[1:types.HashSize+1], hash.Bytes())

	return key
}

func CreatePointKey(prefix byte, height uint64) []byte {
	key := make([]byte, 1+8)
	key[0] = prefix

	binary.BigEndian.PutUint64(key[1:9], height)
	return key
}
