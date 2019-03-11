package abi

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strings"
	"time"
)

const (
	// Abi of consensus group, register, vote
	jsonConsensusGroup = `
	[
		{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
		{"type":"function","name":"CancelConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"function","name":"ReCreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"registerOfPledge","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeHeight","type":"uint64"}]},
		{"type":"variable","name":"voteOfKeepToken","inputs":[{"name":"keepAmount","type":"uint256"},{"name":"keepToken","type":"tokenId"}]},
		
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"Name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"rewardIndex","type":"uint64"},{"name":"cancelHeight","type":"uint64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"hisName","inputs":[{"name":"name","type":"string"}]},
		
		{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
		{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"voteStatus","inputs":[{"name":"nodeName","type":"string"}]}
	]`

	// Method names and variable names of consensus group
	MethodNameCreateConsensusGroup        = "CreateConsensusGroup"
	MethodNameCancelConsensusGroup        = "CancelConsensusGroup"
	MethodNameReCreateConsensusGroup      = "ReCreateConsensusGroup"
	VariableNameConsensusGroupInfo        = "consensusGroupInfo"
	VariableNameConditionRegisterOfPledge = "registerOfPledge"
	VariableNameConditionVoteOfKeepToken  = "voteOfKeepToken"

	// Method names and variable names of register
	MethodNameRegister           = "Register"
	MethodNameCancelRegister     = "CancelRegister"
	MethodNameReward             = "Reward"
	MethodNameUpdateRegistration = "UpdateRegistration"
	VariableNameRegistration     = "registration"
	VariableNameHisName          = "hisName"

	// Method names and variable names of vote
	MethodNameVote         = "Vote"
	MethodNameCancelVote   = "CancelVote"
	VariableNameVoteStatus = "voteStatus"

	consensusGroupInfoKeySize = types.GidSize
	registerKeySize           = types.HashSize
	registerHisNameKeySize    = 1 + types.GidSize + types.AddressSize
	voteKeySize               = types.GidSize + types.AddressSize
)

var (
	ABIConsensusGroup, _ = abi.JSONToABIContract(strings.NewReader(jsonConsensusGroup))

	hisNameKeyPrefix = []byte{0}
)

// Structs of consensus group
type VariableConditionRegisterOfPledge struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}
type VariableConditionVoteOfKeepToken struct {
	KeepAmount *big.Int
	KeepToken  types.TokenTypeId
}

// Structs of register
type ParamRegister struct {
	Gid      types.Gid
	Name     string
	NodeAddr types.Address
}
type ParamCancelRegister struct {
	Gid  types.Gid
	Name string
}
type ParamReward struct {
	Gid            types.Gid
	Name           string
	BeneficialAddr types.Address
}

// Structs of vote
type ParamVote struct {
	Gid      types.Gid
	NodeName string
}

// Consensus group variable keys
func GetConsensusGroupKey(gid types.Gid) []byte {
	return gid.Bytes()
}
func GetGidFromConsensusGroupKey(key []byte) types.Gid {
	gid, _ := types.BytesToGid(key)
	return gid
}
func isConsensusGroupKey(key []byte) bool {
	return len(key) == consensusGroupInfoKeySize
}

// Register variable keys
func GetRegisterKey(name string, gid types.Gid) []byte {
	return append(gid.Bytes(), types.DataHash([]byte(name)).Bytes()[types.GidSize:]...)
}

func GetHisNameKey(addr types.Address, gid types.Gid) []byte {
	return helper.JoinBytes(hisNameKeyPrefix, addr.Bytes(), gid.Bytes())
}

func IsRegisterKey(key []byte) bool {
	return len(key) == registerKeySize
}

// Vote variable keys
func GetVoteKey(addr types.Address, gid types.Gid) []byte {
	return append(gid.Bytes(), addr.Bytes()...)
}

func isVoteKey(key []byte) bool {
	return len(key) == voteKeySize
}

func GetAddrFromVoteKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[types.GidSize:])
	return addr
}

