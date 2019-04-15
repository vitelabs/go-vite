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

func checkFlusher(times int, opts ...*StorageOptions) error {
	optsLen := len(opts)
	brList := make([]*mockBatchReplay, 0, optsLen)
	storeList := make([]Storage, 0, optsLen)

	canStartCommit := true
	for index, opt := range opts {
		br := newMockBatchReplay()

		brList = append(brList, br)
		storeList = append(storeList, newMockStorage(fmt.Sprintf("mock storage %d", index), opt, br))

		if opt != nil && opt.RedoLogFailed {
			canStartCommit = false
		}
	}

	flusher, err := NewFlusher(storeList, path.Join(test_tools.DefaultDataDir(), "test_flusher"))
	if err != nil {
		return err
	}

	for i := uint64(0); i < uint64(times); i++ {
		flusher.flush()
		if !canStartCommit {
			for _, br := range brList {
				if len(br.Kv) > 0 {
					return errors.New("len(br.Storage) > 0")
				}
			}

			//for _, store := range storeList {
			//	store.
			//}
		} else {
			for _, br := range brList {
				if err := checkBatchReplay(br, i); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkBatchReplay(batchReplay *mockBatchReplay, commitTimes uint64) error {
	Kv := batchReplay.Kv
	if uint64(len(Kv)) != 15*(commitTimes+1) {
		return errors.New(fmt.Sprintf("len(kv) is %d, 15*(commitTimes+1) is %d", len(Kv), 15*(commitTimes+1)))
	}
	baseNum := commitTimes * 30
	for key, value := range batchReplay.Kv {
		keyNum := chain_utils.BytesToUint64([]byte(key))
		if (keyNum > 10+baseNum && keyNum < 20+baseNum) || keyNum > 25+baseNum {
			return errors.New(fmt.Sprintf("keyNum is %d, commitTimes is %d", keyNum, commitTimes))
		}
		valueNum := chain_utils.BytesToUint64(value)
		if (valueNum > 10+baseNum && valueNum < 20+baseNum) || valueNum > 25+baseNum {
			return errors.New(fmt.Sprintf("valueNum is %d, commitTimes is %d", valueNum, commitTimes))
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
	opt *StorageOptions
	id  types.Hash

	batch       *leveldb.Batch
	batchReplay *mockBatchReplay

	commitTimes uint64
}

func newMockStorage(name string, opt *StorageOptions, batchReplay *mockBatchReplay) *MockStorage {
	id, _ := types.BytesToHash(crypto.Hash256([]byte(name)))
	if opt == nil {
		opt = &StorageOptions{}
	}
	if batchReplay == nil {
		batchReplay = newMockBatchReplay()
	}
	return &MockStorage{
		opt:         opt,
		id:          id,
		batchReplay: batchReplay,
	}
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
	if ms.opt.RedoLogFailed {
		return []byte("3456789"), errors.New("redoLog failed")
	}
	return ms.batch.Dump(), nil
}

func (ms *MockStorage) Commit() error {
	ms.commitTimes++
	if ms.opt.CommitFailed {
		return errors.New("commit failed")
	}
	ms.batch.Replay(ms.batchReplay)
	return nil
}

func (ms *MockStorage) PatchRedoLog(data []byte) error {
	if ms.opt.PatchRedoFailed {
		return errors.New("patch redo failed")
	}
	batch := new(leveldb.Batch)
	if err := batch.Load(data); err != nil {
		return err
	}
	ms.batch.Replay(ms.batchReplay)
	return nil

}

type mockBatchReplay struct {
	Kv map[string][]byte
}

func newMockBatchReplay() *mockBatchReplay {
	return &mockBatchReplay{
		Kv: make(map[string][]byte),
	}
}

func (replay *mockBatchReplay) Put(key, value []byte) {
	replay.Kv[string(key)] = value
}

func (replay *mockBatchReplay) Delete(key []byte) {
	delete(replay.Kv, string(key))
}
