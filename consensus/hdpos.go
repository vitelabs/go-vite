package consensus

import (
	"github.com/inconshreveable/log15"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

var DefaultMembers = []string{
	"vite_2ad661b3b5fa90af7703936ba36f8093ef4260aaaeb5f15cf8",
	"vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c3a",
	"vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8",
	"vite_85e8adb768aed85f2445eb1d71b933370d2980916baa3c1f3c",
	"vite_93dd41694edd756512da7c4af429f3e875c374a53bfd217e00",
	//"vite_f8dfcad17c08f9748271cce96eddc2b3732b399f6367597708",
	//"vite_e77a71d44c65155e1474d708134c53c9dfb7af08b0299dc10d",
	//"vite_1e7d413c276725c6a9e2f5fa8fadfa291e16ea3695c27760b1",
	//"vite_946e7f9554cc5efc6e404a6510c3c6a098ac4296c049cc348b",
	//"vite_d6bf8cf4b590c8e7bfc846463e4b67522d5dda3040d4ff83e8",
	//"vite_ed143aa15a2bfb039b903e7368a297dc0a11288b4af7d7ceb5",
	//"vite_559ddc6918f287316a93ed063ca64621fe2e33a45457c90c11",
	//"vite_bdcb0d7ab9b3db2b229b70613f6a3d18c083457fae3289e926",
	//"vite_080d19e9edf8365e79a7a3e57d253f072648c843d1f7084818",
	//"vite_771e5d643432cd3a50249ed42d1e09c198f1ebd0425f8090e4",
	//"vite_0575741ef5b9fe23a107cc8e2affbd6799088f2debab8264af",
	//"vite_bae6cb6e8dcaa3cd7475fdf5520f14c0556bf9b0950aed0fd4",
	//"vite_cd585066ed35c1e27c9ac32a8fe13c6a1addbbe21e9badc84c",
	//"vite_98fe74d859935cbf833b876599abd9fc4334719cd62d039b5e",
	//"vite_d1f21f486b4ae94e2f3ce237e620ad524fbd6c1313275ef64c",
	//"vite_c873add4fada0029da924ec3557bd838a8de2220917da10569",
	//"vite_323033f4d510ee4ff9f0d4be1d71d63372c1c15d0c9001f041",
	//"vite_7a10c045903421df0915dd8d5807d37d2dff343df961365ef7",
	//"vite_c84d89b7adabaac7d86bd318f988ea39568a90eeafd3eece18",
	//"vite_8c52581f7498350be103b0f0ea20a4f66fde1193bd6668d04e",
}

func conv(mems []string) []types.Address {
	addressArr := make([]types.Address, len(mems))
	for i, v := range mems {
		addressArr[i], _ = types.HexToAddress(v)
	}
	return addressArr
}

type SubscribeMem struct {
	Mem    types.Address
	Notify chan time.Time
}

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(a types.Address, data []byte) (signedData, pubkey []byte, err error)

// update committee result
type Committee struct {
	types.LifecycleStatus
	interval     int32
	memberCnt    int32
	teller       *teller
	subscribeMem *SubscribeMem
	signer       types.Address
	signerFn     SignerFn
}

func (self *Committee) Seal() error {
	return nil
}
func (self *Committee) Authorize(signer types.Address, fn SignerFn) {
	self.signer = signer
	self.signerFn = fn
}

func (self *Committee) Verify(reader SnapshotReader, header *ledger.SnapshotBlock) (bool, error) {
	result, err := self.verifyProducer(header)
	if result && err == nil {
		return true, nil
	}
	return false, nil
}

func (self *Committee) verifyProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult := self.teller.electionTime(time.Unix(int64(header.Timestamp), 0))

	for _, plan := range electionResult.plans {
		if plan.member == *header.Producer {
			if uint64(plan.sTime.Unix()) == header.Timestamp {
				return true, nil
			} else {
				return false, nil
			}
		}
	}
	return false, nil
}

func NewCommittee(genesisTime time.Time, interval int32, memberCnt int32) *Committee {
	committee := &Committee{interval: interval, memberCnt: memberCnt}
	committee.teller = newTeller(genesisTime, interval, memberCnt)
	return committee
}

func (self *Committee) Init() {
	self.PreInit()
	defer self.PostInit()
}

func (self *Committee) Start() {
	self.PreStart()
	defer self.PostStart()

	go self.update()
}

