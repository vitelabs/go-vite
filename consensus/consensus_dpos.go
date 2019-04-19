package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/consensus/core"
)

type consensusDpos struct {
	info *core.GroupInfo
}

func (self consensusDpos) GetInfo() *core.GroupInfo {
	return self.info
}

func (self consensusDpos) Time2Index(t time.Time) uint64 {
	return self.info.Time2Index(t)
}

func (self consensusDpos) Index2Time(i uint64) (time.Time, time.Time) {
	sTime := self.info.GenSTime(i)
	eTime := self.info.GenETime(i)
	return sTime, eTime
}
