package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
)

// unpack byte silce to param variables
// pack consensus group condition params by condition id
// unpack consensus group condition param byte slice by condition id

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
	RegisterConditionPrefix     ConditionCode = 10
	VoteConditionPrefix         ConditionCode = 20
	RegisterConditionOfSnapshot ConditionCode = 10
	VoteConditionOfDefault      ConditionCode = 20
	VoteConditionOfBalance      ConditionCode = 21
)

func PackConsensusGroupConditionParam(conditionIdPrefix ConditionCode, conditionId ConditionCode) ([]byte, error) {
	// TODO
	return nil, nil
}
