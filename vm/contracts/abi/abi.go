package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type StorageDatabase interface {
	GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte
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
