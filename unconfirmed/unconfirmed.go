package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// db struct
type unconfirmedBlock struct {
	gid 	string
	address types.Address
	hash    types.Hash
}

// memery struct
type TokenInfo struct {
	TokenId     *types.TokenTypeId
	TotalAmount *big.Int
}

type UnconfirmedMeta struct {
	TotalNumber   *big.Int
	TokenInfoList []*TokenInfo
}

type AccountBlock struct {
	hash            *types.Hash
	From            *types.Address
	To              *types.Address
	Height          *big.Int
	Type            int
	PrevHash        *types.Hash
	FromHash        *types.Hash
	Amount          *big.Int
	TokenId         *types.TokenTypeId
	CreateFee       *big.Int
	Data            []byte
	StateHash       types.Hash
	SummaryHashList []types.Hash
	LogHash         types.Hash
	SnapshotHash    types.Hash
	Depth           int
	Quota           uint64
	Hash            *types.Hash
	Balance         map[types.TokenTypeId]*big.Int
}

