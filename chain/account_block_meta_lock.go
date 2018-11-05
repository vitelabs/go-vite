package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type abmLocker struct {
	doing sync.Map
}

func newAbmLocker() *abmLocker {
	return &abmLocker{}
}

func (locker *abmLocker) Lock(blockHash types.Hash) {
	keyLock, _ := locker.doing.LoadOrStore(blockHash, &sync.Mutex{})

	keyLock.(*sync.Mutex).Lock()
}

func (locker *abmLocker) Unlock(blockHash types.Hash) {
	keyLock, ok := locker.doing.Load(blockHash)

	if !ok {
		panic("abmLocker unlock of unlocked mutext")
	}
	keyLock.(*sync.Mutex).Unlock()
}
