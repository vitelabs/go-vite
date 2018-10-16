package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
)

type ContractsApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewContractsApi(vite *vite.Vite) *ContractsApi {
	return &ContractsApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/contracts_api"),
	}
}

func (c ContractsApi) String() string {
	return "ContractsApi"
}

func (c *ContractsApi) GetPledgeData(beneficialAddr types.Address) ([]byte, error) {
	return contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, beneficialAddr)
}

func (c *ContractsApi) GetCancelPledgeData(beneficialAddr types.Address, amount *big.Int) ([]byte, error) {
	return contracts.ABIPledge.PackMethod(contracts.MethodNameCancelPledge, beneficialAddr, amount)
}

type MintageParams struct {
	SelfAddr     types.Address
	Height       uint64
	PrevHash     types.Hash
	SnapshotHash types.Hash
	TokenName    string
	TokenSymbol  string
	TotalSupply  *big.Int
	Decimals     uint8
}

func (c *ContractsApi) GetMintageData(param MintageParams) ([]byte, error) {
	tokenId := contracts.NewTokenId(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	return contracts.ABIMintage.PackMethod(contracts.MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals)
}
func (c *ContractsApi) GetMintageCancelPledgeData(tokenId types.TokenTypeId) ([]byte, error) {
	return contracts.ABIMintage.PackMethod(contracts.MethodNameMintageCancelPledge, tokenId)
}

func (c *ContractsApi) GetCreateContractToAddress(selfAddr types.Address, height uint64, prevHash types.Hash, snapshotHash types.Hash) types.Address {
	return contracts.NewContractAddress(selfAddr, height, prevHash, snapshotHash)
}

func (c *ContractsApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address, beneficialAddr types.Address, publicKey []byte, signature []byte) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameRegister, gid, name, nodeAddr, beneficialAddr, publicKey, signature)
}
func (c *ContractsApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister, gid, name)
}
func (c *ContractsApi) GetRewardData(gid types.Gid, name string, endHeight uint64, startHeight uint64, rewardAmount *big.Int) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameReward, gid, name, endHeight, startHeight, rewardAmount)
}
func (c *ContractsApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address, beneficialAddr types.Address, publicKey []byte, signature []byte) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration, gid, name, nodeAddr, beneficialAddr, publicKey, signature)
}

func (c *ContractsApi) GetVoteData(gid types.Gid, name string) ([]byte, error) {
	return contracts.ABIVote.PackMethod(contracts.MethodNameVote, gid, name)
}
func (c *ContractsApi) GetCancelVoteData(gid types.Gid) ([]byte, error) {
	return contracts.ABIVote.PackMethod(contracts.MethodNameCancelVote, gid)
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
	CountingTokenId        types.TokenTypeId
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
}

func (c *ContractsApi) GetConditionRegisterOfPledge(amount *big.Int, tokenId types.TokenTypeId, height uint64) ([]byte, error) {
	return contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionRegisterOfPledge, amount, tokenId, height)
}
func (c *ContractsApi) GetConditionVoteOfDefault() ([]byte, error) {
	return []byte{}, nil
}
func (c *ContractsApi) GetConditionVoteOfKeepToken(amount *big.Int, tokenId types.TokenTypeId) ([]byte, error) {
	return contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionVoteOfKeepToken, amount, tokenId)

}
func (c *ContractsApi) GetCreateConsensusGroupData(param CreateConsensusGroupParam) ([]byte, error) {
	gid := contracts.NewGid(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	return contracts.ABIConsensusGroup.PackMethod(
		contracts.MethodNameCreateConsensusGroup,
		gid,
		param.NodeCount,
		param.Interval,
		param.PerCount,
		param.RandCount,
		param.RandRank,
		param.CountingTokenId,
		param.RegisterConditionId,
		param.RegisterConditionParam,
		param.VoteConditionId,
		param.VoteConditionParam)

}
func (c *ContractsApi) GetCancelConsensusGroupData(gid types.Gid) ([]byte, error) {
	return contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCancelConsensusGroup, gid)

}
func (c *ContractsApi) GetReCreateConsensusGroupData(gid types.Gid) ([]byte, error) {
	return contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameReCreateConsensusGroup, gid)
}
