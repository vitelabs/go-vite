package vnode

import (
	crand "crypto/rand"
	mrand "math/rand"
)

func mockID() (id NodeID) {
	crand.Read(id[:])
	return
}

func mockHost() (host []byte) {
	host = make([]byte, mrand.Intn(MaxHostLength))

	crand.Read(host)

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

func MockNode(ext bool) *Node {
	n := &Node{
		ID:       mockID(),
		Hostname: mockHost(),
		Port:     mockPort(),
		Net:      mrand.Int(),
		Ext:      nil,
	}

	if ext {
		n.Ext = mockRest()
	}

	return n
}
