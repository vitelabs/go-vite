package consensus

import (
	"fmt"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/consensus_db"
)

type Cache interface {
	// Add adds a value to the cache.  Returns true if an eviction occurred.
	Add(key, value interface{}) (evicted bool)

	// Get looks up a key's value from the cache.
	Get(key interface{}) (value interface{}, ok bool)
}

type DbCache struct {
	db *consensus_db.ConsensusDB
}

//func (self *DbCache) Get(hashes interface{}) (interface{}, bool) {
//	bytes, err := self.db.GetElectionResultByHash(hashes.(types.Hash))
//	if err != nil {
//		return nil, false
//	}
//	if bytes == nil {
//		return nil, false
//	}
//
//	var result []types.Address
//	err = json.Unmarshal(bytes, &result)
//	if err != nil {
//		fmt.Printf("parse fail. %s, %s\n", err.Error(), string(bytes))
//		return nil, false
//	}
//	return result, true
//}
//
//func (self *DbCache) Add(hashes interface{}, addrArr interface{}) bool {
//	bytes, err := json.Marshal(addrArr)
//	if err != nil {
//		return false
//	}
//	fmt.Printf("store %s, %+v\n", hashes, addrArr)
//	self.db.StoreElectionResultByHash(hashes.(types.Hash), bytes)
//	return false
//}

func (self *DbCache) Get(hashes interface{}) (interface{}, bool) {
	bytes, err := self.db.GetElectionResultByHash(hashes.(types.Hash))
	if err != nil {
		return nil, false
	}
	if bytes == nil {
		return nil, false
	}

	var result consensus_db.AddrArr
	result, err = result.SetBytes(bytes)
	if err != nil {
		fmt.Printf("parse fail. %s, %s\n", err.Error(), string(bytes))
		return nil, false
	}
	return result, true
}

func (self *DbCache) Add(hashes interface{}, addrArr interface{}) bool {
	arrs := addrArr.([]types.Address)
	bytes := consensus_db.AddrArr(arrs).Bytes()
	fmt.Printf("store %s, %+v\n", hashes, addrArr)
	self.db.StoreElectionResultByHash(hashes.(types.Hash), bytes)
	return false
}
