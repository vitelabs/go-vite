package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/vitelabs/go-vite/ledger"
)

type UnconfirmedMeta struct {
	Gid     []byte
	Address types.Address
	Hash    types.Hash
}

type CommonAccountInfo struct {
	AccountAddress *types.Address
	TotalNumber   *big.Int
	TokenInfoList []*TokenInfo
}

// pack the data for handler
type TokenInfo struct {
	Token       *ledger.Mintage
	TotalAmount *big.Int
}

// fixme: AccountBlock is tmp
type AccountBlock struct {
	hash            *types.Hash
	From            *types.Address
	To              *types.Address
	Height          *big.Int
	Type            int
	Code            []byte
	Gid             []byte
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