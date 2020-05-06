package tools

import "sync"

type ForkWall interface {
	Wait()
	Add(delta int)
	Done()
}

type ViteForkWall struct {
	rw sync.RWMutex
}


func (self *ViteForkWall) Lock() {
	self.rw.Lock()
}

func (self *ViteForkWall) UnLock(delta int) {
	self.rw.Unlock()
}

func (self *ViteForkWall) RLock() {
	self.rw.RLock()
}

func (self *ViteForkWall) RUnLock() {
	self.rw.RUnlock()
}
