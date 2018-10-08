package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type TokenInfo struct {
	TokenName      string
	TokenSymbol    string
	TotalSupply    *big.Int
	Decimals       uint8
	Owner          types.Address
	PledgeAmount   *big.Int
	WithdrawHeight uint64
}

type Registration struct {
	Name           string
	NodeAddr       types.Address
	PledgeAddr     types.Address
	BeneficialAddr types.Address
	Amount         *big.Int
	PledgeHeight   uint64
	RewardHeight   uint64
	CancelHeight   uint64
}

func (r *Registration) IsActive() bool {
	return r.CancelHeight == 0
}

type VoteInfo struct {
	VoterAddr types.Address
	NodeName  string
}

type PledgeInfo struct {
	Amount         *big.Int
	WithdrawHeight uint64
	BeneficialAddr types.Address
}

type ConsensusGroupInfo struct {
	Gid                    types.Gid         // Consensus group id
	NodeCount              uint8             // Active miner count
	Interval               int64             // Timestamp gap between two continuous block
	PerCount               int64             // Continuous block generation interval count
	RandCount              uint8             // Random miner count
	RandRank               uint8             // Chose random miner with a rank limit of vote
	CountingTokenId        types.TokenTypeId // Token id for selecting miner through vote
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
	Owner                  types.Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}

func (groupInfo *ConsensusGroupInfo) IsActive() bool {
	return groupInfo.WithdrawHeight > 0
}
