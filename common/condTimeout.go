package common

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

var TimeoutErr = errors.New("timeout")

type TimeoutCond struct {
	notifyNum uint32
	L         sync.Locker
	signal    chan uint8
}

func NewTimeoutCond() *TimeoutCond {
	mutex := &sync.Mutex{}
	return &TimeoutCond{L: mutex, signal: make(chan uint8, 0)}
}

func (self *TimeoutCond) Wait() {
	old := atomic.SwapUint32(&self.notifyNum, 0)
	if old > 0 {
		return
	}
	ch := self.signal
	select {
	case <-ch:
		return
	}
}

func (self *TimeoutCond) WaitTimeout(t time.Duration) error {
	old := atomic.SwapUint32(&self.notifyNum, 0)
	if old > 0 {
		return nil
	}
	ch := self.signal
	select {
	case <-ch:
		return nil
	case <-time.After(t):
		//fmt.Println(time.Now())
		return TimeoutErr
	}
}

func (self *TimeoutCond) Broadcast() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.L.Lock()
	defer self.L.Unlock()
	close(self.signal)
	self.signal = make(chan uint8, 0)
}
func (self *TimeoutCond) Signal() {
	atomic.AddUint32(&self.notifyNum, 1)
	select {
	case self.signal <- uint8(1):
	default:
	}
}
