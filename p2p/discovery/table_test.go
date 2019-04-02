package discovery

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

var self = vnode.MockNode(false, true)

type mockPinger struct {
}

func (mp *mockPinger) ping(n *Node) error {
	if rand.Intn(10) > 5 {
		return nil
	}

	return errors.New("mock error")
}

func TestTable_add(t *testing.T) {
	const total = 5
	newTable(self, bucketSize, bucketNum, newListBucket, &mockPinger{})
}
