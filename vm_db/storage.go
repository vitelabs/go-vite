package vm_db

import "github.com/pkg/errors"

func (vdb *vmDb) GetValue(key []byte) ([]byte, error) {
	if value, ok := vdb.unsaved().GetValue(key); ok {
		return value, nil
	}
	return vdb.GetOriginalValue(key)
}
func (vdb *vmDb) GetOriginalValue(key []byte) ([]byte, error) {
	return vdb.chain.GetValue(*vdb.address, key)
}

func (vdb *vmDb) SetValue(key []byte, value []byte) error {
	if len(key) > 32 {
		return errors.New("the length of key is not allowed to exceed 32 bytes")
	}
	vdb.unsaved().SetValue(key, value)
	return nil
}

func (vdb *vmDb) GetUnsavedStorage() [][2][]byte {
	return vdb.unsaved().GetStorage()
}
