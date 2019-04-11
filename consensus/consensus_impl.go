package consensus

import (
	"time"

	"strconv"

	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
)

type subscribeEvent struct {
	addr *types.Address
	gid  types.Gid
	fn   func(Event)
}
type producerSubscribeEvent struct {
	gid types.Gid
	fn  func(ProducersEvent)
}

func (self *consensus) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	return self.snapshot.VerifyProducer(header.Producer(), *header.Timestamp)
}

func (self *consensus) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	var result []*ledger.AccountBlock
	for k, v := range abs {
		blocks, err := self.VerifyABsProducerByGid(k, v)
		if err != nil {
			return nil, err
		}
		for _, b := range blocks {
			result = append(result, b)
		}
	}
	return result, nil
}

func (self *consensus) VerifyABsProducerByGid(gid types.Gid, blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	tel, err := self.contracts.getOrLoadGid(gid)

	if err != nil {
		return nil, err
	}
	if tel == nil {
		return nil, errors.New("consensus group not exist")
	}
	return tel.verifyAccountsProducer(blocks)
}

func (self *consensus) VerifyAccountProducer(accountBlock *ledger.AccountBlock) (bool, error) {
	gid, err := self.rw.getGid(accountBlock)
	if err != nil {
		return false, err
	}
	tel, err := self.contracts.getOrLoadGid(*gid)

	if err != nil {
		return false, err
	}
	if tel == nil {
		return false, errors.New("consensus group not exist")
	}
	return tel.VerifyAccountProducer(accountBlock)
}

func (self *consensus) ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error) {
	// load from dpos wrapper
	reader, err := self.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return nil, 0, err
	}

	// cal votes
	eResult, err := reader.ElectionIndex(index)
	if err != nil {
		return nil, 0, err
	}

	voteTime := self.snapshot.GenVoteTime(index)
	var result []*Event
	for _, p := range eResult.Plans {
		e := newConsensusEvent(eResult, p, gid, voteTime)
		result = append(result, &e)
	}
	return result, uint64(eResult.Index), nil
}

func (self *consensus) VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error) {
	// load from dpos wrapper
	reader, err := self.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return 0, err
	}
	return reader.Time2Index(t2), nil
}

func (self *consensus) VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error) {
	// load from dpos wrapper
	reader, err := self.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return nil, nil, errors.Errorf("consensus group[%s] not exist", gid)
	}
	info := reader.GetInfo()

	if info == nil {
		return nil, nil, errors.Errorf("consensus group[%s] not exist", gid)
	}

	st, et := info.Index2Time(i)
	return &st, &et, nil
}

func (self *consensus) Init() error {
	if !self.PreInit() {
		panic("pre init fail.")
	}
	defer self.PostInit()

	snapshot := newSnapshotCs(self.rw, self.mLog)
	self.snapshot = snapshot
	self.rw.initArray(snapshot)

	self.contracts = newContractCs(self.rw, self.mLog)
	err := self.contracts.LoadGid(types.DELEGATE_GID)

	if err != nil {
		panic(err)
	}
	self.dposWrapper = &dposReader{self.snapshot, self.contracts, self.mLog}
	self.api = &ApiSnapshot{snapshot: snapshot}
	return nil
}

func (self *consensus) Start() {
	self.PreStart()
	defer self.PostStart()
	self.closed = make(chan struct{})

	snapshotSubs, _ := self.subscribes.LoadOrStore(types.SNAPSHOT_GID, &sync.Map{})

	common.Go(func() {
		self.wg.Add(1)
		defer self.wg.Done()
		self.update(types.SNAPSHOT_GID, self.snapshot, snapshotSubs.(*sync.Map))
	})

	contractSubs, _ := self.subscribes.LoadOrStore(types.DELEGATE_GID, &sync.Map{})
	reader, err := self.dposWrapper.getDposConsensus(types.DELEGATE_GID)
	if err != nil {
		panic(err)
	}

	common.Go(func() {
		self.wg.Add(1)
		defer self.wg.Done()
		self.update(types.DELEGATE_GID, reader, contractSubs.(*sync.Map))
	})

	self.rw.Start()
}

