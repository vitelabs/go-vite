package chain_db

import (
	"encoding/binary"
	"fmt"
	"github.com/allegro/bigcache"
	"github.com/patrickmn/go-cache"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkLevelDb(b *testing.B) {
	db := memdb.New(comparer.DefaultComparer, 0)
	benchmarkCache(b, db)
}

func BenchmarkBigCache(b *testing.B) {
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	//cache := memdb.New(comparer.DefaultComparer, 0)
	benchmarkCache(b, newBigCacheBenchCache(cache))
}

func BenchmarkGoCache(b *testing.B) {
	c := cache.New(5*time.Minute, 10*time.Minute)

	//cache := memdb.New(comparer.DefaultComparer, 0)
	benchmarkCache(b, newGoCacheBenchCache(c))
}

type BenchCache interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Size() int
}

type bigCacheBenchCache struct {
	cache *bigcache.BigCache
}

func newBigCacheBenchCache(db *bigcache.BigCache) BenchCache {
	return &bigCacheBenchCache{
		cache: db,
	}
}
func (c *bigCacheBenchCache) Put(key, value []byte) error {
	return c.cache.Set(string(key), value)
}

func (c *bigCacheBenchCache) Get(key []byte) ([]byte, error) {
	value, err := c.cache.Get(string(key))
	if err != bigcache.ErrEntryNotFound {
		return nil, err
	}
	return value, nil
}
func (c *bigCacheBenchCache) Delete(key []byte) error {
	return c.cache.Delete(string(key))
}

func (c *bigCacheBenchCache) Size() int {
	return c.cache.Len()
}

type goCacheBenchCache struct {
	cache *cache.Cache
}

func newGoCacheBenchCache(c *cache.Cache) BenchCache {
	return &goCacheBenchCache{
		cache: c,
	}
}
func (c *goCacheBenchCache) Put(key, value []byte) error {
	c.cache.Set(string(key), value, cache.DefaultExpiration)
	return nil
}

func (c *goCacheBenchCache) Get(key []byte) ([]byte, error) {
	value, ok := c.cache.Get(string(key))

	if !ok {
		return nil, nil
	}

	return value.([]byte), nil
}

func (c *goCacheBenchCache) Delete(key []byte) error {
	c.cache.Delete(string(key))
	return nil
}

func (c *goCacheBenchCache) Size() int {
	return c.cache.ItemCount()
}

func benchmarkCache(b *testing.B, cache BenchCache) {
	maxNum := uint64(1000 * 10000)
	b.Run("put", func(b *testing.B) {
		key := make([]byte, 24)
		value := make([]byte, 8)
		for i := 0; i < b.N; i++ {
			//random := uint64(i)
			random := rand.Uint64() % maxNum
			binary.BigEndian.PutUint64(key[:8], random)
			binary.BigEndian.PutUint64(key[8:], random)
			binary.BigEndian.PutUint64(key[16:], random)

			binary.BigEndian.PutUint64(value[:], random)

			if err := cache.Put(key, value); err != nil {
				panic(err)
			}
		}
	})
	fmt.Println(cache.Size())
	b.Run("get", func(b *testing.B) {
		key := make([]byte, 24)
		for i := 0; i < b.N; i++ {

			random := rand.Uint64() % maxNum
			binary.BigEndian.PutUint64(key[:8], random)
			binary.BigEndian.PutUint64(key[8:], random)
			binary.BigEndian.PutUint64(key[16:], random)

			if _, err := cache.Get(key); err != nil &&
				err != leveldb.ErrNotFound {
				panic(err)
			}
		}
	})

	deleteMaxNum := uint64(10000 * 10000)

	b.Run("delete", func(b *testing.B) {
		key := make([]byte, 24)
		for i := 0; i < b.N; i++ {

			random := rand.Uint64() % deleteMaxNum
			binary.BigEndian.PutUint64(key[:8], random)
			binary.BigEndian.PutUint64(key[8:], random)
			binary.BigEndian.PutUint64(key[16:], random)

			cache.Delete(key)
		}
	})
	fmt.Println(cache.Size())
}

