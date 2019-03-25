package vm_db

import (
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/interfaces"
)

func (db *vmDb) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	iter, err := db.chain.GetStateIterator(db.address, prefix)
	if err != nil {
		return nil, err
	}

	unsavedIter := db.unsaved.NewStorageIterator(prefix)
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		unsavedIter,
		iter,
	}, db.unsaved.IsDelete), nil
}
