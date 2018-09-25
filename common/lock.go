package common

import "sync/atomic"

type NonBlockLock struct {
	b int32
}

func (self *NonBlockLock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 0, 1)
}

func (self *NonBlockLock) UnLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 1, 0)
}
