package common

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

var TimeoutErr = errors.New("timeout")

type TimeoutCond struct {
	cd        *sync.Cond
	notifyNum uint32
}

func NewTimeoutCond() *TimeoutCond {
	mutex := &sync.Mutex{}
	return &TimeoutCond{cd: sync.NewCond(mutex)}
}

func (self *TimeoutCond) Wait() {
	old := atomic.SwapUint32(&self.notifyNum, 0)
	if old > 0 {
		return
	}
	self.cd.L.Lock()
	defer self.cd.L.Unlock()
	self.cd.Wait()
}

func (self *TimeoutCond) WaitTimeout(t time.Duration) error {
	old := atomic.SwapUint32(&self.notifyNum, 0)
	if old > 0 {
		fmt.Printf("t:%s, num:%d\n", time.Now(), old)
		return nil
	}
	done := make(chan struct{})
	go func() {
		self.Wait()
		close(done)
	}()
	select {
	case <-time.After(t):
		// timed out
		return TimeoutErr
	case <-done:
		// Wait returned
		return nil
	}
}

func (self *TimeoutCond) Broadcast() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.cd.Broadcast()
}
func (self *TimeoutCond) Signal() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.cd.Signal()
}
