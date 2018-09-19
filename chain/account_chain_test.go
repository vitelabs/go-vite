package chain

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	var blocks []uint64
	fmt.Println(blocks == nil)
	for _, block := range blocks {
		fmt.Println(block)
	}
}
