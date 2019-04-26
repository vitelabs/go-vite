package common

import (
	"sync/atomic"
)

type Version struct {
	version uint64
}

func (self *Version) Inc() {
	atomic.AddUint64(&self.version, 1)
}

func (self *Version) Val() uint64 {
	return atomic.LoadUint64(&self.version)
}
