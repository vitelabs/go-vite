package vm_db

import "github.com/pkg/errors"

func (db *vmDb) GetValue(key []byte) ([]byte, error) {
	if value, ok := db.unsaved.GetValue(key); ok {
		return value, nil
	}
	return db.GetOriginalValue(key)
}
func (db *vmDb) GetOriginalValue(key []byte) ([]byte, error) {
	return db.chain.GetValue(db.address, key)
}

func (db *vmDb) SetValue(key []byte, value []byte) error {
	if len(key) > 32 {
		return errors.New("the length of key is not allowed to exceed 32 bytes")
	}
	db.unsaved.SetValue(key, value)
	return nil
}

func (db *vmDb) GetUnsavedStorage() [][2][]byte {
	return db.unsaved.GetStorage()
}
