package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
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

func newConsensusGroup(source *types.ConsensusGroupInfo, sbHeight uint64) *ConsensusGroup {
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
		WithdrawHeight:      Uint64ToString(source.ExpirationHeight),
	}
	if source.StakeAmount != nil {
		target.PledgeAmount = *bigIntToString(source.StakeAmount)
	}
	if param, err := abi.GetRegisterStakeParamOfConsensusGroup(source.RegisterConditionParam); err == nil {
		target.RegisterConditionParam = &RegisterConditionParam{
			PledgeToken:  param.StakeToken,
			PledgeHeight: Uint64ToString(param.StakeHeight)}
		target.RegisterConditionParam.PledgeAmount = *bigIntToString(contracts.SbpStakeAmountMainnet)
	}
	return target
}

func (c *ConsensusGroupApi) GetConsensusGroupById(gid types.Gid) (*ConsensusGroup, error) {
	db, err := getVmDb(c.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	group, err := abi.GetConsensusGroup(db, gid)
	if err != nil {
		return nil, err
	}
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	return newConsensusGroup(group, sb.Height), nil
}

func (c *ConsensusGroupApi) GetConsensusGroupList() ([]*ConsensusGroup, error) {
	db, err := getVmDb(c.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	list, err := abi.GetConsensusGroupList(db)
	if err != nil {
		return nil, err
	}
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	resultList := make([]*ConsensusGroup, len(list))
	for i, group := range list {
		resultList[i] = newConsensusGroup(group, sb.Height)
	}
	return resultList, nil
}
