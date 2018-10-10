package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
	"strconv"
)

type AccountBlock struct {
	*ledger.AccountBlock

	Height string
	Quota  string

	Amount string
	Fee    string

	ConfirmedTimes string
	TokenInfo      *RpcTokenInfo
}

func (ab *AccountBlock) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	lAb := ab.AccountBlock

	var err error
	lAb.Height, err = strconv.ParseUint(ab.Height, 10, 8)
	if err != nil {
		return nil, err
	}
	lAb.Quota, err = strconv.ParseUint(ab.Quota, 10, 8)
	if err != nil {
		return nil, err
	}

	var parseSuccess bool
	lAb.Amount, parseSuccess = new(big.Int).SetString(ab.Amount, 10)
	if !parseSuccess {
		return nil, errors.New("parse amount failed")
	}
	lAb.Fee, parseSuccess = new(big.Int).SetString(ab.Fee, 10)

	if !parseSuccess {
		return nil, errors.New("parse fee failed")
	}
	return lAb, nil
}

func createAccountBlock(ledgerBlock *ledger.AccountBlock, token *contracts.TokenInfo, confirmedTimes uint64) *AccountBlock {
	ab := &AccountBlock{
		AccountBlock: ledgerBlock,

		Height: strconv.FormatUint(ledgerBlock.Height, 10),
		Quota:  strconv.FormatUint(ledgerBlock.Quota, 10),

		Amount:         "0",
		Fee:            "0",
		TokenInfo:      RawTokenInfoToRpc(token),
		ConfirmedTimes: strconv.FormatUint(confirmedTimes, 10),
	}

	if token != nil {
		ab.TokenInfo = RawTokenInfoToRpc(token)
	}
	if ledgerBlock.Amount != nil {
		ab.Amount = ledgerBlock.Amount.String()
	}
	if ledgerBlock.Fee != nil {
		ab.Fee = ledgerBlock.Fee.String()
	}
	return ab
}

type RpcAccountInfo struct {
	AccountAddress      types.Address
	TotalNumber         string // uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*RpcTokenBalanceInfo `json:",omitempty"`
}

func create() {

}

type RpcTokenBalanceInfo struct {
	TokenInfo   *RpcTokenInfo `json:",omitempty"`
	TotalAmount string  // big int
	Number      *string // uint64
}

type RpcTokenInfo struct {
	TokenName      string
	TokenSymbol    string
	TotalSupply    *string `json:",omitempty"` // *big.Int
	Decimals       uint8
	Owner          types.Address
	PledgeAmount   *string `json:",omitempty"` // *big.Int
	WithdrawHeight string                      // uint64
}

func RawTokenInfoToRpc(tinfo *contracts.TokenInfo) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:      tinfo.TokenName,
			TokenSymbol:    tinfo.TokenSymbol,
			TotalSupply:    nil,
			Decimals:       tinfo.Decimals,
			Owner:          tinfo.Owner,
			PledgeAmount:   nil,
			WithdrawHeight: strconv.FormatUint(tinfo.WithdrawHeight, 10),
		}
		if tinfo.TotalSupply != nil {
			s := tinfo.TotalSupply.String()
			rt.TotalSupply = &s
		}
		if tinfo.PledgeAmount != nil {
			s := tinfo.PledgeAmount.String()
			rt.PledgeAmount = &s
		}
	}
	return rt
}

//// Send tx parms
//type SendTxParms struct {
//	SelfAddr    types.Address     `json:"selfAddr"`    // who sends the tx
//	ToAddr      types.Address     `json:"toAddr"`      // who receives the tx
//	TokenTypeId types.TokenTypeId `json:"tokenTypeId"` // which token will be sent
//	Passphrase  string            `json:"passphrase"`  // sender`s passphrase
//	Amount      string            `json:"amount"`      // the amount of specific token will be sent. bigInt
//}
//
//type BalanceInfo struct {
//	Mintage *Mintage `json:"mintage"`
//
//	Balance string `json:"balance"`
//}
//
//type GetAccountResponse struct {
//	Addr         types.Address `json:"addr"`         // Account address
//	BalanceInfos []BalanceInfo `json:"balanceInfos"` // Account Balance Infos
//	BlockHeight  string        `json:"blockHeight"`  // Account BlockHeight also represents all blocks belong to the account. bigInt.
//}
//
//type GetUnconfirmedInfoResponse struct {
//	Addr                 types.Address `json:"addr"`                 // Account address
//	BalanceInfos         []BalanceInfo `json:"balanceInfos"`         // Account unconfirmed BalanceInfos (In-transit money)
//	UnConfirmedBlocksLen string        `json:"unConfirmedBlocksLen"` // the length of unconfirmed blocks. bigInt
//}
//
//type InitSyncResponse struct {
//	StartHeight      string `json:"startHeight"`      // bigInt. where we start sync
//	TargetHeight     string `json:"targetHeight"`     // bigInt. when CurrentHeight == TargetHeight means that sync complete
//	CurrentHeight    string `json:"currentHeight"`    // bigInt.
//	IsFirstSyncDone  bool   `json:"isFirstSyncDone"`  // true means sync complete
//	IsStartFirstSync bool   `json:"isStartFirstSync"` // true means sync start
//}
//
//type Mintage struct {
//	Name        string             `json:"name"`
//	Id          *types.TokenTypeId `json:"id"`
//	Symbol      string             `json:"symbol"`
//	Owner       *types.Address     `json:"owner"`
//	Decimals    int                `json:"decimals"`
//	TotalSupply *string            `json:"totalSupply"`
//}
//
//func rawMintageToRpc(l *ledger.Mintage) *Mintage {
//	if l == nil {
//		return nil
//	}
//	return &Mintage{
//		Name:        l.Name,
//		Id:          l.Id,
//		Symbol:      l.Symbol,
//		Owner:       l.Owner,
//		Decimals:    l.Decimals,
//		TotalSupply: bigIntToString(l.TotalSupply),
//	}
//}
