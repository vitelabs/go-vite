package vm_db

func (db *vmDB) GetValue(key []byte) []byte {
	return nil
}
func (db *vmDB) GetOriginalValue(key []byte) []byte {
	return nil
}

func (db *vmDB) SetValue(key []byte, value []byte) {

}
func (db *vmDB) DeleteValue(key []byte) {

}

func (db *vmDB) NewStorageIterator(prefix []byte) StorageIterator {
	return nil
}
