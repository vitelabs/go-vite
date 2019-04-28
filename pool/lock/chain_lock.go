package lock

import "sync"

type ChainRollback interface {
	RLockRollback()
	RUnLockRollback()
	LockRollback()
	UnLockRollback()
}

type ChainInsert interface {
	RLockInsert()
	RUnLockInsert()
	LockInsert()
	UnLockInsert()
}

type EasyImpl struct {
	chainInsertRw   sync.RWMutex
	chainRollbackRw sync.RWMutex
}

func (self *EasyImpl) RLockInsert() {
	self.chainInsertRw.RLock()
}

func (self *EasyImpl) RUnLockInsert() {
	self.chainInsertRw.RUnlock()
}

func (self *EasyImpl) LockInsert() {
	self.chainInsertRw.Lock()
}

func (self *EasyImpl) UnLockInsert() {
	self.chainInsertRw.Unlock()
}

func (self *EasyImpl) RLockRollback() {
	self.chainRollbackRw.RLock()
}

func (self *EasyImpl) RUnLockRollback() {
	self.chainRollbackRw.RUnlock()
}

func (self *EasyImpl) LockRollback() {
	self.chainRollbackRw.Lock()
}

func (self *EasyImpl) UnLockRollback() {
	self.chainRollbackRw.Unlock()
}
