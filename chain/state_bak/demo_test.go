package chain_state_bak

import (
	"encoding/binary"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/rand"
	"os"
	"os/user"
	"path"
	"testing"
)

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func BenchmarkDemoDB_Write(b *testing.B) {
	b.StopTimer()
	dir := path.Join(homeDir(), "demo_test")

	dDb, err := NewDemoDB(dir)
	if err != nil {
		b.Fatal(err)
	}

	keyMaxNum := uint64(1000)
	snapshotHeight := uint64(1000 * 10000)
	//count := 1000 * 10000
	//for i := 0; i < count; i++ {
	//	//for j := 0; j < 10; j++ {
	//	num := uint64(i) % keyMaxNum
	//	key := make([]byte, 8)
	//	value := make([]byte, 8)
	//
	//	binary.BigEndian.PutUint64(key, num)
	//	binary.BigEndian.PutUint64(value, num)
	//
	//	dDb.Write(key, value, snapshotHeight)
	//	//}
	//	snapshotHeight++
	//}
	//
	//if err := dDb.db.CompactRange(*util.BytesPrefix(nil)); err != nil {
	//	b.Fatal(err)
	//}

	keyBytes := make([]byte, 8)
	iter := dDb.db.NewIterator(util.BytesPrefix(nil), nil)
	defer iter.Release()

	fmt.Println("Start...")
	for i := 0; i < b.N; i++ {
		num := rand.Uint64() % keyMaxNum
		tmpSnapshotHeight := rand.Uint64() % snapshotHeight

		binary.BigEndian.PutUint64(keyBytes, num)

		key := ToInnerKey(keyBytes, tmpSnapshotHeight)
		b.StartTimer()
		iter.Seek(key)
		if ok := iter.Prev(); !ok {
			fmt.Println("no no no!")
		} else {
			iter.Value()
			//key := iter.Key()
			//fmt.Printf("%d (%d): %d (%d)\n",
			//	binary.BigEndian.Uint64(key[:8]), binary.BigEndian.Uint64(key[len(key)-8:]),
			//	binary.BigEndian.Uint64(iter.Value()), tmpSnapshotHeight)
		}

		b.StopTimer()

	}

	fmt.Println("Stop...")
	if err := dDb.Destory(); err != nil {
		b.Fatal(err)
	}
}
