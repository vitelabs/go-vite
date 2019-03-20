package consensus_db

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
)

const (
	INDEX_ElectionResult = byte(0)
)

type AddrArr []types.Address

func (self AddrArr) Bytes() []byte {
	arr := []types.Address(self)
	var result []byte
	for _, v := range arr {
		result = append(result, v.Bytes()...)
	}
	return result
}

func (self AddrArr) SetBytes(byt []byte) ([]types.Address, error) {
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

type ConsensusDB struct {
	db *leveldb.DB
}

func NewConsensusDBByDir(dir string) *ConsensusDB {
	db, err := database.NewLevelDb(dir)
	if err != nil {
		panic(err)
	}
	return NewConsensusDB(db)
}

func NewConsensusDB(db *leveldb.DB) *ConsensusDB {
	return &ConsensusDB{
		db: db,
	}
}

func (self *ConsensusDB) GetElectionResultByHash(hash types.Hash) ([]byte, error) {
	key, _ := database.EncodeKey(INDEX_ElectionResult, hash.Bytes())
	value, err := self.db.Get(key, nil)

	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return value, nil
}

func (self *ConsensusDB) StoreElectionResultByHash(hash types.Hash, data []byte) error {
	key, _ := database.EncodeKey(INDEX_ElectionResult, hash.Bytes())
	return self.db.Put(key, data, nil)
}

func (self *ConsensusDB) Check() {
	db := self.db
	key, err := database.EncodeKey(INDEX_ElectionResult)
	if err != nil {
		panic(err)
	}
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
