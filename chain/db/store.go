package chain_db

import (
	"errors"

	"github.com/vitelabs/go-vite/common/db"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/memdb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/interfaces"
	"os"
	"sync"
)

type Store struct {
	id   types.Hash
	name string

	memDbMu sync.RWMutex
	memDb   *db.MemDB

	snapshotBatch *leveldb.Batch
	flushingBatch *leveldb.Batch

	unconfirmedBatchs *UnconfirmedBatchs

	dbDir string
	db    *leveldb.DB

	afterRecoverFuncs []func()
}

func NewStore(dataDir string, name string) (*Store, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte(name)))

	diskStore, err := leveldb.OpenFile(dataDir, nil)

	if err != nil {
		return nil, err
	}

	store := &Store{
		id:    id,
		name:  name,
		memDb: db.NewMemDB(),

		unconfirmedBatchs: NewUnconfirmedBatchs(),

		dbDir: dataDir,
		db:    diskStore,
	}

	store.snapshotBatch = store.getNewBatch()

	return store, nil
}

func (store *Store) CompactRange(r util.Range) error {
	return store.db.CompactRange(r)
}

func (store *Store) NewBatch() *leveldb.Batch {
	return new(leveldb.Batch)
}

func (store *Store) Get(key []byte) ([]byte, error) {
	mdb, seq := store.getSnapshotMemDb()

	value, err := store.db.Get2(key, nil, mdb, seq)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return value, nil
}

func (store *Store) GetOriginal(key []byte) ([]byte, error) {
	mdb, seq := store.getSnapshotMemDb()
	return store.db.Get2(key, nil, mdb, seq)
}

func (store *Store) Has(key []byte) (bool, error) {
	mdb, seq := store.getSnapshotMemDb()

	_, err := store.db.Get2(key, nil, mdb, seq)

	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (store *Store) NewIterator(slice *util.Range) interfaces.StorageIterator {
	mdb, seq := store.getSnapshotMemDb()

	return store.db.NewIterator2(slice, nil, mdb, seq)
}

func (store *Store) Close() error {
	store.memDb = nil
	return store.db.Close()
}

func (store *Store) Clean() error {
	if err := store.Close(); err != nil {
		return err
	}

	if err := os.RemoveAll(store.dbDir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + store.dbDir + " failed, error is " + err.Error())
	}

	store.db = nil

	return nil
}

func (store *Store) RegisterAfterRecover(f func()) {
	store.afterRecoverFuncs = append(store.afterRecoverFuncs, f)
}

func (store *Store) getSnapshotMemDb() (*memdb.DB, uint64) {
	store.memDbMu.RLock()
	mdb := store.memDb.GetDb()
	seq := store.memDb.GetSeq()
	store.memDbMu.RUnlock()

	return mdb, seq
}

func (store *Store) putMemDb(batch *leveldb.Batch) {
	batch.Replay(store.memDb)
}
