package consensus

import (
	"math/big"
	"time"

	"sort"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
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

type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}
type ByBalance []*VoteDetails

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[i].Balance.Cmp(a[j].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	} else {
		return r < 0
	}
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

func (self *chainRw) CalVotes(info *core.GroupInfo, block ledger.HashHeight) ([]*core.Vote, error) {
	return core.CalVotes(info, block, self.rw)
}

func (self *chainRw) CalVoteDetails(gid types.Gid, info *core.GroupInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList, _ := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVoteDetails(block.Hash, v, votes, info.CountingTokenId))
	}
	sort.Sort(ByBalance(registers))
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
	balanceTotal := big.NewInt(0)
	for _, v := range balanceMap {
		balanceTotal.Add(balanceTotal, v)
	}
	return &VoteDetails{
		Vote: core.Vote{
			Name:    registration.Name,
			Addr:    registration.NodeAddr,
			Balance: balanceTotal,
		},
		CurrentAddr:  registration.NodeAddr,
		RegisterList: registration.HisAddrList,
		Addr:         balanceMap,
	}
}

func (self *chainRw) GetMemberInfo(gid types.Gid, genesis time.Time) *core.GroupInfo {
	// todo consensus group maybe change ??
	var result *core.GroupInfo
	head := self.rw.GetLatestSnapshotBlock()
	consensusGroupList, _ := self.rw.GetConsensusGroupList(head.Hash)
	for _, v := range consensusGroupList {
		if v.Gid == gid {
			result = core.NewGroupInfo(genesis, *v)
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
