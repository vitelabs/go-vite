package chain_db

import (
	"encoding/binary"
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/storage"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("storeTest")))

	store, err := NewStore(path.Join(test_tools.DefaultDataDir(), "test_store"), id)
	if err != nil {
		t.Fatal(err)
	}
	writeBlock(store, 1)
	queryBlock(store, 1)

	flushToDisk(store)

	writeBlock(store, 2)
	queryBlock(store, 2)

	snapshot(store, []int{1})

	queryBlock(store, 1)
	queryBlock(store, 2)

	writeBlock(store, 3)
	queryBlock(store, 3)

	writeBlock(store, 4)
	queryBlock(store, 4)

	snapshot(store, []int{3})

	queryBlock(store, 1)
	queryBlock(store, 2)
	queryBlock(store, 3)
	queryBlock(store, 4)

}

func TestSeekToLastAndPrev(t *testing.T) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("storeTest")))

	store, err := NewStore(path.Join(test_tools.DefaultDataDir(), "test_store"), id)
	if err != nil {
		t.Fatal(err)
	}

	batch := new(leveldb.Batch)

	end := uint64(10)
	for i := uint64(0); i < end; i++ {
		writeKv(batch, i, i)
	}

	store.WriteDirectly(batch)

	iter := store.NewIterator(nil)
	if iter.Seek(chain_utils.Uint64ToBytes(end + 100)) {
		t.Fatal("error")
	}
	if !iter.Prev() {
		t.Fatal("error")
	}

	if binary.BigEndian.Uint64(iter.Value()) != end-1 {
		t.Fatal("error")
	}
}

func TestPutAndDelete(t *testing.T) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("storeTest")))

	store, err := NewStore(path.Join(test_tools.DefaultDataDir(), "test_store"), id)
	if err != nil {
		t.Fatal(err)
	}

	batch := new(leveldb.Batch)

	end := uint64(10)
	for i := uint64(0); i < end; i++ {
		writeKv(batch, i, i)
	}

	store.WriteDirectly(batch)

	flushToDisk(store)
	batch.Delete(chain_utils.Uint64ToBytes(7))

	batch.Put(chain_utils.Uint64ToBytes(7), chain_utils.Uint64ToBytes(77))
	batch.Delete(chain_utils.Uint64ToBytes(7))

	store.WriteDirectly(batch)
	value, err := store.Get(chain_utils.Uint64ToBytes(7))
	fmt.Println(value)

	iter := store.NewIterator(nil)
	for iter.Next() {
		fmt.Printf("%d: %d\n", iter.Key(), iter.Value())
	}

}

func TestIterator(t *testing.T) {
	id, err := types.BytesToHash(crypto.Hash256([]byte("123")))
	if err != nil {
		panic(err)
	}
	store, err := NewStore("test_mem", id)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	count := uint64(0)
	go func() {
		defer wg.Done()

		for {
			batch := new(leveldb.Batch)
			count++
			batch.Put(chain_utils.Uint64ToBytes(count), chain_utils.Uint64ToBytes(count))
			store.WriteDirectly(batch)

			random := rand.Intn(100)
			if random > 50 {
				flushToDisk(store)
			}
			time.Sleep(10 * time.Millisecond)
		}

	}()
	go func() {
		defer wg.Done()

		for {
			iter := store.NewIterator(nil)
			prev := uint64(0)
			for iter.Next() {
				current := chain_utils.BytesToUint64(iter.Key())
				if prev+1 != current {
					fmt.Printf("error, %d, %d\n", prev, current)
				}
				prev = current
			}

			fmt.Println("check ", prev)
		}

	}()
	wg.Wait()

}

func TestIterator2(t *testing.T) {
	a, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d", uint64(1)<<8)
	a.Put(chain_utils.Uint64ToBytes(1), chain_utils.Uint64ToBytes(1), nil)
	iter := a.NewIterator(&util.Range{Start: chain_utils.Uint64ToBytes(1), Limit: chain_utils.Uint64ToBytes(helper.MaxUint64)}, nil)
	iter.Last()
}

func TestIterator3(t *testing.T) {
	id, err := types.BytesToHash(crypto.Hash256([]byte("123")))
	if err != nil {
		panic(err)
	}
	os.RemoveAll("test_mem")
	store, err := NewStore("test_mem", id)
	if err != nil {
		panic(err)
	}

	//batch2 := new(leveldb.Batch)
	//
	//for i := 0; i <= 12; i++ {
	//	batch2.Put(chain_utils.Uint64ToBytes(uint64(i%3)), chain_utils.Uint64ToBytes(uint64(i)))
	//}
	//store.WriteDirectly(batch2)

	//store.memDb = db.NewMemDB()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			batch := new(leveldb.Batch)

			for i := 0; i <= 12; i++ {
				batch.Put(chain_utils.Uint64ToBytes(uint64(i)), chain_utils.Uint64ToBytes(uint64(i)))
			}

			store.WriteDirectly(batch)
			flushToDisk(store)
		}
	}()
	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		for {
			iter := store.NewIterator(&util.Range{Start: chain_utils.Uint64ToBytes(0), Limit: chain_utils.Uint64ToBytes(helper.MaxUint64)})
			if iter.Last() {
				fmt.Println(iter.Key(), iter.Value())
			}
			iter.Release()
		}

	}()

	wg.Wait()

}

func writeKv(batch *leveldb.Batch, key uint64, value uint64) {
	batch.Put(chain_utils.Uint64ToBytes(key), chain_utils.Uint64ToBytes(value))
}

func flushToDisk(store *Store) {
	// mock flush
	store.Prepare()

	// redo log
	store.RedoLog()

	// commit
	store.Commit()

	// after commit
	store.AfterCommit()
}

func writeBlock(store *Store, blockIndex int) {
	batch := store.NewBatch()
	baseNum := uint64(blockIndex * 3)
	batch.Put(chain_utils.Uint64ToBytes(1+baseNum), chain_utils.Uint64ToBytes(1+baseNum))
	batch.Put(chain_utils.Uint64ToBytes(2+baseNum), chain_utils.Uint64ToBytes(2+baseNum))
	batch.Put(chain_utils.Uint64ToBytes(3+baseNum), chain_utils.Uint64ToBytes(3+baseNum))

	randNum := rand.Intn(3) + 1
	for i := 1; i < randNum; i++ {
		batch.Delete(chain_utils.Uint64ToBytes(uint64(i) + baseNum))
	}

	blockHash, _ := types.BytesToHash(crypto.Hash256([]byte(fmt.Sprintf("blockHash%d", blockIndex))))
	store.WriteAccountBlockByHash(batch, blockHash)
}

func snapshot(store *Store, blockIndexList []int) {
	batch := store.NewBatch()
	blockHashList := make([]types.Hash, len(blockIndexList))
	for _, blockIndex := range blockIndexList {
		blockHash, _ := types.BytesToHash(crypto.Hash256([]byte(fmt.Sprintf("blockHash%d", blockIndex))))

		blockHashList = append(blockHashList, blockHash)
	}

	store.WriteSnapshotByHash(batch, blockHashList)
}

func queryBlock(store *Store, blockIndex int) {
	baseNum := uint64(blockIndex * 3)

	fmt.Println("blockIndex", blockIndex)
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(1 + baseNum)))
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(2 + baseNum)))
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(3 + baseNum)))

}

func Test_map(t *testing.T) {
	a := make(map[uint64]uint64)
	a[1] = 2
	a[2] = 3
	delete(a, 1)
	delete(a, 2)
	a[4] = 3

}