// Consensus group readers
func NewGid(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Gid {
	return types.DataToGid(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetActiveConsensusGroupList(db StorageDatabase, snapshotHash *types.Hash) []*types.ConsensusGroupInfo {
	defer monitor.LogTime("vm", "GetActiveConsensusGroupList", time.Now())
	iterator := db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, nil, snapshotHash)
	consensusGroupInfoList := make([]*types.ConsensusGroupInfo, 0)
	if iterator == nil {
		return consensusGroupInfoList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if !isConsensusGroupKey(key) {
			continue
		}
		consensusGroupInfo := new(types.ConsensusGroupInfo)
		if err := ABIConsensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value); err == nil {
			if consensusGroupInfo.IsActive() {
				consensusGroupInfo.Gid = GetGidFromConsensusGroupKey(key)
				consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
			}
		}
	}
	return consensusGroupInfoList
}

func GetConsensusGroup(db StorageDatabase, gid types.Gid) *types.ConsensusGroupInfo {
	data := db.GetStorage(&types.AddressConsensusGroup, GetConsensusGroupKey(gid))

	if len(data) > 0 {
		consensusGroupInfo := new(types.ConsensusGroupInfo)
		ABIConsensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo
	}
	return nil
}

// Register readers
func IsActiveRegistration(db StorageDatabase, name string, gid types.Gid) bool {
	if value := db.GetStorage(&types.AddressConsensusGroup, GetRegisterKey(name, gid)); len(value) > 0 {
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
			return registration.IsActive()
		}
	}
	return false
}

func GetCandidateList(db StorageDatabase, gid types.Gid, snapshotHash *types.Hash) []*types.Registration {
	defer monitor.LogTime("vm", "GetCandidateList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, types.SNAPSHOT_GID.Bytes(), snapshotHash)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, gid.Bytes(), snapshotHash)
	}
	registerList := make([]*types.Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if !IsRegisterKey(key) {
			continue
		}
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil && registration.IsActive() {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

func GetRegistrationList(db StorageDatabase, gid types.Gid, pledgeAddr types.Address) []*types.Registration {
	defer monitor.LogTime("vm", "GetRegistrationList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, types.SNAPSHOT_GID.Bytes(), nil)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, gid.Bytes(), nil)
	}
	registerList := make([]*types.Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if !IsRegisterKey(key) {
			continue
		}
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil && registration.PledgeAddr == pledgeAddr {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

func GetRegistration(db StorageDatabase, gid types.Gid, name string) *types.Registration {
	defer monitor.LogTime("vm", "GetRegistration", time.Now())
	value := db.GetStorage(&types.AddressConsensusGroup, GetRegisterKey(name, gid))
	registration := new(types.Registration)
	if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
		return registration
	}
	return nil
}

// Vote readers
func GetVote(db StorageDatabase, gid types.Gid, addr types.Address) *types.VoteInfo {
	defer monitor.LogTime("vm", "GetVote", time.Now())
	data := db.GetStorage(&types.AddressConsensusGroup, GetVoteKey(addr, gid))
	if len(data) > 0 {
		nodeName := new(string)
		ABIConsensusGroup.UnpackVariable(nodeName, VariableNameVoteStatus, data)
		return &types.VoteInfo{addr, *nodeName}
	}
	return nil
}

func GetVoteList(db StorageDatabase, gid types.Gid, snapshotHash *types.Hash) []*types.VoteInfo {
	defer monitor.LogTime("vm", "GetVoteList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, types.SNAPSHOT_GID.Bytes(), snapshotHash)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, gid.Bytes(), snapshotHash)
	}
	voteInfoList := make([]*types.VoteInfo, 0)
	if iterator == nil {
		return voteInfoList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if !isVoteKey(key) {
			continue
		}
		voterAddr := GetAddrFromVoteKey(key)
		nodeName := new(string)
		if err := ABIConsensusGroup.UnpackVariable(nodeName, VariableNameVoteStatus, value); err == nil {
			voteInfoList = append(voteInfoList, &types.VoteInfo{voterAddr, *nodeName})
		}
	}
	return voteInfoList
}
