package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
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
		VoteConditionOfBalance:    VariableNameConditionVoteOfBalance,
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
