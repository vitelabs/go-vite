package pool

import (
	"strconv"
	"sync/atomic"

	"github.com/vitelabs/go-vite/log15"
)

type ForkVersion struct {
	version int32
}

func (self *ForkVersion) Inc() {
	for {
		i := self.version
		if atomic.CompareAndSwapInt32(&self.version, i, i+1) {
			return
		} else {
			log15.Info("fork version concurrent.", "version", i)
		}
	}
}

func (self *ForkVersion) Val() int {
	return int(self.version)
}

func (self *ForkVersion) String() string {
	return strconv.Itoa(int(self.version))
}
