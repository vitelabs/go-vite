package vm_db

func (db *vmDb) GetValue(key []byte) ([]byte, error) {
	if value, ok := db.unsaved.GetValue(key); ok {
		return value, nil
	}
	return db.GetOriginalValue(key)
}
func (db *vmDb) GetOriginalValue(key []byte) ([]byte, error) {
	return db.chain.GetValue(db.address, key)
}

func (db *vmDb) SetValue(key []byte, value []byte) {
	db.unsaved.SetValue(key, value)
}

func (db *vmDb) GetUnsavedStorage() ([][2][]byte, map[string]struct{}) {
	return db.unsaved.GetStorage()
}
