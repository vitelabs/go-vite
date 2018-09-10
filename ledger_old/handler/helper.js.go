package handler

import "sync"

var lock sync.Mutex
var currentNewTaskId = uint64(0)

func getNewNetTaskId() uint64 {
	lock.Lock()
	defer lock.Unlock()

	currentNewTaskId++
	return currentNewTaskId
}
