package p2p

import (
	"sync"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type peerList struct {
	list []PeerMux
	max  int
}

func (l *peerList) available() bool {
	return len(l.list) < l.max
}

func (l *peerList) add(p PeerMux) {
	l.list = append(l.list, p)
}

type peers struct {
	rw      sync.RWMutex
	peerMap map[vnode.NodeID]PeerMux
	levels  map[Level]*peerList
}

func newPeers(maxPeers map[Level]int) *peers {
	ps := &peers{
		peerMap: make(map[vnode.NodeID]PeerMux),
		levels:  make(map[Level]*peerList),
	}

	for level, max := range maxPeers {
		ps.levels[level] = &peerList{
			list: make([]PeerMux, 0, max),
			max:  max,
		}
	}

	return ps
}

func (s *peers) has(id vnode.NodeID) bool {
	s.rw.RLock()
	defer s.rw.RUnlock()

	_, ok := s.peerMap[id]

	return ok
}

func (s *peers) resize(level Level, max int) {
	s.rw.Lock()
	defer s.rw.Unlock()

	if l, ok := s.levels[level]; ok {
		l.max = max
	} else {
		s.levels[level] = &peerList{
			list: make([]PeerMux, 0, max),
			max:  max,
		}
	}
}

// add success return ok is true.
// if level is full, then fall to low level, and so on, until level Inbound.
// if all those low levels are full, then return PeerTooManyPeers.
func (s *peers) add(p PeerMux) (e PeerError, ok bool) {
	s.rw.Lock()
	defer s.rw.Unlock()

	id := p.ID()
	if _, exist := s.peerMap[id]; exist {
		return PeerAlreadyConnected, false
	}

	level := p.Level()
	for {
		if l, exist := s.levels[level]; exist {
			if l.available() {
				// not full
				l.add(p)
				s.peerMap[id] = p
				return 0, true
			}
		}

		if level == Inbound {
			break
		}

		level--
	}

	return PeerTooManyPeers, false
}

// remove return error if p is not exist
func (s *peers) remove(p PeerMux) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	id := p.ID()
	if _, ok := s.peerMap[id]; ok {
		delete(s.peerMap, id)
		level := p.Level()
		var l *peerList
		for {
			if l, ok = s.levels[level]; ok {
				for i, p2 := range l.list {
					if p2.ID() == id {
						copy(l.list[i:], l.list[i+1:])
						l.list = l.list[:len(l.list)-1]
						return nil
					}
				}
			}

			if level == Inbound {
				break
			}

			level--
		}
	}

	return errPeerNotExist
}

// changeLevel move p from old level to new level.
// if p is not exist, return `errPeerNotExist`.
// if new level is full, return `errPeerNotExist`.
// if old and those lower levels cannot find p, return `errPeerNotExist`.
func (s *peers) changeLevel(p PeerMux, old Level) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	id := p.ID()

	if _, ok := s.peerMap[id]; ok {
		newLevel := p.Level()
		var l *peerList
		if l, ok = s.levels[newLevel]; ok {
			if l.available() {
				l.add(p)

				// remove from old level
				for {
					if l, ok = s.levels[old]; ok {
						for i, p2 := range l.list {
							if p2.ID() == id {
								copy(l.list[i:], l.list[i+1:])
								l.list = l.list[:len(l.list)-1]
								return nil
							}
						}
					}

					if old == Inbound {
						return errPeerNotExist
					}

					old--
				}

			} else {
				return errLevelIsFull
			}
		}
	}

	return errPeerNotExist
}

func (s *peers) count() (n int) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	return len(s.peerMap)
}

// levelCount is for test
func (s *peers) levelCount(level Level) (n int) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	if l, ok := s.levels[level]; ok {
		return len(l.list)
	}

	return 0
}

func (s *peers) max() (n int) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	for _, l := range s.levels {
		n += l.max
	}

	return
}

func (s *peers) inboundSlots() int {
	s.rw.RLock()
	defer s.rw.RUnlock()

	if l, ok := s.levels[Inbound]; ok {
		return l.max - len(l.list)
	}

	return 0
}

func (s *peers) close() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for _, p := range s.peerMap {
		_ = p.Close(PeerQuitting)
	}

	s.peerMap = make(map[vnode.NodeID]PeerMux)
	for _, l := range s.levels {
		l.list = l.list[:0]
	}
}

func (s *peers) info() []PeerInfo {
	s.rw.RLock()
	defer s.rw.RUnlock()

	infos := make([]PeerInfo, 0, len(s.peerMap))

	for _, p := range s.peerMap {
		infos = append(infos, p.Info())
	}

	return infos
}

func (s *peers) peers() []PeerMux {
	s.rw.RLock()
	defer s.rw.RUnlock()

	ps := make([]PeerMux, 0, len(s.peerMap))
	for _, p := range s.peerMap {
		ps = append(ps, p)
	}

	return ps
}
