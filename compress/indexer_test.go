package compress

import (
	"fmt"
	"testing"
)

func TestNewIndexer(t *testing.T) {
	a := []uint64{1, 2, 3, 4, 5}
	fmt.Println(a[:3])
	fmt.Println(a[3:])
}
