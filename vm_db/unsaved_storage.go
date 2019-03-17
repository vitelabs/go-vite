package vm_db

import (
	"bytes"
	"sort"
)

type kvList [][2][]byte

func (list kvList) Len() int           { return len(list) }
func (list kvList) Swap(i, j int)      { list[i], list[j] = list[j], list[i] }
func (list kvList) Less(i, j int) bool { return bytes.Compare(list[i][0], list[j][0]) < 0 }

type UnsavedStorage struct {
	dirty bool

	list kvList
}

func newUnsavedStorage() *UnsavedStorage {
	return &UnsavedStorage{
		dirty: false,
	}
}

func (us *UnsavedStorage) Add(key []byte, value []byte) {
	us.dirty = true
	us.list = append(us.list, [2][]byte{key, value})
}

func (us *UnsavedStorage) GetSortedList() [][2][]byte {
	if us.dirty {
		sort.Sort(us.list)
		us.dirty = false
	}

	return us.list
}
