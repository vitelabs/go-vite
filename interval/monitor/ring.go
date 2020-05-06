package monitor

import "sync"

type ring struct {
	cap   int
	datas []interface{}
	i     int
	mu    sync.Mutex
}

func newRing(n int) *ring {
	r := &ring{cap: 0, i: 0, datas: make([]interface{}, n)}
	return r
}

func (r *ring) add(data interface{}) *ring {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.datas[r.i] = data
	r.i = r.nextI(r.i)
	if r.cap < len(r.datas) {
		r.cap++
	}
	return r
}

func (r *ring) reset() *ring {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.i = 0
	r.cap = 0
	return r
}

func (r *ring) all() []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.cap
	result := make([]interface{}, c)
	j := r.i

	for n := c - 1; n >= 0; n-- {
		j = r.lastI(j)
		result[n] = r.datas[j]
	}
	return result
}

func (r *ring) lastI(i int) int {
	if i == 0 {
		l := len(r.datas)
		return l - 1
	} else {
		return i - 1
	}
}
func (r *ring) nextI(i int) int {
	l := len(r.datas)
	i = i + 1
	if i >= l {
		return 0
	} else {
		return i
	}
}
