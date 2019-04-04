package discovery

import (
	"net"
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type mockSocket struct {
}

func (m *mockSocket) ping(n *Node, ch chan<- *Node) (err error) {
	go func() {
		n.ID = vnode.RandomNodeID()
		ch <- n
	}()

	return nil
}

func (m *mockSocket) pong(echo []byte, n *Node) (err error) {
	panic("implement me")
}

func (m *mockSocket) findNode(target vnode.NodeID, count uint32, n *Node, ch chan<- []*vnode.EndPoint) (err error) {
	go func() {
		var eps = make([]*vnode.EndPoint, count)
		for i := uint32(0); i < count; i++ {
			eps[i] = &vnode.EndPoint{
				Host: []byte{0, 0, 0, 0},
				Port: int(i),
				Typ:  vnode.HostIPv4,
			}
		}

		ch <- eps
	}()

	return nil
}

func (m *mockSocket) sendNodes(eps []*vnode.EndPoint, addr *net.UDPAddr) (err error) {
	panic("implement me")
}

func (m *mockSocket) start() error {
	return nil
}

func (m *mockSocket) stop() error {
	return nil
}

func TestFindNode(t *testing.T) {
	tab := newTable(vnode.ZERO, bucketSize, bucketNum, newListBucket, nil)
	tab.add(&Node{
		Node: vnode.Node{
			ID: vnode.RandFromDistance(tab.id, 100),
			EndPoint: vnode.EndPoint{
				Host: []byte{127, 0, 0, 1},
				Port: 8888,
				Typ:  vnode.HostIPv4,
			},
			Net: 0,
			Ext: nil,
		},
	})

	var d = &discovery{
		node: &vnode.Node{
			ID: vnode.ZERO,
			EndPoint: vnode.EndPoint{
				Host: []byte{127, 0, 0, 1},
				Port: 8483,
				Typ:  vnode.HostIPv4,
			},
			Net: 0,
			Ext: nil,
		},
		table:  tab,
		finder: nil,
		stage:  make(map[string]*Node),
		socket: &mockSocket{},
	}

	nodes := d.lookup(vnode.ZERO, 32)
	if len(nodes) != 32 {
		t.Errorf("should not find %d nodes", len(nodes))
	}
}
