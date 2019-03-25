package common

import (
	"sync"
	"time"

	"github.com/go-errors/errors"
)

var TimeoutErr = errors.New("timeout")

type TimeoutCond struct {
	cd sync.Cond
}

func (self *TimeoutCond) Wait() {
	self.cd.Wait()
}

func (self *TimeoutCond) WaitTimeout(t time.Duration) error {
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
	self.cd.Broadcast()
}
