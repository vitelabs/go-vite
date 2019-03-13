package vm_db

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
)

func (db *vmDB) getPrevStateSnapshot() (StateSnapshot, error) {
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

func (db *vmDB) GetReceiptHash() *types.Hash {
	return nil
}

func (db *vmDB) Reset() {

}
