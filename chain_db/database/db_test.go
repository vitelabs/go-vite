package database

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common"
	"os"
	"path/filepath"
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
