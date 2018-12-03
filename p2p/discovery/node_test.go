package discovery

import (
	crand "crypto/rand"
	"fmt"
	"github.com/vitelabs/go-vite/p2p/network"
	mrand "math/rand"
	"net"
	"testing"
)

func mockID() (id NodeID) {
	crand.Read(id[:])
	return
}

func mockIP() net.IP {
	var n int
	if mrand.Intn(10) > 5 {
		n = net.IPv4len
	} else {
		n = net.IPv6len
	}

	ip := make([]byte, n)
	crand.Read(ip)

	return ip
}

func mockPort() uint16 {
	return uint16(mrand.Intn(65535))
}

func mockNode() *Node {
	return &Node{
		ID:  mockID(),
		IP:  mockIP(),
		UDP: mockPort(),
		TCP: mockPort(),
		Net: network.ID(mrand.Uint32()),
	}
}

func compare(n, n2 *Node) bool {
	if n.ID != n2.ID {
		fmt.Println("id", n.ID, n2.ID)
		return false
	}

	if !n.IP.Equal(n2.IP) {
		fmt.Println("ip", n.IP, n2.IP)
		return false
	}

	if n.UDP != n2.UDP {
		fmt.Println("udp", n.UDP, n2.UDP)
		return false
	}

	if n.TCP != n2.TCP {
		fmt.Println("tcp", n.TCP, n2.TCP)
		return false
	}

	if n.Net != n2.Net {
		fmt.Printf("net %d %d\n", n.Net, n2.Net)
		return false
	}

	return true
}

func TestNodeID_IsZero(t *testing.T) {
	if !ZERO_NODE_ID.IsZero() {
		t.Fail()
	}
}

func TestNode_Deserialize(t *testing.T) {
	n := mockNode()
	data, err := n.Serialize()
	if err != nil {
		t.Error(err)
	}

	n2 := new(Node)
	err = n2.Deserialize(data)
	if err != nil {
		t.Error(err)
	}

	if !compare(n, n2) {
		t.Fail()
	}
}

func TestNode_String(t *testing.T) {
	n := mockNode()

	str := n.String()

	n2, err := ParseNode(str)
	if err != nil {
		t.Error(err)
	}

	if !compare(n, n2) {
		t.Fail()
	}
}
