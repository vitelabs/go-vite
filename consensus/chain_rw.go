package consensus

import (
	"math/big"

	"time"

	"strconv"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
)

type A interface {
	headSnapshot() *ledger.SnapshotBlock
	GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo                                                 // 获取所有的共识组
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration                                              // 获取共识组下的参与竞选的候选人
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo                                                       // 获取候选人的投票
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) map[types.Address]*big.Int // 获取所有用户的余额
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)
}
type chainRw struct {
}

func (self *chainRw) headSnapshot() *ledger.SnapshotBlock {
	panic("implement me")
}

func (self *chainRw) GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo {
	panic("implement me")
}

func (self *chainRw) GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration {
	panic("implement me")
}

func (self *chainRw) GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo {
	panic("implement me")
}

func (self *chainRw) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) map[types.Address]*big.Int {
	panic("implement me")
}
func (self *chainRw) GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

type Vote struct {
	name    string
	addr    types.Address
	balance *big.Int
}

func (self *chainRw) CalVotes(gid types.Gid, t time.Time) ([]*Vote, *HashHeight, error) {
	block, e := self.GetSnapshotBlockBeforeTime(&t)

	if e != nil {
		return nil, nil, e
	}

	if block == nil {
		return nil, nil, errors.New("before time[" + t.String() + "] block not exist")
	}
	head := self.headSnapshot()

	if block.Height > head.Height {
		return nil, nil, errors.New("rollback happened, block height[" + strconv.FormatUint(block.Height, 10) + "], head height[" + strconv.FormatUint(head.Height, 10) + "]")
	}
	// query register info
	registerList := self.GetRegisterList(head.Hash, gid)
	// query vote info
	votes := self.GetVoteMap(head.Hash, gid)

	voteMap := toMap(votes)

	var registers []*Vote

	// cal candidate
	for _, v := range registerList {
		_, ok := voteMap[v.Name]
		if ok {
			registers = append(registers, self.GenVote(head.SnapshotHash, v, votes))
		}
	}
	return registers, &HashHeight{Height: head.Height, Hash: head.Hash}, nil
}
func (self *chainRw) GenVote(snapshotHash types.Hash, registration *contracts.Registration, infos []*contracts.VoteInfo) *Vote {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap := self.GetBalanceList(snapshotHash, ledger.ViteTokenId, addrs)

	result := &Vote{balance: big.NewInt(0), name: registration.Name, addr: registration.NodeAddr}
	for _, v := range balanceMap {
		result.balance.Add(result.balance, v)
	}
	return result
}
func (rw *chainRw) GetMemberInfo(gid types.Gid, genesis time.Time) *membersInfo {
	return &membersInfo{}
}
