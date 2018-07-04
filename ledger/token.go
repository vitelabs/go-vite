package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"encoding/json"
)

type Mintage struct {
	Name string
	Id *types.TokenTypeId
	Introduction string
	Symbol string

	Owner *types.Address
	Decimals int
	TotalSupply *big.Int
}

func NewMintage (mintageBlock *AccountBlock) (*Mintage, error){
	var mintage *Mintage
	if err := json.Unmarshal([]byte(mintageBlock.Data), &mintage); err != nil {
		return nil, err
	}

	return mintage, nil
}

type Token struct {
	MintageBlock *AccountBlock
	Mintage *Mintage
}

func NewToken (mintageBlock *AccountBlock) (*Token, error){
	mintage, err := NewMintage(mintageBlock)
	return &Token {
		MintageBlock: mintageBlock,
		Mintage: mintage,
	}, err
}