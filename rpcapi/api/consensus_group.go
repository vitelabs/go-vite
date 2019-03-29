package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
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