func (self *consensus) Stop() {
	self.PreStop()
	defer self.PostStop()
	self.rw.Stop()
	close(self.closed)
	self.wg.Wait()
}

func (self *consensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event)) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		value, _ = self.subscribes.LoadOrStore(gid, &sync.Map{})
	}
	v := value.(*sync.Map)
	v.Store(id, &subscribeEvent{addr: addr, fn: fn, gid: gid})
}
func (self *consensus) UnSubscribe(gid types.Gid, id string) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		return
	}
	v := value.(*sync.Map)
	v.Delete(id)
}

func (self *consensus) SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent)) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		value, _ = self.subscribes.LoadOrStore(gid, &sync.Map{})
	}
	v := value.(*sync.Map)
	v.Store(id, &producerSubscribeEvent{fn: fn, gid: gid})
}

func (self *consensus) update(gid types.Gid, t DposReader, m *sync.Map) {
	index := t.Time2Index(time.Now())
	for !self.Stopped() {
		//var current *memberPlan = nil
		electionResult, err := t.ElectionIndex(index)

		if err != nil {
			self.mLog.Error("can't get election result. time is "+time.Now().Format(time.RFC3339Nano)+"\".", "err", err)
			select {
			case <-self.closed:
				return
			case <-time.After(time.Second):
			}
			// error handle
			continue
		}

		if electionResult.Index != index {
			self.mLog.Error("can't get Index election result. Index is " + strconv.FormatInt(int64(index), 10))
			index = t.Time2Index(time.Now())
			continue
		}
		subs1, subs2 := copyMap(m)

		if len(subs1) == 0 && len(subs2) == 0 {
			select {
			case <-time.After(electionResult.ETime.Sub(time.Now())):
			case <-self.closed:
				return
			}
			index = index + 1
			continue
		}

		for _, v := range subs1 {
			tmpV := v
			tmpResult := electionResult
			common.Go(func() {
				self.event(tmpV, tmpResult, t.GenVoteTime(index))
			})
		}

		for _, v := range subs2 {
			tmpV := v
			tmpResult := electionResult
			common.Go(func() {
				self.eventProducer(tmpV, tmpResult, t.GenVoteTime(index))
			})
		}

		sleepT := electionResult.ETime.Sub(time.Now()) - time.Millisecond*500
		select {
		case <-time.After(sleepT):
		case <-self.closed:
			return
		}
		index = electionResult.Index + 1
	}
}
func copyMap(m *sync.Map) (map[string]*subscribeEvent, map[string]*producerSubscribeEvent) {
	r1 := make(map[string]*subscribeEvent)
	r2 := make(map[string]*producerSubscribeEvent)
	m.Range(func(k, v interface{}) bool {
		switch t := v.(type) {
		case *subscribeEvent:
			r1[k.(string)] = t
		case *producerSubscribeEvent:
			r2[k.(string)] = t
		}
		return true
	})
	return r1, r2
}
func (self *consensus) eventProducer(e *producerSubscribeEvent, result *electionResult, voteTime time.Time) {
	self.wg.Add(1)
	defer self.wg.Done()
	var r []types.Address
	for _, v := range result.Plans {
		r = append(r, v.Member)
	}
	e.fn(ProducersEvent{Addrs: r, Index: result.Index, Gid: e.gid})
}

func (self *consensus) event(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	self.wg.Add(1)
	defer self.wg.Done()
	if e.addr == nil {
		// all
		self.eventAll(e, result, voteTime)
	} else {
		self.eventAddr(e, result, voteTime)
	}
}

func (self *consensus) eventAll(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		now := time.Now()
		sub := p.STime.Sub(now)
		if sub+time.Second < 0 {
			continue
		}
		if sub > time.Millisecond*10 {
			select {
			case <-self.closed:
				return
			case <-time.After(sub):
			}
		}
		e.fn(newConsensusEvent(result, p, e.gid, voteTime))
	}
}
func (self *consensus) eventAddr(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		if p.Member == *e.addr {
			now := time.Now()
			sub := p.STime.Sub(now)
			if sub+time.Second < 0 {
				continue
			}
			if sub > time.Millisecond*10 {
				select {
				case <-self.closed:
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
