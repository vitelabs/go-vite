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
	core.GroupInfo
	rw   *chainRw
	algo core.Algo

	log log15.Logger
}

func (contract *contractDposCs) GetInfo() *core.GroupInfo {
	return &contract.GroupInfo
}

func newContractDposCs(info *core.GroupInfo, rw *chainRw, log log15.Logger) *contractDposCs {
	cs := &contractDposCs{}
	cs.rw = rw
	cs.GroupInfo = *info
	cs.algo = core.NewAlgo(info)
	cs.log = log.New("gid", fmt.Sprintf("contract-%s", info.Gid.String()))
	return cs
}

func (contract *contractDposCs) electionTime(t time.Time) (*electionResult, error) {
	index := contract.Time2Index(t)
	return contract.ElectionIndex(index)
}

func (contract *contractDposCs) ElectionIndex(index uint64) (*electionResult, error) {
	voteResults, _, err := contract.electionAddrsIndex(index)
	if err != nil {
		return nil, err
	}
	plans := genElectionResult(&contract.GroupInfo, index, voteResults)
	return plans, nil
}

func (contract *contractDposCs) electionAddrsIndex(index uint64) ([]types.Address, *ledger.SnapshotBlock, error) {
	sTime := contract.GenProofTime(index)

	block, e := contract.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		contract.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, nil, e
	}

	voteResults, err := contract.calVotes(block)
	if err != nil {
		return nil, nil, err
	}
	return voteResults, block, nil
}

func (contract *contractDposCs) calVotes(block *ledger.SnapshotBlock) ([]types.Address, error) {
	// load from cache
	r, ok := contract.rw.getVoteLRUCache(contract.Gid, block.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	hashH := ledger.HashHeight{Hash: block.Hash, Height: block.Height}
	// record vote
	votes, err := contract.rw.CalVotes(&contract.GroupInfo, hashH)
	if err != nil {
		return nil, err
	}

	randomSeed := contract.rw.GetSeedsBeforeHashH(block.Hash)
	seed := core.NewSeedInfo(randomSeed)

	context := core.NewVoteAlgoContext(votes, &hashH, nil, seed)
	// filter size of members
	finalVotes := contract.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = contract.algo.ShuffleVotes(finalVotes, &hashH, seed)

	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	contract.rw.updateVoteLRUCache(contract.Gid, hashH.Hash, address)
	return address, nil
}

// generate the vote time for account consensus group
func (contract *contractDposCs) GenProofTime(idx uint64) time.Time {
	sTime, _ := contract.Index2Time(idx)
	sTime = sTime.Add(-time.Second * 75)
	// if before genesis'time, just use genesis'time + 1s
	if sTime.Before(contract.GenesisTime) {
		return contract.GenesisTime.Add(time.Second)
	}
	return sTime
}

func (contract *contractDposCs) verifyAccountsProducer(accountBlocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	head := contract.rw.GetLatestSnapshotBlock()

	index := contract.Time2Index(*head.Timestamp)
	result, _, err := contract.electionAddrsIndex(index)
	if err != nil {
		return nil, err
	}
	resultM := make(map[types.Address]bool)
	for _, v := range result {
		resultM[v] = true
	}
	return contract.verifyProducers(accountBlocks, resultM), nil
}

func (contract *contractDposCs) verifyProducers(blocks []*ledger.AccountBlock, result map[types.Address]bool) []*ledger.AccountBlock {
	var inValid []*ledger.AccountBlock
	for _, v := range blocks {
		if !result[v.Producer()] {
			inValid = append(inValid, v)
		}
	}
	return inValid
}

func (contract *contractDposCs) VerifyAccountProducer(accountBlock *ledger.AccountBlock) (bool, error) {
	if contract.CheckLevel == 1 {
		return contract.VerifyAccountProducerLevel1(accountBlock)
	}
	head := contract.rw.GetLatestSnapshotBlock()
	electionResult, err := contract.electionTime(*head.Timestamp)
	if err != nil {
		return false, err
	}

	return contract.verifyProducer(accountBlock.Producer(), electionResult), nil
}

func (contract *contractDposCs) VerifyAccountProducerLevel1(accBlock *ledger.AccountBlock) (bool, error) {
	head := contract.rw.GetLatestSnapshotBlock()
	index := contract.Time2Index(*head.Timestamp)
	addrs, _, err := contract.electionAddrsIndex(index)
	if err != nil {
		return false, err
	}
	for _, v := range addrs {
		if v == accBlock.Producer() {
			return true, nil
		}
	}
	return false, nil
}

func (contract *contractDposCs) VerifyProducer(address types.Address, t time.Time) (bool, error) {
	electionResult, err := contract.electionTime(t)
	if err != nil {
		return false, err
	}

	return contract.verifyProducer(address, electionResult), nil
}

func (contract *contractDposCs) verifyProducer(address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}

	for _, plan := range result.Plans {
		if plan.Member == address {
			if contract.CheckLevel == 1 {
				return true
			}
		}
	}
	return false
}
