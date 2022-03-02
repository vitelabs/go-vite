package consensus

import (
	"context"
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

type subscribeEvent struct {
	addr *types.Address
	gid  types.Gid
	fn   func(Event)
}

func (e subscribeEvent) trigger(ctx context.Context, result *electionResult, voteTime time.Time) {
	if e.addr == nil {
		// all
		e.triggerAll(ctx, result, voteTime)
	} else {
		e.triggerAddr(ctx, result, voteTime)
	}
}

func (e subscribeEvent) triggerAll(ctx context.Context, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		now := time.Now()
		sub := p.STime.Sub(now)
		if sub+time.Second < 0 {
			continue
		}
		if sub > time.Millisecond*10 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sub):
			}
		}
		e.fn(newConsensusEvent(result, p, e.gid, voteTime))
	}
}
func (e subscribeEvent) triggerAddr(ctx context.Context, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		if p.Member == *e.addr {
			now := time.Now()
			sub := p.STime.Sub(now)
			if sub+time.Second < 0 {
				continue
			}
			if sub > time.Millisecond*10 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(sub):
				}
			}
			e.fn(newConsensusEvent(result, p, e.gid, voteTime))
		}
	}
}

func newConsensusEvent(r *electionResult, p *core.MemberPlan, gid types.Gid, voteTime time.Time) Event {
	return Event{
		Gid:         gid,
		Address:     p.Member,
		Stime:       p.STime,
		Etime:       p.ETime,
		Timestamp:   p.STime,
		VoteTime:    voteTime,
		PeriodStime: r.STime,
		PeriodEtime: r.ETime,
	}
}

type producerSubscribeEvent struct {
	gid types.Gid
	fn  func(ProducersEvent)
}

func (e producerSubscribeEvent) trigger(ctx context.Context, result *electionResult, voteTime time.Time) {
	var r []types.Address
	for _, v := range result.Plans {
		r = append(r, v.Member)
	}
	e.fn(ProducersEvent{Addrs: r, Index: result.Index, Gid: e.gid})
}
