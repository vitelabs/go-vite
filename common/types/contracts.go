package types

import (
	"math/big"
)

type ConsensusGroupInfo struct {
	Gid                    Gid         // Consensus group id
	NodeCount              uint8       // Active miner count
	Interval               int64       // Timestamp gap between two continuous block
	PerCount               int64       // Continuous block generation interval count
	RandCount              uint8       // Random miner count
	RandRank               uint8       // Chose random miner with chain rank limit of vote
	Repeat                 uint16      // reuse consensus info to produce blocks within repeat times
	CheckLevel             uint8       // consensus check param, 0-check address and sequence, 1-check address only
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
	RewardTime     int64
	CancelTime     int64
	HisAddrList    []Address
}

func (r *Registration) IsActive() bool {
	return r.CancelTime == 0
}

type TokenInfo struct {
	TokenName     string   `json:"tokenName"`
	TokenSymbol   string   `json:"tokenSymbol"`
	TotalSupply   *big.Int `json:"totalSupply"`
	Decimals      uint8    `json:"decimals"`
	Owner         Address  `json:"owner"`
	MaxSupply     *big.Int `json:"maxSupply"`
	OwnerBurnOnly bool     `json:"ownerBurnOnly"`
	IsReIssuable  bool     `json:"isReIssuable"`
	Index         uint16   `json:"index"`
}

type PledgeInfo struct {
	Amount         *big.Int `json:"amount"`
	WithdrawHeight uint64   `json:"withdrawHeight"`
	BeneficialAddr Address  `json:"beneficialAddr"`
	Agent          bool     `json:"agent"`
	AgentAddress   Address  `json:"agentAddr"`
	Bid            uint8    `json:"bid"`
	PledgeAddress  Address  `json:"pledgeAddr"`
}
