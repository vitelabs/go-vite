package core

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type stateCh interface {
	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                     // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                                  // Get register for consensus group
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                           // Get the candidate's vote
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
}

func CalVotes(info *GroupInfo, block ledger.HashHeight, rw stateCh) ([]*Vote, error) {
	// query register info
	registerList, _ := rw.GetRegisterList(block.Hash, info.Gid)
	// query vote info
	votes, _ := rw.GetVoteMap(block.Hash, info.Gid)

	var registers []*Vote

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, GenVote(block.Hash, v, votes, info.CountingTokenId, rw))
	}
	return registers, nil
}
func GenVote(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId, rw stateCh) *Vote {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := rw.GetBalanceList(snapshotHash, id, addrs)

	result := &Vote{Balance: big.NewInt(0), Name: registration.Name, Addr: registration.NodeAddr}
	for _, v := range balanceMap {
		result.Balance.Add(result.Balance, v)
	}
	return result
}
