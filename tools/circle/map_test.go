package circle

import (
	"fmt"
	"testing"
)

func TestCircleMap_Size(t *testing.T) {
	const total = 5
	cm := NewMap(total)

	for i, j := 1, 10; i < total; i, j = i+1, j+1 {
		cm.Put(i, j)
		if cm.Size() != i {
			t.Fail()
		}
	}

	for i, j := 10, 10; i < total; i, j = i+1, j+1 {
		cm.Put(i, j)
		if cm.Size() != total {
			t.Fail()
		}
	}
}

func TestCircleMap_Get(t *testing.T) {

}

func TestCircleMap_Traverse(t *testing.T) {
	const total = 5
	cm := NewMap(total)

	for i, j := 1, 10; i < total; i, j = i+1, j+1 {
		cm.Put(i, j)
	}

	k, v := 1, 10
	cm.Traverse(func(key Key, value Value) bool {
		fmt.Println(key, value)
		if key != k || value != v {
			t.Fail()
		}
		k++
		v++
		return true
	})

	for i, j := 5, 10; i < 10; i, j = i+1, j+1 {
		cm.Put(i, j)
	}
	k, v = 5, 10
	cm.Traverse(func(key Key, value Value) bool {
		if key != k || value != v {
			t.Fail()
		}
		k++
		v++
		return true
	})

	if k != 10 || v != 15 {
		t.Fail()
	}
}
