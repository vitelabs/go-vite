package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

type CountingRuleCode string

const (
	CountingRuleOfBalance       CountingRuleCode = "counting1"
	RegisterConditionOfSnapshot CountingRuleCode = "register1"
	VoteConditionOfDefault      CountingRuleCode = "vote1"
	VoteConditionOfBalance      CountingRuleCode = "vote2"
)

type createConsensusGroupCondition interface {
	checkParam(param []byte, db VmDatabase) bool
}

var SimpleCountingRuleList = map[CountingRuleCode]createConsensusGroupCondition{
	CountingRuleOfBalance:       &countingRuleOfBalance{},
	RegisterConditionOfSnapshot: &registerConditionOfSnapshot{},
	VoteConditionOfDefault:      &voteConditionOfDefault{},
	VoteConditionOfBalance:      &voteConditionOfBalance{},
}

type countingRuleOfBalance struct{}

func (c countingRuleOfBalance) checkParam(param []byte, db VmDatabase) bool {
	if len(param) != 32 {
		return false
	}
	if tokenId, err := types.BytesToTokenTypeId(util.LeftPadBytes(new(big.Int).SetBytes(param).Bytes(), 10)); err != nil || !db.IsExistToken(tokenId) {
		return false
	}
	return true
}

type registerConditionOfSnapshot struct{}

func (c registerConditionOfSnapshot) checkParam(param []byte, db VmDatabase) bool {
	if len(param) != 96 {
		return false
	}
	if tokenId, err := types.BytesToTokenTypeId(util.LeftPadBytes(new(big.Int).SetBytes(param[32:64]).Bytes(), 10)); err != nil || !db.IsExistToken(tokenId) {
		return false
	}
	return true
}

type voteConditionOfDefault struct{}

func (c voteConditionOfDefault) checkParam(param []byte, db VmDatabase) bool {
	if len(param) != 0 {
		return false
	}
	return true
}

type voteConditionOfBalance struct{}

func (c voteConditionOfBalance) checkParam(param []byte, db VmDatabase) bool {
	if len(param) != 64 {
		return false
	}
	if tokenId, err := types.BytesToTokenTypeId(util.LeftPadBytes(new(big.Int).SetBytes(param[32:64]).Bytes(), 10)); err != nil || !db.IsExistToken(tokenId) {
		return false
	}
	return true
}
