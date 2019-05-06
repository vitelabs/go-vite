package chain_flusher

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"path"
	"sync"
	"testing"
)

func TestFlusher(t *testing.T) {
	t.Run("TestFlushMultiTimes", func(t *testing.T) {
		TestFlushMultiTimes(t)
	})

	t.Run("TestRedoLogFailed", func(t *testing.T) {
		TestRedoLogFailed(t)
	})

	t.Run("TestCommitFailed", func(t *testing.T) {
		TestCommitFailed(t)
	})

	t.Run("TestPatchRedoFailed", func(t *testing.T) {
		TestPatchRedoFailed(t)
	})

	t.Run("TestRecover", func(t *testing.T) {
		TestRecover(t)
	})
}

func TestFlushMultiTimes(t *testing.T) {
	if err := checkFlusher(100, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRedoLogFailed(t *testing.T) {
	if err := checkFlusher(1, nil, &StorageOptions{RedoLogFailed: true}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommitFailed(t *testing.T) {
	if err := checkFlusher(1, nil, &StorageOptions{CommitFailed: true}, nil); err != nil {
		t.Fatal(err)
	}

}

func TestPatchRedoFailed(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			return
		}
		t.Fatal("error")
	}()
	if err := checkFlusher(1, nil, &StorageOptions{CommitFailed: true, PatchRedoFailed: true}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRecover(t *testing.T) {
	if err := checkRecover(100, nil, &StorageOptions{CommitFailed: true, PatchRedoFailed: true}, nil); err != nil {
		t.Fatal(err)
	}

}

func checkRecover(times int, opts ...*StorageOptions) error {
	flusher, _, dbList := initFlusher(opts)
	i := uint64(0)
	defer func() {
		if err := recover(); err != nil {
			// check db
			for _, db := range dbList {
				checkDB(db, i, false)
			}

			// recover
			flusher.Recover()

			// check db
			for _, db := range dbList {
				checkDB(db, i, true)
			}
			return
		}
		panic("err")
	}()
	for ; i < uint64(times); i++ {
		// flush
		flusher.flush()

	}

	return nil

}

func checkFlusher(times int, opts ...*StorageOptions) error {
	flusher, storeList, dbList := initFlusher(opts)
	writeToDB := true
	for _, store := range storeList {
		currentOpts := store.(*MockStorage).Opts()
		if writeToDB {
			writeToDB = !currentOpts.RedoLogFailed
		}
	}

	for i := uint64(0); i < uint64(times); i++ {
		flusher.flush()
		//if canCommit {
		for _, db := range dbList {
			if err := checkDB(db, i, writeToDB); err != nil {
				return err
			}
		}
	}

	return nil
}

func initFlusher(opts []*StorageOptions) (*Flusher, []Storage, []*mockDB) {
	optsLen := len(opts)
	dbList := make([]*mockDB, 0, optsLen)
	storeList := make([]Storage, 0, optsLen)

	for index, opt := range opts {
		br := newMockDB()

		dbList = append(dbList, br)
		storeList = append(storeList, newMockStorage(fmt.Sprintf("mock storage %d", index), opt, br))

	}
	var mu sync.RWMutex
	flusher, err := NewFlusher(storeList, &mu, path.Join(test_tools.DefaultDataDir(), "test_flusher"))
	if err != nil {
		panic(err)
	}
	return flusher, storeList, dbList
}

func checkDB(db *mockDB, commitTimes uint64, writeToDB bool) error {
	Kv := db.Kv
	if writeToDB {
		if uint64(len(Kv)) != 15*(commitTimes+1) {
			return errors.New(fmt.Sprintf("len(kv) is %d, 15*(commitTimes+1) is %d", len(Kv), 15*(commitTimes+1)))
		}
	}
	baseNum := commitTimes * 30
	for key, value := range db.Kv {
		keyNum := chain_utils.BytesToUint64([]byte(key))
		valueNum := chain_utils.BytesToUint64(value)
		if writeToDB {
			if (keyNum > 10+baseNum && keyNum < 20+baseNum) || keyNum > 25+baseNum {
				return errors.New(fmt.Sprintf("keyNum is %d, commitTimes is %d", keyNum, commitTimes))
			}
			if (valueNum > 10+baseNum && valueNum < 20+baseNum) || valueNum > 25+baseNum {
				return errors.New(fmt.Sprintf("valueNum is %d, commitTimes is %d", valueNum, commitTimes))
			}
		} else {
			if keyNum >= baseNum || valueNum >= baseNum {
				return errors.New(fmt.Sprintf("baseNum is %d, keyNum is %d, valueNum is %d", baseNum, keyNum, valueNum))
			}
		}

	}
	return nil
}

type StorageOptions struct {
	RedoLogFailed bool
	CommitFailed  bool

	PatchRedoFailed bool
}

type MockStorage struct {
	opts *StorageOptions
	id   types.Hash

	batch *leveldb.Batch
	db    *mockDB

	commitTimes uint64
}

func newMockStorage(name string, opt *StorageOptions, db *mockDB) *MockStorage {
	id, _ := types.BytesToHash(crypto.Hash256([]byte(name)))
	if opt == nil {
		opt = &StorageOptions{}
	}
	if db == nil {
		db = newMockDB()
	}
	return &MockStorage{
		opts: opt,
		id:   id,
		db:   db,
	}
}

func (ms *MockStorage) Opts() *StorageOptions {
	return ms.opts
}

func (ms *MockStorage) Id() types.Hash {
	return ms.id
}

func (ms *MockStorage) BaseNum() uint64 {
	return ms.commitTimes * 30
}

func (ms *MockStorage) Prepare() {
	ms.batch = new(leveldb.Batch)
	baseNum := ms.BaseNum()
	for i := uint64(0 + baseNum); i < baseNum+10; i++ {
		ms.batch.Put(chain_utils.Uint64ToBytes(i), chain_utils.Uint64ToBytes(i))
	}

	for i := uint64(10 + baseNum); i < baseNum+20; i++ {
		ms.batch.Delete(chain_utils.Uint64ToBytes(i))
	}

	for i := uint64(20 + baseNum); i < baseNum+30; i++ {
		ms.batch.Put(chain_utils.Uint64ToBytes(i), chain_utils.Uint64ToBytes(i))
	}

	for i := uint64(25 + baseNum); i < baseNum+35; i++ {
		ms.batch.Delete(chain_utils.Uint64ToBytes(i))
	}
}

func (ms *MockStorage) CancelPrepare() {
	ms.batch.Reset()
}

func (ms *MockStorage) RedoLog() ([]byte, error) {
	if ms.opts.RedoLogFailed {
		return []byte("3456789"), errors.New("redoLog failed")
	}
	return ms.batch.Dump(), nil
}

func (ms *MockStorage) Commit() error {
	ms.commitTimes++
	if ms.opts.CommitFailed {
		return errors.New("commit failed")
	}
	ms.batch.Replay(ms.db)
	return nil
}

func (ms *MockStorage) PatchRedoLog(data []byte) error {
	if ms.opts.PatchRedoFailed {
		return errors.New("patch redo failed")
	}
	batch := new(leveldb.Batch)
	if err := batch.Load(data); err != nil {
		return err
	}
	ms.batch.Replay(ms.db)
	return nil
}

func (ms *MockStorage) AfterCommit() {
	return
}

func (ms *MockStorage) BeforeRecover([]byte) {
	return
}

func (ms *MockStorage) AfterRecover() {
	return
}

type mockDB struct {
	Kv map[string][]byte
}

func newMockDB() *mockDB {
	return &mockDB{
		Kv: make(map[string][]byte),
	}
}

func (replay *mockDB) Put(key, value []byte) {
	replay.Kv[string(key)] = value
}

func (replay *mockDB) Delete(key []byte) {
	delete(replay.Kv, string(key))
}
