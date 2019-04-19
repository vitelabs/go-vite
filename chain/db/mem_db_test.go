package chain_db

import (
	"encoding/binary"
	"fmt"
	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/ryszard/goskiplist/skiplist"
	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/tidwall/buntdb"
	"github.com/vitelabs/go-vite/chain/utils"
	"log"
	"math/rand"
	"testing"
)

func BenchmarkLedisdb(b *testing.B) {
	b.StopTimer()
	cfg := lediscfg.NewConfigDefault()
	cfg.DBName = "memory"
	l, err := ledis.Open(cfg)
	if err != nil {
		panic(err)
	}
	db, _ := l.Select(0)

	b.StartTimer()
	key := make([]byte, 24)
	for i := 0; i < b.N; i++ {

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		value := chain_utils.Uint64ToBytes(uint64(i))

		db.Set(key, value)
	}
	b.StopTimer()

	l.Close()

}

func BenchmarkLevelDb(b *testing.B) {
	b.StopTimer()
	db := memdb.New(comparer.DefaultComparer, 0)

	key := make([]byte, 24)

	for i := uint64(1); i <= 100*10000; i++ {
		random := rand.Uint64() % i
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		value := chain_utils.Uint64ToBytes(uint64(i))

		db.Put(key, value)
	}
	b.StartTimer()
	for i := 0; i <= b.N; i++ {
		random := rand.Uint64() % 100 * 10000
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		//value := chain_utils.Uint64ToBytes(uint64(i))

		db.Get(key)
		//db.Get(key)
		//db.Get(key)
	}

}

func BenchmarkBoltDb(b *testing.B) {
	b.StopTimer()

	db, err := buntdb.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}

	b.StartTimer()

	key := make([]byte, 24)

	for i := 0; i < b.N; i++ {

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		value := chain_utils.Uint64ToBytes(uint64(i))
		db.Update(func(tx *buntdb.Tx) error {

			tx.Set(string(key), string(value), nil)
			key[0] = 15
			tx.Set(string(key), string(value), nil)
			key[0] = 16
			tx.Set(string(key), string(value), nil)
			key[0] = 17
			tx.Set(string(key), string(value), nil)
			return nil
		})
	}
	b.StopTimer()
	db.Close()
}

func BenchmarkMap(b *testing.B) {
	b.StopTimer()

	a := make(map[string][]byte, 100*10000)

	key := make([]byte, 24)

	for i := 0; i < 100*10000; i++ {

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		value := chain_utils.Uint64ToBytes(uint64(i))
		a[string(key)] = value
	}

	b.StartTimer()

	var ok bool
	for i := 0; i < b.N; i++ {

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		//value := chain_utils.Uint64ToBytes(uint64(i))
		_, ok = a[string(key)]
	}
	b.StopTimer()
	fmt.Println(ok)

}

func BenchmarkSkipList(b *testing.B) {
	b.StopTimer()

	s := skiplist.NewStringMap()
	b.StartTimer()

	key := make([]byte, 24)

	for i := 0; i < b.N; i++ {

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		value := chain_utils.Uint64ToBytes(uint64(i))
		s.Set(string(key), value)
	}
	b.StopTimer()
}

func BenchmarkTreeSet(b *testing.B) {
	b.StopTimer()
	//
	//s := treeset.NewWithStringComparator()
	//s := linkedhashset.New()
	//m := treemap.NewWithStringComparator()
	//m := linkedhashmap.New()
	//m := redblacktree.NewWithStringComparator()
	//m := avltree.NewWithStringComparator()
	//m := btree.NewWithStringComparator(5)
	m := binaryheap.NewWithStringComparator()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := make([]byte, 24)

		random := rand.Uint64()
		binary.BigEndian.PutUint64(key[:8], random)
		binary.BigEndian.PutUint64(key[8:], random)
		binary.BigEndian.PutUint64(key[16:], random)

		//value := chain_utils.Uint64ToBytes(uint64(i))
		//copy(key[24:], value)
		m.Push(string(key))
		//s.Add()
	}
	fmt.Println(m.Size())
	b.StopTimer()
}

type MockMem struct {
	slice [][]byte
}

func NewMockMem() *MockMem {
	return &MockMem{}
}

func (mem *MockMem) Put(key []byte) {
	mem.slice = append(mem.slice, key)
}
