package discovery

import (
	"sync"
)

type stage struct {
	pool sync.Map // map[address_string]*Node
}

func (s *stage) add(n *Node) {
	s.pool.LoadOrStore(n.Address(), n)
}

func (s *stage) remove(address string) {
	s.pool.Delete(address)
}

func (s *stage) resolve(address string) *Node {
	v, ok := s.pool.Load(address)
	if ok {
		return v.(*Node)
	}

	return nil
}
