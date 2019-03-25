package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

func (db *vmDb) GetReceiptHash() *types.Hash {
	kvList := db.unsaved.GetStorage()
	if len(kvList) <= 0 {
		return nil
	}

	size := 0
	for _, kv := range kvList {
		size += len(kv[0]) + len(kv[1])
	}

	hashSource := make([]byte, 0, size)
	for _, kv := range kvList {
		hashSource = append(hashSource, kv[0]...)
		hashSource = append(hashSource, kv[1]...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(hashSource))

	return &hash
}

func (db *vmDb) Reset() {
	db.unsaved.Reset()
}

func (db *vmDb) Finish() {
	db.unsaved.ReleaseRuntime()
}
