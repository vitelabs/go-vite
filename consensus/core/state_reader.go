package core

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type stateCh interface {
	//GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                    // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                              // Get register for consensus group
	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                      // Get the candidate's vote
	GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
}

func CalVotes(info *GroupInfo, hash types.Hash, rw stateCh) ([]*Vote, error) {
	// query register info
	registerList, err := rw.GetRegisterList(hash, info.Gid)
	if err != nil {
		return nil, err
	}
	// query vote info
	votes, err := rw.GetVoteList(hash, info.Gid)
	if err != nil {
		return nil, err
	}

	var registers []*Vote

	// cal candidate
	for _, v := range registerList {
		register := &Vote{Balance: big.NewInt(0), Name: v.Name, Addr: v.NodeAddr}
		err := voteCompleting(hash, register, votes, info.CountingTokenId, rw)
		if err != nil {
			return nil, err
		}

		registers = append(registers, register)
	}
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
