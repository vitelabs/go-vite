package abi

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strings"
	"time"
)

const (
	// Abi of consensus group, register, vote
	jsonConsensusGroup = `
	[
		{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"repeat","type":"uint16"},{"name":"checkLevel","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
		{"type":"function","name":"CancelConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"function","name":"ReCreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"repeat","type":"uint16"},{"name":"checkLevel","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"registerOfPledge","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeHeight","type":"uint64"}]},
		
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"Name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"rewardTime","type":"int64"},{"name":"cancelTime","type":"int64"},{"name":"hisAddrList","type":"address[]"}]},
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

	consensusGroupInfoKeySize = 1 + types.GidSize
	registerKeySize           = types.HashSize
	registerHisNameKeySize    = types.GidSize + types.AddressSize
	voteKeySize               = 1 + types.GidSize + types.AddressSize
)

var (
	ABIConsensusGroup, _ = abi.JSONToABIContract(strings.NewReader(jsonConsensusGroup))

	groupInfoKeyPrefix = []byte{1}
	voteKeyPrefix      = []byte{0}
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
	return append(groupInfoKeyPrefix, gid.Bytes()...)
}
func GetGidFromConsensusGroupKey(key []byte) types.Gid {
	gid, _ := types.BytesToGid(key[1:])
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
	return append(addr.Bytes(), gid.Bytes()...)
}

func IsRegisterKey(key []byte) bool {
	return len(key) == registerKeySize
}

// Vote variable keys
func GetVoteKey(addr types.Address, gid types.Gid) []byte {
	return helper.JoinBytes(voteKeyPrefix, gid.Bytes(), addr.Bytes())
}

func GetVoteKeyPerfixByGid(gid types.Gid) []byte {
	return append(voteKeyPrefix, gid.Bytes()...)
}

func isVoteKey(key []byte) bool {
	return len(key) == voteKeySize
}

func GetAddrFromVoteKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[1+types.GidSize:])
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

func GetActiveConsensusGroupList(db StorageDatabase) ([]*types.ConsensusGroupInfo, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getActiveConsensusGroupList"}, time.Now())
	iterator, err := db.NewStorageIterator(groupInfoKeyPrefix)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	consensusGroupInfoList := make([]*types.ConsensusGroupInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isConsensusGroupKey) {
			continue
		}
		if info, err := parseConsensusGroup(iterator.Value(), GetGidFromConsensusGroupKey(iterator.Key())); err == nil && info != nil && info.IsActive() {
			consensusGroupInfoList = append(consensusGroupInfoList, info)
		} else {
			return nil, err
		}
	}
	return consensusGroupInfoList, nil
}

func GetConsensusGroup(db StorageDatabase, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	data, err := db.GetValue(GetConsensusGroupKey(gid))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return parseConsensusGroup(data, gid)
	} else {
		return nil, nil
	}
}

func parseConsensusGroup(data []byte, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	consensusGroupInfo := new(types.ConsensusGroupInfo)
	err := ABIConsensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
	if err == nil {
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo, nil
	} else {
		return nil, err
	}
}

func GetRegisterOfPledgeInfo(data []byte) (*VariableConditionRegisterOfPledge, error) {
	pledgeParam := new(VariableConditionRegisterOfPledge)
	err := ABIConsensusGroup.UnpackVariable(pledgeParam, VariableNameConditionRegisterOfPledge, data)
	return pledgeParam, err
}

// Register readers
func IsActiveRegistration(db StorageDatabase, name string, gid types.Gid) (bool, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return false, util.ErrAddressNotMatch
	}
	value, err := db.GetValue(GetRegisterKey(name, gid))
	if err != nil {
		return false, err
	}
	if len(value) > 0 {
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
			return registration.IsActive(), nil
		}
	}
	return false, errors.New("registration not exists")
}

func GetCandidateList(db StorageDatabase, gid types.Gid) ([]*types.Registration, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getCandidateList"}, time.Now())
	var iterator interfaces.StorageIterator
	var err error
	if gid == types.DELEGATE_GID {
		iterator, err = db.NewStorageIterator(types.SNAPSHOT_GID.Bytes())
	} else {
		iterator, err = db.NewStorageIterator(gid.Bytes())
	}
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	registerList := make([]*types.Registration, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsRegisterKey) {
			continue
		}
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, iterator.Value()); err == nil && registration.IsActive() {
			registerList = append(registerList, registration)
		}
	}
	return registerList, nil
}

func GetRegistrationList(db StorageDatabase, gid types.Gid, pledgeAddr types.Address) ([]*types.Registration, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getRegistrationList"}, time.Now())
	var iterator interfaces.StorageIterator
	var err error
	if gid == types.DELEGATE_GID {
		iterator, err = db.NewStorageIterator(types.SNAPSHOT_GID.Bytes())
	} else {
		iterator, err = db.NewStorageIterator(gid.Bytes())
	}
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	registerList := make([]*types.Registration, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsRegisterKey) {
			continue
		}
		registration := new(types.Registration)
		if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, iterator.Value()); err == nil && registration.PledgeAddr == pledgeAddr {
			registerList = append(registerList, registration)
		}
	}
	return registerList, nil
}

func GetRegistration(db StorageDatabase, gid types.Gid, name string) (*types.Registration, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getRegistration"}, time.Now())
	value, err := db.GetValue(GetRegisterKey(name, gid))
	if err != nil {
		return nil, err
	}
	registration := new(types.Registration)
	if err := ABIConsensusGroup.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
		return registration, nil
	}
	return nil, nil
}

// Vote readers
func GetVote(db StorageDatabase, gid types.Gid, addr types.Address) (*types.VoteInfo, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getVote"}, time.Now())
	data, err := db.GetValue(GetVoteKey(addr, gid))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		nodeName := new(string)
		ABIConsensusGroup.UnpackVariable(nodeName, VariableNameVoteStatus, data)
		return &types.VoteInfo{addr, *nodeName}, nil
	}
	return nil, nil
}

func GetVoteList(db StorageDatabase, gid types.Gid) ([]*types.VoteInfo, error) {
	if *db.Address() != types.AddressConsensusGroup {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getVoteList"}, time.Now())
	var iterator interfaces.StorageIterator
	var err error
	if gid == types.DELEGATE_GID {
		iterator, err = db.NewStorageIterator(GetVoteKeyPerfixByGid(types.SNAPSHOT_GID))
	} else {
		iterator, err = db.NewStorageIterator(GetVoteKeyPerfixByGid(gid))
	}
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	voteInfoList := make([]*types.VoteInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isVoteKey) {
			continue
		}
		voterAddr := GetAddrFromVoteKey(iterator.Key())
		nodeName := new(string)
		if err := ABIConsensusGroup.UnpackVariable(nodeName, VariableNameVoteStatus, iterator.Value()); err == nil {
			voteInfoList = append(voteInfoList, &types.VoteInfo{voterAddr, *nodeName})
		}
	}
	return voteInfoList, nil
}
