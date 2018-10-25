package consensus

import (
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
)

type ch interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo                                                 // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration                                              // Get register for consensus group
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo                                                       // Get the candidate's vote
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) map[types.Address]*big.Int // Get balance for addressList
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error)
}
type chainRw struct {
	rw ch
}

type Vote struct {
	name    string
	addr    types.Address
	balance *big.Int
}
type VoteDetails struct {
	name string
	addr map[types.Address]*big.Int
}

func (self *chainRw) GetSnapshotBeforeTime(t time.Time) (*ledger.SnapshotBlock, error) {
	block, e := self.rw.GetSnapshotBlockBeforeTime(&t)

	if e != nil {
		return nil, e
	}

	if block == nil {
		return nil, errors.New("before time[" + t.String() + "] block not exist")
	}
	return block, nil
}

func (self *chainRw) CalVotes(gid types.Gid, info *membersInfo, block ledger.HashHeight) ([]*Vote, error) {

	// query register info
	registerList := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*Vote

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVote(block.Hash, v, votes, info.countingTokenId))
	}
	return registers, nil
}
func (self *chainRw) GenVote(snapshotHash types.Hash, registration *contracts.Registration, infos []*contracts.VoteInfo, id types.TokenTypeId) *Vote {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap := self.rw.GetBalanceList(snapshotHash, id, addrs)

	result := &Vote{balance: big.NewInt(0), name: registration.Name, addr: registration.NodeAddr}
	for _, v := range balanceMap {
		result.balance.Add(result.balance, v)
	}
	return result
}
func (self *chainRw) CalVoteDetails(gid types.Gid, info *membersInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVoteDetails(block.Hash, v, votes, info.countingTokenId))
	}
	return registers, nil
}
func (self *chainRw) GenVoteDetails(snapshotHash types.Hash, registration *contracts.Registration, infos []*contracts.VoteInfo, id types.TokenTypeId) *VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap := self.rw.GetBalanceList(snapshotHash, id, addrs)
	return &VoteDetails{name: registration.Name, addr: balanceMap}
}
func (self *chainRw) GetMemberInfo(gid types.Gid, genesis time.Time) *membersInfo {
	// todo consensus group maybe change ??
	var result *membersInfo
	head := self.rw.GetLatestSnapshotBlock()
	consensusGroupList := self.rw.GetConsensusGroupList(head.Hash)
	for _, v := range consensusGroupList {
		if v.Gid == gid {
			result = &membersInfo{
				genesisTime:     genesis,
				interval:        int32(v.Interval),
				memberCnt:       int32(v.NodeCount),
				seed:            new(big.Int).SetBytes(v.Gid.Bytes()),
				perCnt:          int32(v.PerCount),
				randCnt:         int32(v.RandCount),
				randRange:       int32(v.RandRank),
				countingTokenId: v.CountingTokenId,
			}
		}
	}

	return result
}

func (self *chainRw) getGid(block *ledger.AccountBlock) (types.Gid, error) {
	gid, e := self.rw.GetContractGidByAccountBlock(block)
	return *gid, e
}
func (self *chainRw) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return self.rw.GetLatestSnapshotBlock()
}
