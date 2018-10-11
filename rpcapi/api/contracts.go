package api

import (
	"encoding/hex"
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

func (c *ContractsApi) GetPledgeData(beneficialAddr types.Address) (string, error) {
	c.log.Info("GetPledgeData")
	data, err := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, beneficialAddr)
	return hex.EncodeToString(data), err
}

func (c *ContractsApi) GetCancelPledgeData(beneficialAddr types.Address, amount *big.Int) (string, error) {
	c.log.Info("GetCancelPledgeData")
	data, err := contracts.ABIPledge.PackMethod(contracts.MethodNameCancelPledge, beneficialAddr, amount)
	return hex.EncodeToString(data), err
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

func (c *ContractsApi) GetMintageData(param MintageParams) (string, error) {
	c.log.Info("GetMintageData")
	tokenId := contracts.NewTokenId(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	data, err := contracts.ABIMintage.PackMethod(contracts.MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetMintageCancelPledgeData(tokenId types.TokenTypeId) (string, error) {
	c.log.Info("GetMintageCancelPledgeData")
	data, err := contracts.ABIMintage.PackMethod(contracts.MethodNameMintageCancelPledge, tokenId)
	return hex.EncodeToString(data), err
}

func (c *ContractsApi) GetCreateContractToAddress(selfAddr types.Address, height uint64, prevHash types.Hash, snapshotHash types.Hash) types.Address {
	c.log.Info("GetCreateContractToAddress")
	return contracts.NewContractAddress(selfAddr, height, prevHash, snapshotHash)
}

func (c *ContractsApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address, beneficialAddr types.Address) (string, error) {
	c.log.Info("GetRegisterData")
	data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameRegister, gid, name, nodeAddr, beneficialAddr)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetCancelRegisterData(gid types.Gid, name string) (string, error) {
	c.log.Info("GetCancelRegisterData")
	data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister, gid, name)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetRewardData(gid types.Gid, name string, endHeight uint64, startHeight uint64, rewardAmount *big.Int) (string, error) {
	c.log.Info("GetRewardData")
	data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameReward, gid, name, endHeight, startHeight, rewardAmount)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address, beneficialAddr types.Address) (string, error) {
	c.log.Info("GetUpdateRegistrationData")
	data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration, gid, name, nodeAddr, beneficialAddr)
	return hex.EncodeToString(data), err
}

func (c *ContractsApi) GetVoteData(gid types.Gid, name string) (string, error) {
	c.log.Info("GetVoteData")
	data, err := contracts.ABIVote.PackMethod(contracts.MethodNameVote, gid, name)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetCancelVoteData(gid types.Gid) (string, error) {
	c.log.Info("GetCancelVoteData")
	data, err := contracts.ABIVote.PackMethod(contracts.MethodNameCancelVote, gid)
	return hex.EncodeToString(data), err
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

func (c *ContractsApi) GetConditionRegisterOfPledge(amount *big.Int, tokenId types.TokenTypeId, height uint64) (string, error) {
	c.log.Info("GetConditionRegisterOfPledge")
	data, err := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionRegisterOfPledge, amount, tokenId, height)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetConditionVoteOfDefault() (string, error) {
	c.log.Info("GetConditionVoteOfDefault")
	return "", nil
}
func (c *ContractsApi) GetConditionVoteOfKeepToken(amount *big.Int, tokenId types.TokenTypeId) (string, error) {
	c.log.Info("GetConditionVoteOfKeepToken")
	data, err := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionVoteOfKeepToken, amount, tokenId)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetCreateConsensusGroupData(param CreateConsensusGroupParam) (string, error) {
	c.log.Info("GetCreateConsensusGroupData")
	gid := contracts.NewGid(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	data, err := contracts.ABIConsensusGroup.PackMethod(
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
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetCancelConsensusGroupData(gid types.Gid) (string, error) {
	c.log.Info("GetCancelConsensusGroupData")
	data, err := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCancelConsensusGroup, gid)
	return hex.EncodeToString(data), err
}
func (c *ContractsApi) GetReCreateConsensusGroupData(gid types.Gid) (string, error) {
	c.log.Info("GetCancelConsensusGroupData")
	data, err := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameReCreateConsensusGroup, gid)
	return hex.EncodeToString(data), err
}
