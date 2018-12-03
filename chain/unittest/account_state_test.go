package chain_unittest

import (
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func Test_getAccountState(t *testing.T) {
	chainInstance := newChainInstance("testdata", false, true)
	stateHash, _ := types.HexToHash("f5a20112ed31ec82c5c5ffdccd1f5685e957b195295b44034bfbf49b8ac1d66b")
	trie := chainInstance.GetStateTrie(&stateHash)
	iter := trie.NewIterator(nil)
	for {
		_, _, ok := iter.Next()
		if !ok {
			break
		}
	}
}
