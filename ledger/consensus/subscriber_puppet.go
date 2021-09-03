package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
)

type subscriber_puppet struct {
	*consensusSubscriber

	snapshot DposReader
}

func newSubscriberPuppet(snapshot DposReader) *subscriber_puppet {
	return &subscriber_puppet{
		consensusSubscriber: newConsensusSubscriber(),
		snapshot:            snapshot,
	}
}

func (cs subscriber_puppet) triggerEvent(gid types.Gid, fn func(*subscribeEvent)) {
	if gid == types.SNAPSHOT_GID {
		return
	}
	cs.consensusSubscriber.triggerEvent(gid, fn)
}

func (cs subscriber_puppet) TriggerMineEvent(addr types.Address) {
	sTime := time.Now()
	eTime := sTime.Add(time.Duration(cs.snapshot.GetInfo().Interval))
	index := cs.snapshot.Time2Index(sTime)
	periodStartTime, periodEndTime := cs.snapshot.Index2Time(index)
	voteTime := cs.snapshot.GenProofTime(index)

	cs.triggerEvent(types.SNAPSHOT_GID, func(e *subscribeEvent) {
		common.Go(func() {
			e.fn(Event{
				Gid:         types.SNAPSHOT_GID,
				Address:     addr,
				Stime:       sTime,
				Etime:       eTime,
				Timestamp:   sTime,
				VoteTime:    voteTime,
				PeriodStime: periodStartTime,
				PeriodEtime: periodEndTime,
			})
		})

	})
}
