package version

import (
	"strconv"
	"sync/atomic"

	"github.com/vitelabs/go-vite/interval/common/log"
)

//var forkVersion int32
//
//func ForkVersion() int {
//	return int(forkVersion)
//}
//
//func IncForkVersion() {
//	for {
//		i := forkVersion
//		if atomic.CompareAndSwapInt32(&forkVersion, i, i+1) {
//			return
//		} else {
//			log.Info("fork version concurrent for %d.", i)
//		}
//	}
//}

type Version struct {
	version int32
}

func (v *Version) Inc() {
	for {
		i := v.version
		if atomic.CompareAndSwapInt32(&v.version, i, i+1) {
			return
		} else {
			log.Info("fork version concurrent for %d.", i)
		}
	}
}

func (v *Version) Val() int {
	return int(v.version)
}

func (v *Version) String() string {
	return strconv.Itoa(int(v.version))
}
