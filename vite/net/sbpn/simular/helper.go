package main

import (
	crand "crypto/rand"
	"math/rand"
	"sync"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/discovery"
)

type mocker struct {
	self types.Address
	adds []types.Address
	IDs  []discovery.NodeID
	rw   sync.RWMutex
}

func (m *mocker) randAddr() (addr types.Address) {
	if rand.Intn(100) < 30 {
		return m.self
	}

	if len(m.adds) > 0 && rand.Intn(10) > 5 {
		m.rw.RLock()
		defer m.rw.RUnlock()
		i := rand.Intn(len(m.adds))
		addr = m.adds[i]
	} else {
		m.rw.Lock()
		defer m.rw.Unlock()
		crand.Read(addr[:])
		m.adds = append(m.adds, addr)
	}
	return
}

func (m *mocker) randID() (id discovery.NodeID) {
	if len(m.IDs) > 0 && rand.Intn(10) > 5 {
		m.rw.RLock()
		defer m.rw.RUnlock()
		i := rand.Intn(len(m.IDs))
		id = m.IDs[i]
	} else {
		m.rw.Lock()
		defer m.rw.Unlock()
		crand.Read(id[:])
		m.IDs = append(m.IDs, id)
	}
	return
}
