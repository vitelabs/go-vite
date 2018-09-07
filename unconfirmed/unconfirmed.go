package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type UnconfirmedMeta struct {
	Gid     []byte
	Address types.Address
	Hash    types.Hash
}

type CommonAccountInfo struct {
	AccountAddress *types.Address
	TotalNumber    uint64
	TokenInfoMap   map[*types.TokenTypeId]*TokenInfo
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
	Gid             *types.Gid
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

func (ab AccountBlock) IsContractTx() bool {
	return len(ab.Code) != 0
}
