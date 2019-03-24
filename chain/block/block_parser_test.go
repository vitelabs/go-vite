package chain_block

import (
	"fmt"
	"testing"
)

func TestSlice(t *testing.T) {
	a := make([]byte, 100)
	b := []byte{1, 2, 3}
	fmt.Println(cap(a))

	a[:len(b)] = b
	fmt.Println(a)
	fmt.Println(cap(a))

}
