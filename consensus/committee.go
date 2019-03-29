package consensus

import (
	"fmt"
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

// update committee result
//type committee struct {
//	common.LifecycleStatus
//
//	mLog log15.Logger
//
//	genesis  time.Time
//	rw       *chainRw
//	periods  *periodLinkedArray
//	snapshot *teller
//	contract *teller
//	tellers  sync.Map
//	signer   types.Address
//
//	whiteProducers map[string]bool
//
//	// subscribes map[types.Gid]map[string]*subscribeEvent
//	subscribes sync.Map
//
//	lruCache *lru.Cache
//	dbCache  *DbCache
//	dbDir    string
//
//	wg     sync.WaitGroup
//	closed chan struct{}
//}

func (self *committee) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	return self.snapshot.VerifySnapshotProducer(header)
}

func (self *committee) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
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

func (self *committee) VerifyABsProducerByGid(gid types.Gid, blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	tel, err := self.contracts.getOrLoadGid(gid)

	if err != nil {
		return nil, err
	}
	if tel == nil {
		return nil, errors.New("consensus group not exist")
	}
	return tel.verifyAccountsProducer(blocks)
}

func (self *committee) VerifyAccountProducer(accountBlock *ledger.AccountBlock) (bool, error) {
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

func (self *committee) ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error) {
	var eResult *electionResult
	var err error
	if gid == types.SNAPSHOT_GID {
		eResult, err = self.snapshot.electionIndex(index)
	} else {
		cs, e := self.contracts.getOrLoadGid(gid)
		if e != nil {
			return nil, 0, e
		}
		eResult, err = cs.electionIndex(index)
	}
	if err != nil {
		return nil, 0, err
	}

	voteTime, _ := self.snapshot.GenVoteTime(index)
	var result []*Event
	for _, p := range eResult.Plans {
		e := newConsensusEvent(eResult, p, gid, voteTime)
		result = append(result, &e)
	}
	return result, uint64(eResult.Index), nil
}

func (self *committee) VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error) {
	if gid == types.SNAPSHOT_GID {
		return self.snapshot.time2Index(t2), nil
	} else {
		cs, e := self.contracts.getOrLoadGid(gid)
		if e != nil {
			return 0, e
		}
		return cs.time2Index(t2), nil
	}
}

func (self *committee) VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error) {
	var info *core.GroupInfo
	if gid == types.SNAPSHOT_GID {
		info = self.snapshot.info
	} else {
		cs, e := self.contracts.getOrLoadGid(gid)
		if e != nil {
			return nil, nil, e
		}
		info = cs.info
	}

	if info == nil {
		return nil, nil, errors.Errorf("consensus group[%s] not exist", gid)
	}

	st, et := info.Index2Time(i)
	return &st, &et, nil
}

//func NewConsensus(genesisTime time.Time, ch ch) *committee {
//	points, err := newPeriodPointArray(ch)
//	if err != nil {
//		panic(errors.Wrap(err, "create period point fail."))
//	}
//
//	rw := &chainRw{rw: ch, periodPoints: points}
//	committee := &committee{rw: rw, periods: points, genesis: genesisTime, whiteProducers: make(map[string]bool)}
//	committee.mLog = log15.New("module", "consensus/committee")
//	db, err := ch.NewDb("consensus")
//	if err != nil {
//		panic(err)
//	}
//	committee.dbCache = &DbCache{db: consensus_db.NewConsensusDB(db)}
//	cache, err := lru.New(1024 * 10)
//	if err != nil {
//		panic(err)
//	}
//	committee.lruCache = cache
//	return committee
//}

func (self *committee) Init() error {
	if !self.PreInit() {
		panic("pre init fail.")
	}
	defer self.PostInit()

	self.snapshot = newSnapshotCs(self.rw, self.mLog)
	self.rw.initArray(self.snapshot)

	self.contracts = newContractCs(self.rw, self.mLog)
	err := self.contracts.LoadGid(types.DELEGATE_GID)

	if err != nil {
		panic(err)
	}
	self.api = &ApiSnapshot{snapshot: self.snapshot}
	return nil
}

