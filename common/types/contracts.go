package types

import "math/big"

type ConsensusGroupInfo struct {
	Gid                    Gid         // Consensus group id
	NodeCount              uint8       // Active miner count
	Interval               int64       // Timestamp gap between two continuous block
	PerCount               int64       // Continuous block generation interval count
	RandCount              uint8       // Random miner count
	RandRank               uint8       // Chose random miner with a rank limit of vote
	CountingTokenId        TokenTypeId // Token id for selecting miner through vote
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
	Owner                  Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}

func (groupInfo *ConsensusGroupInfo) IsActive() bool {
	return groupInfo.WithdrawHeight > 0
}

type VoteInfo struct {
	VoterAddr Address
	NodeName  string
}

type Registration struct {
	Name           string
	NodeAddr       Address
	PledgeAddr     Address
	Amount         *big.Int
	WithdrawHeight uint64
	RewardIndex    uint64
	CancelHeight   uint64
	HisAddrList    []Address
}

func (r *Registration) IsActive() bool {
	return r.CancelHeight == 0
}

type TokenInfo struct {
	TokenName      string   `json:"tokenName"`
	TokenSymbol    string   `json:"tokenSymbol"`
	TotalSupply    *big.Int `json:"totalSupply"`
	Decimals       uint8    `json:"decimals"`
	Owner          Address  `json:"owner"`
	PledgeAmount   *big.Int `json:"pledgeAmount"`
	WithdrawHeight uint64   `json:"withdrawHeight"`
}
