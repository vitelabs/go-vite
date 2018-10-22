package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
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