func (self *committee) Start() {
	self.PreStart()
	defer self.PostStart()
	self.closed = make(chan struct{})

	self.wg.Add(1)
	snapshotSubs, _ := self.subscribes.LoadOrStore(types.SNAPSHOT_GID, &sync.Map{})

	wrapper := dposReader{self.snapshot, self.contracts, self.mLog}
	common.Go(func() {
		defer self.wg.Done()
		self.update(types.SNAPSHOT_GID, wrapper, snapshotSubs.(*sync.Map))
	})

	self.wg.Add(1)
	contractSubs, _ := self.subscribes.LoadOrStore(types.DELEGATE_GID, &sync.Map{})

	common.Go(func() {
		defer self.wg.Done()
		self.update(types.SNAPSHOT_GID, wrapper, contractSubs.(*sync.Map))
	})
}

func (self *committee) Stop() {
	self.PreStop()
	defer self.PostStop()

	close(self.closed)
	self.wg.Wait()
}

func (self *committee) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event)) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		value, _ = self.subscribes.LoadOrStore(gid, &sync.Map{})
	}
	v := value.(*sync.Map)
	v.Store(id, &subscribeEvent{addr: addr, fn: fn, gid: gid})
}
func (self *committee) UnSubscribe(gid types.Gid, id string) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		return
	}
	v := value.(*sync.Map)
	v.Delete(id)
}

func (self *committee) SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent)) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		value, _ = self.subscribes.LoadOrStore(gid, &sync.Map{})
	}
	v := value.(*sync.Map)
	v.Store(id, &producerSubscribeEvent{fn: fn, gid: gid})
}

func (self *committee) update(gid types.Gid, t dposReader, m *sync.Map) {
	index, err := t.Time2Index(gid, time.Now())
	if err != nil {
		self.mLog.Error(fmt.Sprintf("vote time to index fail, gid:%s", gid), "err", err)
		return
	}
	for !self.Stopped() {
		//var current *memberPlan = nil
		electionResult, err := t.ElectionIndex(gid, index)

		if err != nil {
			self.mLog.Error("can't get election result. time is "+time.Now().Format(time.RFC3339Nano)+"\".", "err", err)
			time.Sleep(time.Second)
			// error handle
			continue
		}

		if electionResult.Index != index {
			self.mLog.Error("can't get Index election result. Index is " + strconv.FormatInt(int64(index), 10))
			index, err = t.Time2Index(gid, time.Now())
			if err != nil {
				self.mLog.Error(fmt.Sprintf("can't get index by time, gid:%s", gid))
				return
			}
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
				self.event(tmpV, tmpResult, t.GenVoteTime(gid, index))
			})
		}

		for _, v := range subs2 {
			tmpV := v
			tmpResult := electionResult
			common.Go(func() {
				self.eventProducer(tmpV, tmpResult, t.GenVoteTime(gid, index))
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
func (self *committee) eventProducer(e *producerSubscribeEvent, result *electionResult, voteTime time.Time) {
	self.wg.Add(1)
	defer self.wg.Done()
	var r []types.Address
	for _, v := range result.Plans {
		r = append(r, v.Member)
	}
	e.fn(ProducersEvent{Addrs: r, Index: result.Index, Gid: e.gid})
}

func (self *committee) event(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	self.wg.Add(1)
	defer self.wg.Done()
	if e.addr == nil {
		// all
		self.eventAll(e, result, voteTime)
	} else {
		self.eventAddr(e, result, voteTime)
	}
}

func (self *committee) eventAll(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		now := time.Now()
		sub := p.STime.Sub(now)
		if sub+time.Second < 0 {
			continue
		}

		if sub > time.Millisecond*10 {
			time.Sleep(sub)
		}

		e.fn(newConsensusEvent(result, p, e.gid, voteTime))
	}
}
func (self *committee) eventAddr(e *subscribeEvent, result *electionResult, voteTime time.Time) {
	for _, p := range result.Plans {
		if p.Member == *e.addr {
			now := time.Now()
			sub := p.STime.Sub(now)
			if sub+time.Second < 0 {
				continue
			}
			if sub > time.Millisecond*10 {
				time.Sleep(sub)
			}
			e.fn(newConsensusEvent(result, p, e.gid, voteTime))
		}
	}
}

func newConsensusEvent(r *electionResult, p *core.MemberPlan, gid types.Gid, voteTime time.Time) Event {
	return Event{
		Gid:               gid,
		Address:           p.Member,
		Stime:             p.STime,
		Etime:             p.ETime,
		Timestamp:         p.STime,
		SnapshotHash:      r.Hash,
		SnapshotHeight:    r.Height,
		SnapshotTimeStamp: r.Timestamp,
		VoteTime:          voteTime,
		PeriodStime:       r.STime,
		PeriodEtime:       r.ETime,
	}
}
