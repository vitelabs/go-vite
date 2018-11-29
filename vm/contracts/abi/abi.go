package abi

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type StorageDatabase interface {
	GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte
	NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator
}

var (
	precompiledContractsAbiMap = map[types.Address]abi.ABIContract{
		types.AddressRegister:       ABIRegister,
		types.AddressVote:           ABIVote,
		types.AddressPledge:         ABIPledge,
		types.AddressConsensusGroup: ABIConsensusGroup,
		types.AddressMintage:        ABIMintage,
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
