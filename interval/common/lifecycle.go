package common

import (
	"strconv"
	"sync/atomic"
)

// BlockType types.
const (
	Origin    = 0
	PreInit   = 1
	PostInit  = 2
	PreStart  = 3
	PostStart = 4
	PreStop   = 5
	PostStop  = 6
)

type LifecycleStatus struct {
	S int32 // 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped
}

func (self *LifecycleStatus) PreInit() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, Origin, PreInit) {
		return self
	}
	panic(self.panicFailMsg("PreInit"))
}
func (self *LifecycleStatus) PostInit() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, PreInit, PostInit) {
		return self
	}
	panic(self.panicFailMsg("PostInit"))
}
func (self *LifecycleStatus) PreStart() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, PostInit, PreStart) {
		return self
	}
	panic(self.panicFailMsg("PreStart"))
}
func (self *LifecycleStatus) PostStart() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, PreStart, PostStart) {
		return self
	}
	panic(self.panicFailMsg("PostStart"))
}
func (self *LifecycleStatus) PreStop() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, PostStart, PreStop) {
		return self
	}
	panic(self.panicFailMsg("PreStop"))
}
func (self *LifecycleStatus) PostStop() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&self.S, PreStop, PostStop) {
		return self
	}
	panic(self.panicFailMsg("PostStop"))
}

func (self *LifecycleStatus) Stopped() bool {
	return self.S == 6
}
func (self *LifecycleStatus) Status() int32 {
	return self.S
}

func (self *LifecycleStatus) panicFailMsg(prefix string) string {
	return prefix + " fail. status:" + strconv.Itoa(int(self.S))
}

type Lifecycle interface {
	Init()
	Start()
	Stop()
	Status() int32
}
