package vmctxt_interface

type StorageIterator interface {
	Next() (key, value []byte, ok bool)
}
