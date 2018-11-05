package consensus

import (
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type ch interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                     // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                                  // Get register for consensus group
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                           // Get the candidate's vote
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
	GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
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
	Name         string
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
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
	registerList, _ := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*Vote

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVote(block.Hash, v, votes, info.countingTokenId))
	}
	return registers, nil
}
func (self *chainRw) GenVote(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *Vote {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := self.rw.GetBalanceList(snapshotHash, id, addrs)

	result := &Vote{balance: big.NewInt(0), name: registration.Name, addr: registration.NodeAddr}
	for _, v := range balanceMap {
		result.balance.Add(result.balance, v)
	}
	return result
}
func (self *chainRw) CalVoteDetails(gid types.Gid, info *membersInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList, _ := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVoteDetails(block.Hash, v, votes, info.countingTokenId))
	}
	return registers, nil
}
func (self *chainRw) GenVoteDetails(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := self.rw.GetBalanceList(snapshotHash, id, addrs)
	return &VoteDetails{Name: registration.Name, Addr: balanceMap, CurrentAddr: registration.NodeAddr, RegisterList: registration.HisAddrList}
}
func (self *chainRw) GetMemberInfo(gid types.Gid, genesis time.Time) *membersInfo {
	// todo consensus group maybe change ??
	var result *membersInfo
	head := self.rw.GetLatestSnapshotBlock()
	consensusGroupList, _ := self.rw.GetConsensusGroupList(head.Hash)
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
func (self *chainRw) checkSnapshotHashValid(startHeight uint64, startHash types.Hash, actual types.Hash) error {
	if startHash == actual {
		return nil
	}
	startB, e := self.rw.GetSnapshotBlockByHash(&startHash)
	if e != nil {
		return e
	}
	if startB == nil {
		return errors.Errorf("start snapshot block is nil. hashH:%s-%d", startHash, startHeight)
	}
	actualB, e := self.rw.GetSnapshotBlockByHash(&actual)
	if e != nil {
		return e
	}
	if actualB == nil {
		return errors.Errorf("refer snapshot block is nil. hashH:%s", actual)
	}
	if actualB.Height < startB.Height {
		return errors.Errorf("refer snapshot block height must >= start snapshot block height")
	}
	return nil
}
