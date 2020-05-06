package utils

import "sync"

type ForkWall interface {
	Wait()
	Add(delta int)
	Done()
}

type ViteForkWall struct {
	rw sync.RWMutex
}

func (fw *ViteForkWall) Lock() {
	fw.rw.Lock()
}

func (fw *ViteForkWall) UnLock(delta int) {
	fw.rw.Unlock()
}

func (fw *ViteForkWall) RLock() {
	fw.rw.RLock()
}

func (fw *ViteForkWall) RUnLock() {
	fw.rw.RUnlock()
}
