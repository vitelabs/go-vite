package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type ConsensusGroupApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewConsensusGroupApi(vite *vite.Vite) *ConsensusGroupApi {
	return &ConsensusGroupApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/consensus_group_api"),
	}
}

func (c ConsensusGroupApi) String() string {
	return "ConsensusGroupApi"
}

type CreateConsensusGroupParam struct {
	SelfAddr               types.Address
	Height                 uint64
	PrevHash               types.Hash
	SnapshotHash           types.Hash
	NodeCount              uint8
	Interval               int64
	PerCount               int64
	RandCount              uint8
	RandRank               uint8
	Repeat                 uint16
	CheckLevel             uint8
	CountingTokenId        types.TokenTypeId
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
}

func (c *ConsensusGroupApi) GetConditionRegisterOfPledge(amount *big.Int, tokenId types.TokenTypeId, height uint64) ([]byte, error) {
	return abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, amount, tokenId, height)
}
func (c *ConsensusGroupApi) GetConditionVoteOfDefault() ([]byte, error) {
	return []byte{}, nil
}
func (c *ConsensusGroupApi) GetConditionVoteOfKeepToken(amount *big.Int, tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionVoteOfKeepToken, amount, tokenId)
}
func (c *ConsensusGroupApi) GetCreateConsensusGroupData(param CreateConsensusGroupParam) ([]byte, error) {
	gid := abi.NewGid(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	return abi.ABIConsensusGroup.PackMethod(
		abi.MethodNameCreateConsensusGroup,
		gid,
		param.NodeCount,
		param.Interval,
		param.PerCount,
		param.RandCount,
		param.RandRank,
		param.Repeat,
		param.CheckLevel,
		param.CountingTokenId,
		param.RegisterConditionId,
		param.RegisterConditionParam,
		param.VoteConditionId,
		param.VoteConditionParam)

}
func (c *ConsensusGroupApi) GetCancelConsensusGroupData(gid types.Gid) ([]byte, error) {
	return abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelConsensusGroup, gid)

}
func (c *ConsensusGroupApi) GetReCreateConsensusGroupData(gid types.Gid) ([]byte, error) {
	return abi.ABIConsensusGroup.PackMethod(abi.MethodNameReCreateConsensusGroup, gid)
}

type ConsensusGroup struct {
	Gid                    types.Gid               `json:"gid"`
	NodeCount              uint8                   `json:"nodeCount"`
	Interval               int64                   `json:"interval"`
	PerCount               int64                   `json:"perCount"`
	RandCount              uint8                   `json:"randCount"`
	RandRank               uint8                   `json:"randRank"`
	Repeat                 uint16                  `json:"repeat"`
	CheckLevel             uint8                   `json:"checkLevel"`
	CountingTokenId        types.TokenTypeId       `json:"countingTokenId"`
	RegisterConditionId    uint8                   `json:"registerConditionId"`
	RegisterConditionParam *RegisterConditionParam `json:"registerConditionParam"`
	VoteConditionId        uint8                   `json:"voteConditionId"`
	VoteConditionParam     *VoteConditionParam     `json:"voerConditionParam"`
	Owner                  types.Address           `json:"owner"`
	PledgeAmount           string                  `json:"pledgeAmount"`
	WithdrawHeight         string                  `json:"withdrawHeight"`
}
type RegisterConditionParam struct {
	PledgeAmount string            `json:"pledgeAmount"`
	PledgeToken  types.TokenTypeId `json:"pledgeToken"`
	PledgeHeight string            `json:"pledgeHeight"`
}

type VoteConditionParam struct {
}

func newConsensusGroup(source *types.ConsensusGroupInfo) *ConsensusGroup {
	if source == nil {
		return nil
	}
	target := &ConsensusGroup{
		Gid:                 source.Gid,
		NodeCount:           source.NodeCount,
		Interval:            source.Interval,
		PerCount:            source.PerCount,
		RandCount:           source.RandCount,
		RandRank:            source.RandRank,
		Repeat:              source.Repeat,
		CheckLevel:          source.CheckLevel,
		CountingTokenId:     source.CountingTokenId,
		RegisterConditionId: source.RegisterConditionId,
		VoteConditionId:     source.VoteConditionId,
		Owner:               source.Owner,
		WithdrawHeight:      uint64ToString(source.WithdrawHeight),
	}
	if source.PledgeAmount != nil {
		target.PledgeAmount = *bigIntToString(source.PledgeAmount)
	}
	if param, err := abi.GetRegisterOfPledgeInfo(source.RegisterConditionParam); err == nil {
		target.RegisterConditionParam = &RegisterConditionParam{PledgeAmount: *bigIntToString(param.PledgeAmount),
			PledgeToken:  param.PledgeToken,
			PledgeHeight: uint64ToString(param.PledgeHeight)}
	}
	return target
}

func (c *ConsensusGroupApi) GetConsensusGroupById(gid types.Gid) (*ConsensusGroup, error) {
	snapshotBlock := c.chain.GetLatestSnapshotBlock()
	prevHash, err := getPrevBlockHash(c.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(c.chain, &types.AddressConsensusGroup, &snapshotBlock.Hash, prevHash)
	if err != nil {
		return nil, err
	}
	group, err := abi.GetConsensusGroup(db, gid)
	if err != nil {
		return nil, err
	}
	return newConsensusGroup(group), nil
}

func (c *ConsensusGroupApi) GetConsensusGroupList() ([]*ConsensusGroup, error) {
	snapshotBlock := c.chain.GetLatestSnapshotBlock()
	prevHash, err := getPrevBlockHash(c.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(c.chain, &types.AddressConsensusGroup, &snapshotBlock.Hash, prevHash)
	if err != nil {
		return nil, err
	}
	list, err := abi.GetActiveConsensusGroupList(db)
	if err != nil {
		return nil, err
	}
	resultList := make([]*ConsensusGroup, len(list))
	for i, group := range list {
		resultList[i] = newConsensusGroup(group)
	}
	return resultList, nil
}
