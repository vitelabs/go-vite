package common

import "sync"

func SyncMapLen(m *sync.Map) uint64 {
	if m == nil {
		return 0
	}
	i := uint64(0)
	m.Range(func(key, value interface{}) bool {
		i++
		return true
	})
	return i
}
