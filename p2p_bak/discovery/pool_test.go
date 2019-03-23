package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

func TestPool_loop(t *testing.T) {
	pool := newWtPool()
	pool.start()

	done := make(chan bool, 1)
	nodes := make(chan []*Node, 1)

	add := pool.add(&wait{
		expectFrom: NodeID{},
		expectCode: 0,
		sourceHash: types.Hash{},
		handler: &pingWait{
			done: done,
		},
		expiration: time.Now().Add(10 * time.Second),
	})
	if !add {
		t.Fail()
	}

	add = pool.add(&wait{
		expectFrom: NodeID{},
		expectCode: 0,
		sourceHash: types.Hash{},
		handler: &findnodeWait{
			rec: nil,
			ch:  nodes,
		},
		expiration: time.Time{},
	})

	if !add {
		t.Fail()
	}

	if pool.size() != 2 {
		t.Fail()
	}

	fmt.Println("wait")
	if false != <-done || nil != <-nodes {
		t.Fail()
	}

	if pool.size() != 0 {
		t.Fail()
	}

	pool.stop()
}