func (self *Committee) Stop() {
	self.PreStop()
	defer self.PostStop()
}

func (self *Committee) Subscribe(subscribeMem *SubscribeMem) {
	self.subscribeMem = subscribeMem
}

func (self *Committee) update() {
	log := log15.New("module", "committee")
	var lastIndex int32 = -1
	var lastRemoveTime = time.Now()
	for !self.Stopped() {
		var current *memberPlan = nil
		electionResult := self.teller.electionTime(time.Now())

		if electionResult == nil {
			log.Error("can't get election result. time is " + time.Now().Format(time.RFC3339Nano) + "\".")
			time.Sleep(time.Duration(self.interval) * time.Second)
			// error handle
			continue
		}

		if electionResult.index <= lastIndex {
			time.Sleep(electionResult.eTime.Sub(time.Now()))
			continue
		}
		mem := self.subscribeMem
		if mem == nil {
			time.Sleep(electionResult.eTime.Sub(time.Now()))
			continue
		}

		plans := electionResult.plans
		for _, plan := range plans {
			if plan.member == mem.Mem {
				current = plan
				break
			}
		}

		if current != nil && lastIndex != -1 {
			time.Sleep(current.sTime.Sub(time.Now()))

			// write timeout
			select {
			case mem.Notify <- current.sTime:
			case <-time.After(electionResult.eTime.Sub(time.Now())):
				log.Error("timeout for notify miner. miner time is \"" + current.sTime.Format(time.RFC3339Nano) + "\".")
				continue
			}
			time.Sleep(electionResult.eTime.Sub(time.Now()))
		} else {
			time.Sleep(electionResult.eTime.Sub(time.Now()))
		}
		lastIndex = electionResult.index

		// clear ever hour
		removeTime := time.Now().Add(-time.Hour)
		if lastRemoveTime.Before(removeTime) {
			self.teller.removePrevious(removeTime)
			lastRemoveTime = removeTime
		}

	}
}

type membersInfo struct {
	genesisTime time.Time
	memberCnt   int32
	interval    int32 // unit: second
}

type memberPlan struct {
	sTime  time.Time
	member types.Address
}

type electionResult struct {
	plans []*memberPlan
	sTime time.Time
	eTime time.Time
	index int32
}

func (self *membersInfo) genPlan(index int32, members []types.Address) *electionResult {
	result := electionResult{}
	planInterval := self.interval * self.memberCnt
	var sTime time.Time = self.genesisTime.Add(time.Duration(planInterval*index) * time.Second)
	result.sTime = sTime
	plans := make([]*memberPlan, 0, len(members))
	for _, member := range members {
		plan := memberPlan{sTime: sTime, member: member}
		plans = append(plans, &plan)
		sTime = sTime.Add(time.Duration(self.interval) * time.Second)
	}
	result.eTime = sTime
	result.plans = plans
	result.index = index
	return &result
}

func (self *membersInfo) time2Index(t time.Time) int32 {
	subSec := int64(t.Sub(self.genesisTime).Seconds())
	i := subSec / int64((self.interval * self.memberCnt))
	return int32(i)
}

// Ensure that all nodes get same result
type teller struct {
	info        *membersInfo
	electionHis map[int32]*electionResult
}

func newTeller(genesisTime time.Time, interval int32, memberCnt int32) *teller {
	t := &teller{}
	t.info = &membersInfo{genesisTime: genesisTime, memberCnt: memberCnt, interval: interval}
	t.electionHis = make(map[int32]*electionResult)
	return t
}

func (self *teller) voteResults() ([]types.Address, error) {
	// record vote and elect
	return conv(DefaultMembers), nil
}

func (self *teller) electionIndex(index int32) *electionResult {
	result, ok := self.electionHis[index]
	if ok {
		return result
	} else {
		voteResults, err := self.voteResults()
		if err == nil {
			plans := self.info.genPlan(index, voteResults)
			self.electionHis[index] = plans
			return plans
		}
	}
	return nil
}

func (self *teller) electionTime(t time.Time) *electionResult {
	index := self.info.time2Index(t)
	return self.electionIndex(index)
}
func (self *teller) removePrevious(rtime time.Time) int32 {
	var i int32 = 0
	for k, v := range self.electionHis {
		if v.eTime.Before(rtime) {
			delete(self.electionHis, k)
			i = i + 1
		}
	}
	return i
}
