package consensus

import (
	"time"

	"strconv"

	"sync"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type subscribeEvent struct {
	addr *types.Address
	gid  types.Gid
	fn   func(Event)
}

// update committee result
type committee struct {
	common.LifecycleStatus

	genesis  time.Time
	rw       *chainRw
	snapshot *teller
	contract *teller
	tellers  sync.Map
	signer   types.Address
	signerFn SignerFn

	// subscribes map[types.Gid]map[string]*subscribeEvent
	subscribes sync.Map

	wg sync.WaitGroup
}

func (self *committee) Authorize(signer types.Address, fn SignerFn) {
	self.signer = signer
	self.signerFn = fn
}

func (self *committee) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	gid := types.SNAPSHOT_GID
	t, ok := self.tellers.Load(gid)
	if !ok {
		t = self.initTeller(gid)
	}
	tel := t.(*teller)
	electionResult, err := tel.electionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}
func (self *committee) initTeller(gid types.Gid) *teller {
	info := self.rw.GetMemberInfo(gid, self.genesis)
	t := newTeller(info, self.rw)
	self.tellers.Store(gid, t)
	return t
}

func (self *committee) VerifyAccountProducer(header *ledger.AccountBlock) (bool, error) {
	gid := types.DELEGATE_GID
	t, ok := self.tellers.Load(gid)
	if !ok {
		t = self.initTeller(gid)
	}
	tel := t.(*teller)

	electionResult, err := tel.electionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	if electionResult.Hash != header.SnapshotHash {
		return false, nil
	}
	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (self *committee) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}
	for _, plan := range result.Plans {
		if plan.Member == address {
			if plan.STime == t {
				return true
			} else {
				return false
			}
		}
	}
	return true
}

func NewCommittee(genesisTime time.Time, interval int32, memberCnt int32, perCnt int32, rw *chainRw) *committee {
	committee := &committee{rw: rw, genesis: genesisTime}
	return committee
}

func (self *committee) Init() {
	self.PreInit()
	defer self.PostInit()
	self.snapshot = self.initTeller(types.SNAPSHOT_GID)
	self.contract = self.initTeller(types.DELEGATE_GID)
}

func (self *committee) Start() {
	self.PreStart()
	defer self.PostStart()

	self.wg.Add(1)
	snapshotSubs, _ := self.subscribes.LoadOrStore(types.SNAPSHOT_GID, &sync.Map{})
	go self.update(self.snapshot, snapshotSubs.(*sync.Map))

	self.wg.Add(1)
	contractSubs, _ := self.subscribes.LoadOrStore(types.DELEGATE_GID, &sync.Map{})
	go self.update(self.contract, contractSubs.(*sync.Map))
}

func (self *committee) Stop() {
	self.PreStop()
	defer self.PostStop()
	self.wg.Wait()
}

func (self *committee) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event)) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		value, _ = self.subscribes.LoadOrStore(gid, &sync.Map{})
	}
	v := value.(*sync.Map)
	v.Store(id, &subscribeEvent{addr: addr, fn: fn})
}
func (self *committee) UnSubscribe(gid types.Gid, id string) {
	value, ok := self.subscribes.Load(gid)
	if !ok {
		return
	}
	v := value.(*sync.Map)
	v.Delete(id)
}

func (self *committee) update(t *teller, m *sync.Map) {
	defer self.wg.Done()
	log := log15.New("module", "committee")

	index := t.time2Index(time.Now())
	var lastRemoveTime = time.Now()
	for !self.Stopped() {
		//var current *memberPlan = nil
		electionResult, err := t.electionIndex(index)

		if err != nil {
			log.Error("can't get election result. time is "+time.Now().Format(time.RFC3339Nano)+"\".", "err", err)
			time.Sleep(time.Duration(t.info.interval) * time.Second)
			// error handle
			continue
		}

		if electionResult.Index != index {
			log.Error("can't get Index election result. Index is " + strconv.FormatInt(int64(index), 10))
			index = index + 1
			continue
		}
		subs := copyMap(m)

		if len(subs) == 0 {
			time.Sleep(electionResult.ETime.Sub(time.Now()))
			continue
		}

		for _, v := range subs {
			self.wg.Add(1)
			go self.event(v, electionResult)
		}

		time.Sleep(electionResult.ETime.Sub(time.Now()) - time.Second)
		index = electionResult.Index + 1

		// clear ever hour
		removeTime := time.Now().Add(-time.Hour)
		if lastRemoveTime.Before(removeTime) {
			t.removePrevious(removeTime)
			lastRemoveTime = removeTime
		}

	}
}
func copyMap(m *sync.Map) map[string]*subscribeEvent {
	var result map[string]*subscribeEvent
	m.Range(func(k, v interface{}) bool {
		result[k.(string)] = v.(*subscribeEvent)
		return true
	})
	return result
}

func (self *committee) event(e *subscribeEvent, result *electionResult) {
	self.wg.Done()
	defer self.wg.Done()
	if e.addr == nil {
		// all
		self.eventAll(e, result)
	} else {
		self.eventAddr(e, result)
	}
}

func (self *committee) eventAll(e *subscribeEvent, result *electionResult) {
	for _, p := range result.Plans {
		now := time.Now()
		sub := p.STime.Sub(now)
		if sub+time.Second < 0 {
			continue
		}

		if sub > time.Millisecond*10 {
			time.Sleep(sub)
		}

		e.fn(newConsensusEvent(result, p, e))
	}
}
func (self *committee) eventAddr(e *subscribeEvent, result *electionResult) {
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
			e.fn(newConsensusEvent(result, p, e))
		}
	}
}

func newConsensusEvent(r *electionResult, p *memberPlan, e *subscribeEvent) Event {
	return Event{
		Gid:            e.gid,
		Address:        p.Member,
		Stime:          p.STime,
		Etime:          p.ETime,
		Timestamp:      p.STime,
		SnapshotHash:   r.Hash,
		SnapshotHeight: r.Height,
	}
}
