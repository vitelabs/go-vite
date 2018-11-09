package database

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSameBatch(t *testing.T) {
	dbDir := filepath.Join(common.GoViteTestDataDir(), "db_test")
	os.RemoveAll(dbDir)
	db, err := NewLevelDb(dbDir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100000; i++ {
		batch := new(leveldb.Batch)

		key := []byte("hahaKey")
		value := []byte("hahaValue")
		batch.Put(key, value)

		value2 := []byte("hahaValueValue")
		batch.Put(key, value2)
		db.Write(batch, nil)

		v, _ := db.Get(key, nil)
		if !bytes.Equal(v, value2) {
			t.Error(fmt.Sprintf("%s", v))
		}

	}

	os.RemoveAll(dbDir)
}

type testAbc struct {
	count uint64
}

func TestWriteMeta(t *testing.T) {
	dbDir := filepath.Join(common.GoViteTestDataDir(), "db_test")
	os.RemoveAll(dbDir)
	db, err := NewLevelDb(dbDir)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}

	count := uint64(0)
	key1 := []byte("k1")
	var lock sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		for count < 1000000 {
			lock.Lock()
			valueByte, _ := db.Get(key1, nil)

			// read tmpCount
			var tmpCount uint64
			if len(valueByte) > 0 {
				tmpCount = binary.BigEndian.Uint64(valueByte)
			}

			if tmpCount != count {
				t.Fatal("error!!!")

			}

			// set writeByte
			writeByte := make([]byte, 8)
			binary.BigEndian.PutUint64(writeByte, tmpCount+1)

			batch := new(leveldb.Batch)
			batch.Put(key1, writeByte)
			batch.Put(key1, writeByte)
			batch.Put(key1, writeByte)
			db.Write(batch, nil)
			count++
			lock.Unlock()
		}

	}()

	//count2 := uint64(0)
	//key2 := []byte("k2")

	wg.Add(1)
	go func() {
		defer wg.Done()
		for count < 1000000 {
			lock.Lock()

			valueByte, _ := db.Get(key1, nil)
			var tmpCount uint64
			if len(valueByte) > 0 {
				tmpCount = binary.BigEndian.Uint64(valueByte)
			}
			if tmpCount != count {
				t.Fatal("error!!!")

			}

			writeByte := make([]byte, 8)
			binary.BigEndian.PutUint64(writeByte, tmpCount+1)

			batch := new(leveldb.Batch)
			batch.Put(key1, writeByte)
			batch.Put(key1, writeByte)
			batch.Put(key1, writeByte)
			batch.Put(key1, writeByte)
			db.Write(batch, nil)
			count++
			lock.Unlock()
		}

	}()

	//go func() {
	//	defer wg.Done()
	//	key2 := []byte("k" + strconv.FormatUint(2, 10))
	//	value2 := []byte("v" + strconv.FormatUint(2, 10))
	//
	//	db.Put(key2, value2, nil)
	//}()

	wg.Wait()
	os.RemoveAll(dbDir)
}
