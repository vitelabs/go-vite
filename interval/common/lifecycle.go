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

func (ls *LifecycleStatus) PreInit() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, Origin, PreInit) {
		return ls
	}
	panic(ls.panicFailMsg("PreInit"))
}
func (ls *LifecycleStatus) PostInit() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, PreInit, PostInit) {
		return ls
	}
	panic(ls.panicFailMsg("PostInit"))
}
func (ls *LifecycleStatus) PreStart() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, PostInit, PreStart) {
		return ls
	}
	panic(ls.panicFailMsg("PreStart"))
}
func (ls *LifecycleStatus) PostStart() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, PreStart, PostStart) {
		return ls
	}
	panic(ls.panicFailMsg("PostStart"))
}
func (ls *LifecycleStatus) PreStop() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, PostStart, PreStop) {
		return ls
	}
	panic(ls.panicFailMsg("PreStop"))
}
func (ls *LifecycleStatus) PostStop() *LifecycleStatus {
	if atomic.CompareAndSwapInt32(&ls.S, PreStop, PostStop) {
		return ls
	}
	panic(ls.panicFailMsg("PostStop"))
}

func (ls *LifecycleStatus) Stopped() bool {
	return ls.S == 6
}
func (ls *LifecycleStatus) Status() int32 {
	return ls.S
}

func (ls *LifecycleStatus) panicFailMsg(prefix string) string {
	return prefix + " fail. status:" + strconv.Itoa(int(ls.S))
}

type Lifecycle interface {
	Init()
	Start()
	Stop()
	Status() int32
}
