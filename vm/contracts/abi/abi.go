package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

type StorageDatabase interface {
	GetValue(key []byte) ([]byte, error)
	NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error)
	Address() *types.Address
}

type ConditionCode uint8

const (
	RegisterConditionPrefix   ConditionCode = 10
	VoteConditionPrefix       ConditionCode = 20
	RegisterConditionOfPledge ConditionCode = 11
	VoteConditionOfDefault    ConditionCode = 21
)

var (
	consensusGroupConditionIdNameMap = map[ConditionCode]string{
		RegisterConditionOfPledge: VariableNameConditionRegisterOfPledge,
	}
)

func filterKeyValue(key, value []byte, f func(key []byte) bool) bool {
	if len(value) > 0 && (f == nil || f(key)) {
		return true
	}
	return false
}
