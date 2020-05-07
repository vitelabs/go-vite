package common

import (
	"sync/atomic"
)

type NonBlockLock struct {
	b int32
}

func (lock *NonBlockLock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&lock.b, 0, 1)
}
func (lock *NonBlockLock) UnLock() bool {
	return atomic.CompareAndSwapInt32(&lock.b, 1, 0)
}

type RWLock struct {
	r int32
	w int32
}

func (lock *RWLock) TryRLock() bool {
	for {
		i := lock.r
		if i < 0 {
			// write event happen.
			return false
		}
		j := i + 1
		if atomic.CompareAndSwapInt32(&lock.r, i, j) {
			return true
		}
	}
}
func (lock *RWLock) UnRLock() bool {
	for {
		i := lock.r
		if i < 0 {
			// write event happen.
			return false
		}
		j := i - 1
		if atomic.CompareAndSwapInt32(&lock.r, i, j) {
			return true
		}
	}
}

func (lock *RWLock) TryWLock() bool {
	if lock.r > 0 {
		return false
	}
	// have no read event happen.
	return atomic.CompareAndSwapInt32(&lock.r, 0, -1)
}

func (lock *RWLock) WLock() bool {
	for {
		if atomic.CompareAndSwapInt32(&lock.r, 0, -1) {
			return true
		}
	}
}
func (lock *RWLock) UnWLock() bool {
	return atomic.CompareAndSwapInt32(&lock.w, -1, 0)
}
