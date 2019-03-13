package vm_db

func (db *vmDB) GetValue(key []byte) ([]byte, error) {
	if value, ok := db.unsaved.GetValue(key); ok {
		return value, nil
	}
	return db.GetOriginalValue(key)
}
func (db *vmDB) GetOriginalValue(key []byte) ([]byte, error) {
	prevStateSnapshot, err := db.getPrevStateSnapshot()
	if err != nil {
		return nil, err
	}

	return prevStateSnapshot.GetValue(key)
}

func (db *vmDB) SetValue(key []byte, value []byte) {
	db.unsaved.SetValue(key, value)
}

func (db *vmDB) DeleteValue(key []byte) {
	db.unsaved.SetValue(key, []byte{})
}

func (db *vmDB) NewStorageIterator(prefix []byte) StorageIterator {
	return nil
}
