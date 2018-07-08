package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"encoding/json"
)

var MintageAddress, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
var MockViteTokenId, _= types.BytesToTokenTypeId([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // mock

type mintageJSON struct {
	Name string 			`json:"tokenName"`
	Id string				`json:"tokenId"`
	Symbol string			`json:"tokenSymbol"`

	Owner string			`json:"owner"`
	Decimals int			`json:"decimals"`
	TotalSupply string		`json:"totalSupply"`
}


type Mintage struct {
	Name string
	Id *types.TokenTypeId
	Symbol string

	Owner *types.Address
	Decimals int
	TotalSupply *big.Int
}

func NewMintage (mintageBlock *AccountBlock) (*Mintage, error){

	var mintageData *mintageJSON
	if err := json.Unmarshal([]byte(mintageBlock.Data), &mintageData); err != nil {
		return nil, err
	}


	var id types.TokenTypeId

	if mintageData.Id != "" {
		var err error
		id, err = types.HexToTokenTypeId(mintageData.Id)
		if err != nil {
			return nil, err
		}
	}


	if &id == nil {
		id = MockViteTokenId
	}

	owner, err := types.HexToAddress(mintageData.Owner)
	if err != nil {
		return nil, err
	}

	totalSupply := &big.Int{}
	totalSupply.SetString(mintageData.TotalSupply, 10)

	mintage := &Mintage{
		Name: mintageData.Name,
		Id: &id,
		Symbol: mintageData.Symbol,

		Owner: &owner,
		Decimals: mintageData.Decimals,
		TotalSupply: totalSupply,
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