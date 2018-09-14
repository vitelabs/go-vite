package types

import "math/big"

type TokenInfo struct {
	TokenName    string
	TokenSymbol  string
	TotalSupply  *big.Int
	Decimals     uint8
	Owner        Address
	PledgeAmount *big.Int
	Timestamp    int64
}

type Registration struct {
	Name           string
	NodeAddr       Address
	PledgeAddr     Address
	BeneficialAddr Address
	Amount         *big.Int
	Timestamp      int64
	RewardHeight   uint64
	CancelHeight   uint64
}

type VoteInfo struct {
	VoterAddr Address
	NodeName  string
}
type ConsensusGroupInfo struct {
	Gid                    Gid
	NodeCount              uint8
	Interval               int64
	CountingRuleId         uint8
	CountingRuleParam      []byte
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
}
