package consensus

import (
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type snapshotCs struct {
	info *core.GroupInfo
	//voteCache map[int32]*electionResult
	//voteCache *lru.Cache
	rw   *chainRw
	algo core.Algo

	log log15.Logger
}

func newSnapshotCs(rw *chainRw, log log15.Logger) *snapshotCs {
	cs := &snapshotCs{}
	cs.rw = rw
	cs.log = log.New("gid", "snapshot")
	info, err := rw.GetMemberInfo(types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	cs.algo = core.NewAlgo(info)
	cs.info = info
	return cs
}
func (self *snapshotCs) time2Index(t time.Time) uint64 {
	return self.info.Time2Index(t)
}

func (self *snapshotCs) index2Time(i uint64) (time.Time, time.Time) {
	sTime := self.info.GenSTime(i)
	eTime := self.info.GenETime(i)
	return sTime, eTime
}

func (self *snapshotCs) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.electionIndex(index)
}

func (self *snapshotCs) electionIndex(index uint64) (*electionResult, error) {
	sTime, voteIndex := self.GenVoteTime(index)

	block, e := self.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}
	// todo
	self.log.Debug(fmt.Sprintf("election index:%d,%s, voteTime:%s", index, block.Hash, sTime))
	seeds := self.rw.GetSeedsBeforeHashH(block)
	seed := core.NewSeedInfo(seeds)
	voteResults, err := self.calVotes(ledger.HashHeight{Hash: block.Hash, Height: block.Height}, seed, voteIndex)
	if err != nil {
		return nil, err
	}

	plans := genElectionResult(self.info, index, voteResults, block)
	return plans, nil
}

func (self *snapshotCs) calVotes(hashH ledger.HashHeight, seed *core.SeedInfo, voteIndex uint64) ([]types.Address, error) {
	// load from cache
	r, ok := self.rw.getSnapshotVoteCache(hashH.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	// record vote
	votes, err := self.rw.CalVotes(self.info, hashH)
	if err != nil {
		return nil, err
	}

	var successRate map[types.Address]int32
	if fork.IsMintFork(hashH.Height) {
		successRate, err = self.rw.GetSuccessRateByHour(voteIndex)
		if err != nil {
			return nil, err
		}
		self.log.Info(fmt.Sprintf("[%d][%d]success rate log: %+v", hashH.Height, voteIndex, successRate))
	}

	context := core.NewVoteAlgoContext(votes, &hashH, successRate, seed)
	// filter size of members
	finalVotes := self.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = self.algo.ShuffleVotes(finalVotes, &hashH, seed)

	result := fmt.Sprintf("CalVotes result: %d:%d:%s, ", voteIndex, hashH.Height, hashH.Hash)
	for _, v := range finalVotes {
		if len(v.Type) > 0 {
			result += fmt.Sprintf("[%s:%+v],", v.Name, v.Type)
		} else {
			result += fmt.Sprintf("[%s],", v.Name)
		}
	}
	self.log.Info(result)
	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	self.rw.updateSnapshotVoteCache(hashH.Hash, address)
	return address, nil
}

// generate the vote time for snapshot consensus group
func (self *snapshotCs) GenVoteTime(idx uint64) (time.Time, uint64) {
	return self.info.GenSTime(idx - 1), idx - 2
}

func (self *snapshotCs) voteDetailsBeforeTime(t time.Time) ([]*VoteDetails, *ledger.HashHeight, error) {
	block, e := self.rw.GetSnapshotBeforeTime(t)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, nil, e
	}

	headH := ledger.HashHeight{Height: block.Height, Hash: block.Hash}
	details, err := self.rw.CalVoteDetails(self.info.Gid, self.info, headH)
	return details, &headH, err
}

func (self *snapshotCs) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult, err := self.electionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (self *snapshotCs) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}
	for _, plan := range result.Plans {
		if plan.Member == address {
			if plan.STime == t {
				return true
			}
		}
	}
	return false
}
