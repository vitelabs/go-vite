package config

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type ConditionRegisterData struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}

type VoteConditionData struct {
	Amount  *big.Int
	TokenId types.TokenTypeId
}

type ConsensusGroupInfo struct {
	NodeCount              uint8
	Interval               int64
	PerCount               int64
	RandCount              uint8
	RandRank               uint8
	CountingTokenId        types.TokenTypeId
	RegisterConditionId    uint8
	RegisterConditionParam ConditionRegisterData
	VoteConditionId        uint8
	VoteConditionParam     VoteConditionData
	Owner                  types.Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}

type ForkPoint struct {
	Height uint64
	Hash   *types.Hash
}

type ForkPoints struct{}

type Genesis struct {
	GenesisAccountAddress  types.Address
	BlockProducers         []types.Address
	SnapshotConsensusGroup *ConsensusGroupInfo
	CommonConsensusGroup   *ConsensusGroupInfo

	ForkPoints *ForkPoints
}
