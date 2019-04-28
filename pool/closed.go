package pool

import "sync/atomic"

type Closed interface {
	Close()
	IsClosed() bool
}

func NewClosed() Closed {
	return &atomicClosed{}
}

type atomicClosed struct {
	val uint32
}

func (self *atomicClosed) Close() {
	atomic.StoreUint32(&self.val, 1)
}

func (self *atomicClosed) IsClosed() bool {
	val := atomic.LoadUint32(&self.val)
	return val == 1
}
