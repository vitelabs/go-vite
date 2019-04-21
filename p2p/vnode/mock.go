package vnode

import (
	crand "crypto/rand"
	mrand "math/rand"
	"net"
)

func mockIP() (ip net.IP, typ HostType) {
	ipv4 := mrand.Intn(10) > 5
	if ipv4 {
		ip = make(net.IP, 4)
		typ = HostIPv4
	} else {
		ip = make(net.IP, 16)
		typ = HostIPv6
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
		ID: RandomNodeID(),
		EndPoint: EndPoint{
			Port: mockPort(),
		},
		Net: mockNet(),
		Ext: nil,
	}

	if domain {
		n.EndPoint.Host = []byte("www.vite.org")
		n.EndPoint.Typ = HostDomain
	} else {
		n.EndPoint.Host, n.EndPoint.Typ = mockIP()
	}

	if ext {
		n.Ext = mockRest()
	}

	return n
}
