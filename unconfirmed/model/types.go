package model

import (
	"container/list"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger_old"
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
	Number      *big.Int
	TxList      list.List
}

func (ab AccountBlock) IsContractTx() bool {
	return len(ab.Code) != 0
}
