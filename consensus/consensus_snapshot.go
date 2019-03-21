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

func newSnapshotCs(info *core.GroupInfo, rw *chainRw, log log15.Logger, cacheDb Cache) *snapshotCs {
	cs := &snapshotCs{}
	cs.rw = rw
	cs.log = log.New("gid", "snapshot")
	// todo
	return cs
}

func (self *snapshotCs) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.electionIndex(index)
}

func (self *snapshotCs) electionIndex(index uint64) (*electionResult, error) {
	sTime, voteIndex := self.GenVoteTime(index, self.info)

	block, e := self.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}
	// todo
	self.log.Debug(fmt.Sprintf("election index:%d,%s, voteTime:%s", index, block.Hash, sTime))
	seeds, err := self.rw.GetSeedsBeforeHashH(block, seedDuration)
	if err != nil {
		return nil, err
	}
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
func (self *snapshotCs) GenVoteTime(idx uint64, info *core.GroupInfo) (time.Time, uint64) {
	return info.GenSTime(idx), idx - 1
}
