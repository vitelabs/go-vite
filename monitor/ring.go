package monitor

import "sync"

// simple implement of ringBuffer
type ring struct {
	cap   int
	datas []interface{}
	i     int
	mu    sync.Mutex
}

// public
func newRing(n int) *ring {
	r := &ring{cap: 0, i: 0, datas: make([]interface{}, n)}
	return r
}

// public
func (self *ring) add(data interface{}) *ring {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.datas[self.i] = data
	self.i = self.nextI(self.i)
	if self.cap < len(self.datas) {
		self.cap++
	}
	return self
}

// public
func (self *ring) reset() *ring {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.i = 0
	self.cap = 0
	return self
}

// public
func (self *ring) all() []interface{} {
	self.mu.Lock()
	defer self.mu.Unlock()
	c := self.cap
	result := make([]interface{}, c)
	j := self.i

	for n := c - 1; n >= 0; n-- {
		j = self.lastI(j)
		result[n] = self.datas[j]
	}
	return result
}

// private
func (self *ring) lastI(i int) int {
	if i == 0 {
		l := len(self.datas)
		return l - 1
	} else {
		return i - 1
	}
}

// private
func (self *ring) nextI(i int) int {
	l := len(self.datas)
	i = i + 1
	if i >= l {
		return 0
	} else {
		return i
	}
}
