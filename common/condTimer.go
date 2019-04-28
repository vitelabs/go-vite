package common

import (
	"sync"
	"sync/atomic"
	"time"
)

type CondTimer struct {
	cd        *sync.Cond
	notifyNum uint32
	closed    chan struct{}
}

func NewCondTimer() *CondTimer {
	mutex := &sync.Mutex{}
	return &CondTimer{cd: sync.NewCond(mutex), closed: make(chan struct{})}
}

func (self *CondTimer) Wait() {
	old := atomic.SwapUint32(&self.notifyNum, 0)
	if old > 0 {
		return
	}
	self.cd.L.Lock()
	defer self.cd.L.Unlock()
	self.cd.Wait()
}

func (self *CondTimer) Broadcast() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.cd.Broadcast()
}

func (self *CondTimer) Signal() {
	atomic.AddUint32(&self.notifyNum, 1)
	self.cd.Signal()
}

func (self *CondTimer) Start(t time.Duration) {
	go func() {
		ticker := time.NewTicker(t)
		defer ticker.Stop()
		for {
			select {
			case <-self.closed:
				return
			case <-ticker.C:
				self.cd.Broadcast()
			}
		}
	}()
}

func (self *CondTimer) Stop() {
	close(self.closed)
	self.Broadcast()
}
