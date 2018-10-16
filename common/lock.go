package common

import (
	"sync/atomic"
	"time"
)

type NonBlockLock struct {
	b int32
}

func (self *NonBlockLock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 0, 1)
}

func (self *NonBlockLock) Lock() {
	i := 0
	for {
		if atomic.CompareAndSwapInt32(&self.b, 0, 1) {
			return
		}
		i++
		if i > 2000 {
			time.Sleep(time.Millisecond)
			i = 0
		}
	}
}

func (self *NonBlockLock) UnLock() bool {
	return atomic.CompareAndSwapInt32(&self.b, 1, 0)
}