//func BenchmarkLedisdb(b *testing.B) {
//	b.StopTimer()
//	cfg := lediscfg.NewConfigDefault()
//	cfg.DBName = "memory"
//	l, err := ledis.Open(cfg)
//	if err != nil {
//		panic(err)
//	}
//	cache, _ := l.Select(0)
//
//	b.StartTimer()
//	key := make([]byte, 24)
//	for i := 0; i < b.N; i++ {
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		value := chain_utils.Uint64ToBytes(uint64(i))
//
//		cache.Set(key, value)
//	}
//	b.StopTimer()
//
//	l.Close()
//
//}
//
//func BenchmarkBigCache(b *testing.B) {
//
//	b.StopTimer()
//	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
//
//	key := make([]byte, 24)
//
//	for i := uint64(1); i <= 100*10000; i++ {
//		random := rand.Uint64() % i
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		value := chain_utils.Uint64ToBytes(uint64(i))
//
//		cache.Put(key, value)
//	}
//	b.StartTimer()
//	for i := 0; i <= b.N; i++ {
//		random := rand.Uint64() % 100 * 10000
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		//value := chain_utils.Uint64ToBytes(uint64(i))
//
//		cache.Get(key)
//		//cache.Get(key)
//		//cache.Get(key)
//	}
//
//}
//

//
//func BenchmarkBoltDb(b *testing.B) {
//	b.StopTimer()
//
//	cache, err := buntdb.Open("data.cache")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	b.StartTimer()
//
//	key := make([]byte, 24)
//
//	for i := 0; i < b.N; i++ {
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		value := chain_utils.Uint64ToBytes(uint64(i))
//		cache.Update(func(tx *buntdb.Tx) error {
//
//			tx.Set(string(key), string(value), nil)
//			key[0] = 15
//			tx.Set(string(key), string(value), nil)
//			key[0] = 16
//			tx.Set(string(key), string(value), nil)
//			key[0] = 17
//			tx.Set(string(key), string(value), nil)
//			return nil
//		})
//	}
//	b.StopTimer()
//	cache.Close()
//}
//
//func BenchmarkMap(b *testing.B) {
//	b.StopTimer()
//
//	a := make(map[string][]byte, 100*10000)
//
//	key := make([]byte, 24)
//
//	for i := 0; i < 100*10000; i++ {
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		value := chain_utils.Uint64ToBytes(uint64(i))
//		a[string(key)] = value
//	}
//
//	b.StartTimer()
//
//	var ok bool
//	for i := 0; i < b.N; i++ {
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		//value := chain_utils.Uint64ToBytes(uint64(i))
//		_, ok = a[string(key)]
//	}
//	b.StopTimer()
//	fmt.Println(ok)
//
//}
//
//func BenchmarkSkipList(b *testing.B) {
//	b.StopTimer()
//
//	s := skiplist.NewStringMap()
//	b.StartTimer()
//
//	key := make([]byte, 24)
//
//	for i := 0; i < b.N; i++ {
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		value := chain_utils.Uint64ToBytes(uint64(i))
//		s.Set(string(key), value)
//	}
//	b.StopTimer()
//}
//
//func BenchmarkTreeSet(b *testing.B) {
//	b.StopTimer()
//	//
//	//s := treeset.NewWithStringComparator()
//	//s := linkedhashset.New()
//	//m := treemap.NewWithStringComparator()
//	//m := linkedhashmap.New()
//	//m := redblacktree.NewWithStringComparator()
//	//m := avltree.NewWithStringComparator()
//	//m := btree.NewWithStringComparator(5)
//	m := binaryheap.NewWithStringComparator()
//	b.StartTimer()
//
//	for i := 0; i < b.N; i++ {
//		key := make([]byte, 24)
//
//		random := rand.Uint64()
//		binary.BigEndian.PutUint64(key[:8], random)
//		binary.BigEndian.PutUint64(key[8:], random)
//		binary.BigEndian.PutUint64(key[16:], random)
//
//		//value := chain_utils.Uint64ToBytes(uint64(i))
//		//copy(key[24:], value)
//		m.Push(string(key))
//		//s.Add()
//	}
//	fmt.Println(m.Size())
//	b.StopTimer()
//}
//
//type MockMem struct {
//	slice [][]byte
//}
//
//func NewMockMem() *MockMem {
//	return &MockMem{}
//}
//
//func (mem *MockMem) Put(key []byte) {
//	mem.slice = append(mem.slice, key)
//}
