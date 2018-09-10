package ledger

import (
	"testing"
)

func TestNewSortedSnapshot(t *testing.T) {
	var testS = map[string]*SnapshotItem{
		"F": &SnapshotItem{},
		"D": &SnapshotItem{},
		"E": &SnapshotItem{},
	}

	ss := newSortedSnapshot(testS)
	for i := 0; i < 1000; i++ {
		addressList := []string{"D", "E", "F"}
		for index, item := range ss {
			if addressList[index] != item.address {
				t.Fatal("Not Valid")
			}
		}
	}

}
