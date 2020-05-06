package common

import (
	"sync/atomic"
)

type NonBlockLock struct {
	b int32
}

func (self *NonBlockLock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 0, 1)
}
func (self *NonBlockLock) UnLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 1, 0)
}

type RWLock struct {
	r int32
	w int32
}

func (self *RWLock) TryRLock() bool {
	for {
		i := self.r
		if i < 0 {
			// write event happen.
			return false
		}
		j := i + 1
		if atomic.CompareAndSwapInt32(&self.r, i, j) {
			return true
		}
	}
}
func (self *RWLock) UnRLock() bool {
	for {
		i := self.r
		if i < 0 {
			// write event happen.
			return false
		}
		j := i - 1
		if atomic.CompareAndSwapInt32(&self.r, i, j) {
			return true
		}
	}
}

func (self *RWLock) TryWLock() bool {
	if self.r > 0 {
		return false
	}
	// have no read event happen.
	return atomic.CompareAndSwapInt32(&self.r, 0, -1)
}

func (self *RWLock) WLock() bool {
	for {
		if atomic.CompareAndSwapInt32(&self.r, 0, -1) {
			return true
		}
	}
}
func (self *RWLock) UnWLock() bool {
	return atomic.CompareAndSwapInt32(&self.w, -1, 0)
}
