package vm_db

func (db *vmDb) DebugGetStorage() (map[string][]byte, error) {
	result := make(map[string][]byte)
	iter := db.NewStorageIterator(nil)
	defer iter.Release()

	for iter.Next() {
		result[string(iter.Key())] = iter.Value()
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return result, nil
}
