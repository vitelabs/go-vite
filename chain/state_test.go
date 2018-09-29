package chain

import (
	"fmt"
	"testing"
)

func TestNewStateTrie(t *testing.T) {
	chainInstance := getChainInstance()
	newTrie := chainInstance.NewStateTrie()
	fmt.Printf("%+v\n", newTrie)
}
