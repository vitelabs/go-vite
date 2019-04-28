package consensus

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
)

// (start, end]
func (self *consensus) OnChainGC(start *ledger.SnapshotBlock, end *ledger.SnapshotBlock) error {
	if start == nil || end == nil {
		panic(fmt.Sprintf("start[%t] or end[%t] is nil.", start == nil, end == nil))
	}

	stime := start.Timestamp
	etime := end.Timestamp

	sIndex := self.rw.dayPoints.Time2Index(*stime)
	eIndex := self.rw.dayPoints.Time2Index(*etime)

	for i := sIndex; i <= eIndex; i++ {
		day, err := self.rw.dayPoints.GetByIndex(i)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("load day index[%d] fail.", i))
		}
		byt, _ := json.Marshal(day)
		self.mLog.Info("reload day[%d] stats for chain gc. %s", i, string(byt))
	}
	return nil
}
