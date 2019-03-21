package consensus

import (
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type contractDposCs struct {
	info *core.GroupInfo
	rw   *chainRw
	algo core.Algo

	log log15.Logger
}

func newContractDposCs(info *core.GroupInfo, rw *chainRw, log log15.Logger) *contractDposCs {
	cs := &contractDposCs{}
	cs.rw = rw
	cs.log = log.New("gid", fmt.Sprintf("contract-%s", info.Gid.String()))
	// todo
	return cs
}

func (self *contractDposCs) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.electionIndex(index)
}

func (self *contractDposCs) electionIndex(index uint64) (*electionResult, error) {
	sTime := self.GenVoteTime(index, self.info)

	block, e := self.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}
	// todo
	self.log.Debug(fmt.Sprintf("election index:%d,%s, voteTime:%s", index, block.Hash, sTime))
	seeds, err := self.rw.GetSeedsByHashH(block, seedDuration, 25)
	if err != nil {
		return nil, err
	}
	seed := core.NewSeedInfo(seeds)
	voteResults, err := self.calVotes(ledger.HashHeight{Hash: block.Hash, Height: block.Height}, seed)
	if err != nil {
		return nil, err
	}

	plans := genElectionResult(self.info, index, voteResults, block)
	return plans, nil
}

func (self *contractDposCs) calVotes(hashH ledger.HashHeight, seed *core.SeedInfo) ([]types.Address, error) {
	// load from cache
	r, ok := self.rw.getContractVoteCache(hashH.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	// record vote
	votes, err := self.rw.CalVotes(self.info, hashH)
	if err != nil {
		return nil, err
	}

	context := core.NewVoteAlgoContext(votes, &hashH, nil, seed)
	// filter size of members
	finalVotes := self.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = self.algo.ShuffleVotes(finalVotes, &hashH, seed)

	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	self.rw.updateContractVoteCache(hashH.Hash, address)
	return address, nil
}

// generate the vote time for account consensus group
func (self *contractDposCs) GenVoteTime(idx uint64, info *core.GroupInfo) time.Time {
	sTime := info.GenSTime(idx)
	return sTime.Add(time.Second * 75)
}
