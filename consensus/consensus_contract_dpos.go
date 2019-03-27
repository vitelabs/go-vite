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
	cs.info = info
	cs.algo = core.NewAlgo(info)
	cs.log = log.New("gid", fmt.Sprintf("contract-%s", info.Gid.String()))
	return cs
}

func (self *contractDposCs) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.electionIndex(index)
}

func (self *contractDposCs) electionIndex(index uint64) (*electionResult, error) {
	voteResults, block, err := self.electionAddrsIndex(index)
	if err != nil {
		return nil, err
	}
	plans := genElectionResult(self.info, index, voteResults, block)
	return plans, nil
}

func (self *contractDposCs) electionAddrsIndex(index uint64) ([]types.Address, *ledger.SnapshotBlock, error) {
	sTime := self.GenVoteTime(index)

	block, e := self.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, nil, e
	}

	voteResults, err := self.calVotes(block)
	if err != nil {
		return nil, nil, err
	}
	return voteResults, block, nil
}

func (self *contractDposCs) calVotes(block *ledger.SnapshotBlock) ([]types.Address, error) {
	// load from cache
	r, ok := self.rw.getContractVoteCache(block.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	hashH := ledger.HashHeight{Hash: block.Hash, Height: block.Height}
	// record vote
	votes, err := self.rw.CalVotes(self.info, hashH)
	if err != nil {
		return nil, err
	}

	randomSeed := self.rw.GetSeedsBeforeHashH(block)
	seed := core.NewSeedInfo(randomSeed)

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
func (self *contractDposCs) GenVoteTime(idx uint64) time.Time {
	sTime := self.info.GenSTime(idx)
	return sTime.Add(-time.Second * 75)
}

func (self *contractDposCs) time2Index(t time.Time) uint64 {
	return self.info.Time2Index(t)
}

func (self *contractDposCs) verifyAccountsProducer(accountBlocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	head := self.rw.GetLatestSnapshotBlock()

	index := self.time2Index(*head.Timestamp)
	result, _, err := self.electionAddrsIndex(index)
	if err != nil {
		return nil, err
	}
	resultM := make(map[types.Address]bool)
	for _, v := range result {
		resultM[v] = true
	}
	return self.verifyProducers(accountBlocks, resultM), nil
}

func (self *contractDposCs) verifyProducers(blocks []*ledger.AccountBlock, result map[types.Address]bool) []*ledger.AccountBlock {
	var inValid []*ledger.AccountBlock
	for _, v := range blocks {
		if !result[v.AccountAddress] {
			inValid = append(inValid, v)
		}
	}
	return inValid
}

func (self *contractDposCs) VerifyAccountProducer(accountBlock *ledger.AccountBlock) (bool, error) {
	head := self.rw.GetLatestSnapshotBlock()
	electionResult, err := self.electionTime(*head.Timestamp)
	if err != nil {
		return false, err
	}
	return self.verifyProducer(accountBlock.Producer(), electionResult), nil
}

func (self *contractDposCs) verifyProducer(address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}

	for _, plan := range result.Plans {
		if plan.Member == address {
			if self.info.CheckLevel == 1 {
				return true
			}
		}
	}
	return false
}
