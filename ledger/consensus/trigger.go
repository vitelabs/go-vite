package consensus

import (
	"context"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/pool/lock"
	"github.com/vitelabs/go-vite/v2/log15"
)

type trigger struct {
	lock lock.ChainRollback
	mLog log15.Logger
}

func newTrigger(lock lock.ChainRollback) *trigger {
	return &trigger{lock: lock, mLog: log15.New("module", "consensus/trigger")}
}

func (tg *trigger) update(ctx context.Context, gid types.Gid, t DposReader, sub subscribeTrigger) {
	index := t.Time2Index(time.Now())

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		//var current *memberPlan = nil

		tg.mLog.Info("trigger update", "gid", gid, "index", index)
		tg.lock.RLockRollback()
		electionResult, err := t.ElectionIndex(index)
		tg.lock.RUnLockRollback()

		if err != nil {
			tg.mLog.Error("can't get election result. time is "+time.Now().Format(time.RFC3339Nano)+"\".", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			// error handle
			continue
		}

		if electionResult.Index != index {
			tg.mLog.Error("can't get Index election result. Index is " + strconv.FormatInt(int64(index), 10))
			index = t.Time2Index(time.Now())
			continue
		}

		// trigger event
		sub.triggerEvent(gid, func(e *subscribeEvent) {
			common.Go(func() {
				e.trigger(ctx, electionResult, t.GenProofTime(index))
			})
		})
		sub.triggerProducerEvent(gid, func(e *producerSubscribeEvent) {
			common.Go(func() {
				e.trigger(ctx, electionResult, t.GenProofTime(index))
			})
		})

		sleepT := time.Until(electionResult.ETime) - time.Millisecond*500
		tg.mLog.Info("trigger update sleep", "gid", gid, "index", index, "sleepT", sleepT.String())
		select {
		case <-time.After(sleepT):
		case <-ctx.Done():
			return
		}
		index = electionResult.Index + 1
	}
}
