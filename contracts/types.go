package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type TokenInfo struct {
	TokenName    string
	TokenSymbol  string
	TotalSupply  *big.Int
	Decimals     uint8
	Owner        types.Address
	PledgeAmount *big.Int
	Timestamp    int64
}

type Registration struct {
	Name           string
	NodeAddr       types.Address
	PledgeAddr     types.Address
	BeneficialAddr types.Address
	Amount         *big.Int
	Timestamp      int64
	RewardHeight   uint64
	CancelHeight   uint64
}

type VoteInfo struct {
	VoterAddr types.Address
	NodeName  string
}
type ConsensusGroupInfo struct {
	Gid                    types.Gid
	NodeCount              uint8
	Interval               int64
	CountingRuleId         uint8
	CountingRuleParam      []byte
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
}
