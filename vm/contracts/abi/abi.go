package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type StorageDatabase interface {
	GetStorage(addr *types.Address, key []byte) []byte
	NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator
	NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator
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
