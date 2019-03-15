package vm_db

import "github.com/vitelabs/go-vite/interfaces"

func (db *vmDb) NewStorageIterator(prefix []byte) interfaces.StorageIterator {
	return nil
}

type storageInterator struct {
}

func NewStorageInterator() {

}

func (iter *storageInterator) Next() {

}

func (iter *storageInterator) Prev() {

}

func (iter *storageInterator) Key() {

}

func (iter *storageInterator) Value() {

}

func (iter *storageInterator) Error() {

}
