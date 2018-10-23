package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
)

var (
	precompiledContractsAbiMap = map[types.Address]abi.ABIContract{
		AddressRegister:       ABIRegister,
		AddressVote:           ABIVote,
		AddressPledge:         ABIPledge,
		AddressConsensusGroup: ABIConsensusGroup,
		AddressMintage:        ABIMintage,
	}

	errInvalidParam = errors.New("invalid param")
)

// pack method params to byte slice
func PackMethodParam(contractsAddr types.Address, methodName string, params ...interface{}) ([]byte, error) {
	if abiContract, ok := precompiledContractsAbiMap[contractsAddr]; ok {
		if data, err := abiContract.PackMethod(methodName, params...); err == nil {
			return data, nil
		}
	}
	return nil, errInvalidParam
}

// Sign certain tx data using private key of node address to prove ownership of node address
func GetRegisterMessageForSignature(accountAddress types.Address, gid types.Gid) []byte {
	return helper.JoinBytes(accountAddress.Bytes(), gid.Bytes())
}

type ConditionCode uint8

const (
	RegisterConditionPrefix   ConditionCode = 10
	VoteConditionPrefix       ConditionCode = 20
	RegisterConditionOfPledge ConditionCode = 11
	VoteConditionOfDefault    ConditionCode = 21
	VoteConditionOfBalance    ConditionCode = 22
)

var (
	consensusGroupConditionIdNameMap = map[ConditionCode]string{
		RegisterConditionOfPledge: VariableNameConditionRegisterOfPledge,
		VoteConditionOfBalance:    VariableNameConditionVoteOfKeepToken,
	}
)

// pack consensus group condition params by condition id
func PackConsensusGroupConditionParam(conditionIdPrefix ConditionCode, conditionId uint8, params ...interface{}) ([]byte, error) {
	if name, ok := consensusGroupConditionIdNameMap[conditionIdPrefix+ConditionCode(conditionId)]; ok {
		if data, err := ABIConsensusGroup.PackVariable(name, params...); err == nil {
			return data, nil
		}
	}
	return nil, errInvalidParam
}

func NewGid(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Gid {
	return types.DataToGid(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func NewTokenId(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetNewContractData(bytecode []byte, gid types.Gid) []byte {
	return append(gid.Bytes(), bytecode...)
}

func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}

func NewContractAddress(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}
