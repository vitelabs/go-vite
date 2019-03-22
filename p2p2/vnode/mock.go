package vnode

import (
	crand "crypto/rand"
	mrand "math/rand"
	"net"
)

func mockID() (id NodeID) {
	crand.Read(id[:])
	return
}

func mockIP() (ip net.IP) {
	ipv4 := mrand.Intn(10) > 5
	if ipv4 {
		ip = make(net.IP, 4)
	} else {
		ip = make(net.IP, 16)
	}

	crand.Read(ip)
	return
}

func mockPort() int {
	return mrand.Intn(65535)
}

func mockRest() (ext []byte) {
	ext = make([]byte, mrand.Intn(1000))

	crand.Read(ext)

	return
}

func mockNet() int {
	return mrand.Intn(1000)
}

func MockNode(domain bool, ext bool) *Node {
	n := &Node{
		ID: mockID(),
		EndPoint: EndPoint{
			mockIP(),
			mockPort(),
			HostIP,
		},
		Net: mockNet(),
		Ext: nil,
	}

	if domain {
		n.Host = []byte("www.vite.org")
		n.typ = HostDomain
	}

	if ext {
		n.Ext = mockRest()
	}

	return n
}
