package vm_db

func (vdb *vmDb) DebugGetStorage() (map[string][]byte, error) {
	result := make(map[string][]byte)
	iter, err := vdb.NewStorageIterator(nil)
	if err != nil {
		return nil, err
	}

	defer iter.Release()

	for iter.Next() {
		result[string(iter.Key())] = iter.Value()
	}
	if err := iter.Error(); err != nil {
		return nil, iter.Error()
	}

	return result, nil
}
