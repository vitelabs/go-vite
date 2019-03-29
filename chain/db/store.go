package chain_db

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/interfaces"
	"os"
	"sync"
	"time"
)

type Store struct {
	memDb *MemDB

	dbDir string
	db    *leveldb.DB

	flushInterval time.Duration
	wg            sync.WaitGroup

	stopped chan struct{}
}

func NewStore(dataDir string, flushInterval time.Duration) (*Store, error) {
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	db, err := leveldb.OpenFile(dataDir, nil)

	if err != nil {
		return nil, err
	}

	store := &Store{
		flushInterval: flushInterval,
		memDb:         NewMemDB(),

		dbDir: dataDir,
		db:    db,
	}
	//
	//store.wg.Add(1)
	//go func() {
	//	defer store.wg.Done()
	//	for {
	//		select {
	//		case <-store.stopped:
	//			store.Flush()
	//			return
	//		default:
	//			store.Flush()
	//			time.Sleep(flushInterval)
	//		}
	//
	//	}
	//
	//}()
	return store, nil
}

func (store *Store) Put(key, value []byte) {
	store.db.Put(key, value, nil)
}

func (store *Store) Delete(key []byte) {
	store.db.Delete(key, nil)
}

func (store *Store) Get(key []byte) ([]byte, error) {
	value, ok := store.memDb.Get(key)
	if !ok {
		var err error
		value, err = store.db.Get(key, nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
	}

	return value, nil
}

func (store *Store) Has(key []byte) (bool, error) {
	if ok, deleted := store.memDb.Has(key); ok {
		return ok, nil

	} else if deleted {
		return false, nil

	}

	return store.db.Has(key, nil)
}

func (store *Store) HasPrefix(prefix []byte) (bool, error) {
	if ok := store.memDb.HasByPrefix(prefix); ok {
		return ok, nil
	}

	iter := store.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	result := false
	for iter.Next() {
		key := iter.Key()
		if ok := store.memDb.IsDelete(key); !ok {
			result = true
			break
		}
	}

	if err := iter.Error(); err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return result, nil
}

//func (store *Store) Flush() error {
//	store.memDb.Lock()
//	defer store.memDb.Unlock()
//
//	batch := new(leveldb.Batch)
//
//	store.memDb.Flush(batch)
//	if err := store.db.Write(batch, nil); err != nil {
//		return err
//	}
//
//	store.memDb.Clean()
//
//	return nil
//}

func (store *Store) NewIterator(slice *util.Range) interfaces.StorageIterator {
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		store.memDb.NewIterator(slice),
		store.db.NewIterator(slice, nil),
	}, store.memDb.IsDelete)
}

func (store *Store) Close() error {
	close(store.stopped)
	store.wg.Wait()

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

//func (store *Store) Put(key, value []byte) {
//	store.memDb.put(key, value)
//}
//
//func (store *Store) Delete(key []byte) {
//	store.memDb.Delete(key)
//}
//
//func (store *Store) Get(key []byte) ([]byte, error) {
//	value, ok := store.memDb.Get(key)
//	if !ok {
//		var err error
//		value, err = store.db.Get(key, nil)
//		if err != nil {
//			if err == leveldb.ErrNotFound {
//				return nil, nil
//			}
//			return nil, err
//		}
//	}
//
//	return value, nil
//}
//
//func (store *Store) Has(key []byte) (bool, error) {
//	if ok, deleted := store.memDb.Has(key); ok {
//		return ok, nil
//
//	} else if deleted {
//		return false, nil
//
//	}
//
//	return store.db.Has(key, nil)
//}
//
//func (store *Store) HasPrefix(prefix []byte) (bool, error) {
//	if ok := store.memDb.HasByPrefix(prefix); ok {
//		return ok, nil
//	}
//
//	iter := store.db.NewIterator(util.BytesPrefix(prefix), nil)
//	defer iter.Release()
//
//	result := false
//	for iter.Next() {
//		key := iter.Key()
//		if ok := store.memDb.IsDelete(key); !ok {
//			result = true
//			break
//		}
//	}
//
//	if err := iter.Error(); err != nil {
//		if err == leveldb.ErrNotFound {
//			return false, nil
//		}
//		return false, err
//	}
//	return result, nil
//}
//
//func (store *Store) Flush() error {
//	store.memDb.Lock()
//	defer store.memDb.Unlock()
//
//	batch := new(leveldb.Batch)
//
//	store.memDb.Flush(batch)
//	if err := store.db.Write(batch, nil); err != nil {
//		return err
//	}
//
//	store.memDb.Clean()
//
//	return nil
//}
//
//func (store *Store) NewIterator(slice *util.Range) interfaces.StorageIterator {
//	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
//		store.memDb.NewIterator(slice),
//		store.db.NewIterator(slice, nil),
//	}, store.memDb.IsDelete)
//}
//
//func (store *Store) Close() error {
//	close(store.stopped)
//	store.wg.Wait()
//
//	store.memDb = nil
//	return store.db.Close()
//}
//
//func (store *Store) Clean() error {
//	if err := store.Close(); err != nil {
//		return err
//	}
//
//	if err := os.RemoveAll(store.dbDir); err != nil && err != os.ErrNotExist {
//		return errors.New("Remove " + store.dbDir + " failed, error is " + err.Error())
//	}
//
//	store.db = nil
//
//	return nil
//}
