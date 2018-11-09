package common

import (
	"sync/atomic"
)

type LifecycleStatus struct {
	Status int32 // 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped
}

func (self *LifecycleStatus) PreInit() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 0, 1)
}
func (self *LifecycleStatus) PostInit() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 1, 2)
}
func (self *LifecycleStatus) PreStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 2, 3)
}
func (self *LifecycleStatus) PostStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 3, 4)
}
func (self *LifecycleStatus) PreStop() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 4, 5)
}
func (self *LifecycleStatus) PostStop() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 5, 6)
}

func (self *LifecycleStatus) Stopped() bool {
	return self.Status == 6 || self.Status == 5
}
func (self *LifecycleStatus) GetStatus() int32 {
	return self.Status
}

type Lifecycle interface {
	Init()
	Start()
	Stop()
	GetStatus() int32
}
