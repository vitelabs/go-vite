package vm_context

type StorageIterator struct{}

func (*StorageIterator) Next() (key, value []byte, ok bool) {
	return nil, nil, false
}
