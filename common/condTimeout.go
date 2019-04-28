package common

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

//
//import (
//	"fmt"
//	"sync"
//	"sync/atomic"
//	"time"
//
//	"github.com/pkg/errors"
//)
//
//var TimeoutErr = errors.New("timeout")
//
//type TimeoutCond struct {
//	cd        *sync.Cond
//	notifyNum uint32
//}
//
//func NewTimeoutCond() *TimeoutCond {
//	mutex := &sync.Mutex{}
//	return &TimeoutCond{cd: sync.NewCond(mutex)}
//}
//
//func (self *TimeoutCond) Wait() {
//	old := atomic.SwapUint32(&self.notifyNum, 0)
//	if old > 0 {
//		return
//	}
//	self.cd.L.Lock()
//	defer self.cd.L.Unlock()
//	self.cd.Wait()
//}
//
//func (self *TimeoutCond) WaitTimeout(t time.Duration) error {
//	old := atomic.SwapUint32(&self.notifyNum, 0)
//	if old > 0 {
//		fmt.Printf("t:%s, num:%d\n", time.Now(), old)
//		return nil
//	}
//	done := make(chan struct{})
//	go func() {
//		self.Wait()
//		close(done)
//	}()
//	select {
//	case <-time.After(t):
//		// timed out
//		return TimeoutErr
//	case <-done:
//		// Wait returned
//		return nil
//	}
//}
//
//func (self *TimeoutCond) Broadcast() {
//	atomic.AddUint32(&self.notifyNum, 1)
//	self.cd.Broadcast()
//}
//func (self *TimeoutCond) Signal() {
//	atomic.AddUint32(&self.notifyNum, 1)
//	self.cd.Signal()
//}

var TimeoutErr = errors.New("timeout")

type TimeoutCond struct {
	notifyNum uint32
	L         sync.Locker
	signal    chan uint8
}

func NewTimeoutCond() *TimeoutCond {
	mutex := &sync.Mutex{}
	return &TimeoutCond{L: mutex, signal: make(chan uint8)}
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
	old := self.signal
	self.signal = make(chan uint8)
	close(old)
}
func (self *TimeoutCond) Signal() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.L.Lock()
	defer self.L.Unlock()
	select {
	case self.signal <- uint8(1):
	default:
	}
}
