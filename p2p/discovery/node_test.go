package discovery

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"net"
	"testing"

	"github.com/vitelabs/go-vite/p2p/network"
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

func mockRest() []byte {
	n := mrand.Intn(1000)
	ret := make([]byte, n)
	crand.Read(ret)
	return ret
}

func mockNode(ext bool) *Node {
	n := &Node{
		ID:  mockID(),
		IP:  mockIP(),
		UDP: mockPort(),
		TCP: mockPort(),
		Net: network.ID(mrand.Uint32()),
	}

	if ext {
		n.Ext = mockRest()
	}

	return n
}

func compare(n, n2 *Node, rest bool) bool {
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

	if rest && !bytes.Equal(n.Ext, n2.Ext) {
		fmt.Printf("different rest")
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
	n := mockNode(true)
	data, err := n.Serialize()
	if err != nil {
		t.Error(err)
	}

	n2 := new(Node)
	err = n2.Deserialize(data)
	if err != nil {
		t.Error(err)
	}

	if !compare(n, n2, true) {
		t.Fail()
	}
}

func TestNode_Serialize(t *testing.T) {
	n := mockNode(false)
	data, err := n.Serialize()
	if err != nil {
		t.Error(err)
	}

	n2 := new(Node)
	err = n2.Deserialize(data)
	if err != nil {
		t.Error(err)
	}

	if !compare(n, n2, true) {
		t.Fail()
	}
}

func TestNode_String(t *testing.T) {
	n := mockNode(true)

	str := n.String()

	n2, err := ParseNode(str)
	if err != nil {
		t.Error(err)
	}

	if !compare(n, n2, false) {
		t.Fail()
	}
}

func TestNode_Update(t *testing.T) {
	n := Node{
		ID:  mockID(),
		IP:  mockIP(),
		UDP: mockPort(),
		TCP: mockPort(),
		Net: 0,
		Ext: mockRest(),
	}

	n2 := new(Node)
	*n2 = n

	// different ID
	n2.Update(&Node{
		ID: mockID(),
		IP: mockIP(),
	})
	if !compare(&n, n2, true) {
		t.Fail()
	}

	// nil Ext
	n2.Update(&Node{
		ID:  n.ID,
		Ext: nil,
	})
	if !compare(&n, n2, true) {
		t.Fail()
	}

	// Ext
	n2.Update(&Node{
		ID:  n.ID,
		Ext: mockRest(),
	})
	if compare(&n, n2, true) {
		t.Fail()
	}

	// IP
	n3 := &Node{
		ID:  n.ID,
		IP:  mockIP(),
		UDP: mockPort(),
		TCP: mockPort(),
		Ext: mockRest(),
		Net: network.ID(mrand.Uint32()),
	}
	n2.Update(n3)
	if !compare(n2, n3, true) {
		t.Fail()
	}
}
