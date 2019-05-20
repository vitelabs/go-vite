package core

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type stateCh interface {
	//GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                    // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                              // Get register for consensus group
	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                      // Get the candidate's vote
	GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
}

func CalVotes(info types.ConsensusGroupInfo, hash types.Hash, rw stateCh) ([]*Vote, error) {
	t0 := time.Now()
	// query register info
	registerList, err := rw.GetRegisterList(hash, info.Gid)
	if err != nil {
		return nil, err
	}

	log15.Info("[doublejie]GetRegisterList time duration", "diff", time.Now().Sub(t0))
	t1 := time.Now()
	// query vote info
	votes, err := rw.GetVoteList(hash, info.Gid)
	if err != nil {
		return nil, err
	}
	log15.Info("[doublejie]GetVoteList time duration", "diff", time.Now().Sub(t1))

	var registers []*Vote

	t2 := time.Now()
	// cal candidate
	for _, v := range registerList {
		register := &Vote{Balance: big.NewInt(0), Name: v.Name, Addr: v.NodeAddr}
		err := voteCompleting(hash, register, votes, info.CountingTokenId, rw)
		if err != nil {
			return nil, err
		}

		registers = append(registers, register)
	}

	log15.Info("[doublejie]voteCompleting time duration", "diff", time.Now().Sub(t2))
	return registers, nil
}
func voteCompleting(snapshotHash types.Hash, vote *Vote, infos []*types.VoteInfo, id types.TokenTypeId, rw stateCh) error {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == vote.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	if len(addrs) > 0 {
		balanceMap, err := rw.GetConfirmedBalanceList(addrs, id, snapshotHash)
		if err != nil {
			return err
		}
		for _, v := range balanceMap {
			vote.Balance.Add(vote.Balance, v)
		}
	}
	return nil
}
