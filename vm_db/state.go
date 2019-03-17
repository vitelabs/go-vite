package vm_db

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/interfaces"
)

func (db *vmDb) getPrevStateSnapshot() (interfaces.StateSnapshot, error) {
	if db.prevStateSnapshot == nil {
		if db.prevAccountBlockHash == nil {
			return nil, errors.New("no context, db.prevAccountBlockHash is nil")
		}

		var err error
		if db.prevStateSnapshot, err = db.chain.GetStateSnapshot(db.prevAccountBlockHash); err != nil {
			return nil, errors.New(fmt.Sprintf("db.chain.GetStateSnapshot failed, error is %s, prevAccountBlockHash is %s", err, db.prevAccountBlockHash))
		} else if db.prevStateSnapshot == nil {
			return nil, errors.New(fmt.Sprintf("the returned stateSnapshot of db.chain.GetStateSnapshot is nil, prevAccountBlockHash is %s", db.prevAccountBlockHash))
		}
	}
	return db.prevStateSnapshot, nil
}

func (db *vmDb) GetReceiptHash() *types.Hash {
	sortedKVList := db.unsaved.GetSortedStorage()

	size := 0
	for _, kv := range sortedKVList {
		size += len(kv[0]) + len(kv[1])
	}

	hashSource := make([]byte, 0, size)
	for _, kv := range sortedKVList {
		hashSource = append(hashSource, kv[0]...)
		hashSource = append(hashSource, kv[1]...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(hashSource))

	return &hash
}

func (db *vmDb) Reset() {
	db.unsaved = NewUnsaved()
}
