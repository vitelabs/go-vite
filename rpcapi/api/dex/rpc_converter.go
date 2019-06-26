package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)



type DividendPoolInfo struct {
	Amount         string           `json:"amount"`
	QuoteTokenType int32            `json:"quoteTokenType"`
	TokenInfo      *RpcDexTokenInfo `json:"tokenInfo,omitempty"`
}

type RpcMarketInfo struct {
	MarketId           int32  `json:"marketId"`
	MarketSymbol       string `json:"marketSymbol"`
	TradeToken         string `json:"tradeToken"`
	QuoteToken         string `json:"quoteToken"`
	QuoteTokenType     int32  `json:"quoteTokenType"`
	TradeTokenDecimals int32  `json:"tradeTokenDecimals,omitempty"`
	QuoteTokenDecimals int32  `json:"quoteTokenDecimals"`
	TakerBrokerFeeRate int32  `json:"takerBrokerFeeRate,omitempty"`
	MakerBrokerFeeRate int32  `json:"makerBrokerFeeRate,omitempty"`
	AllowMine          bool   `json:"allowMine"`
	Valid              bool   `json:"valid"`
	Owner              string `json:"owner"`
	Creator            string `json:"creator"`
	Stopped            bool   `json:"stopped"`
	Timestamp          int64  `json:"timestamp"`
}

func MarketInfoToRpc(mkInfo *dex.MarketInfo) *RpcMarketInfo {
	var rmk *RpcMarketInfo = nil
	if mkInfo != nil {
		tradeToken, _ := types.BytesToTokenTypeId(mkInfo.TradeToken)
		quoteToken, _ := types.BytesToTokenTypeId(mkInfo.QuoteToken)
		owner, _ := types.BytesToAddress(mkInfo.Owner)
		creator, _ := types.BytesToAddress(mkInfo.Creator)
		rmk = &RpcMarketInfo{
			MarketId:           mkInfo.MarketId,
			MarketSymbol:       mkInfo.MarketSymbol,
			TradeToken:         tradeToken.String(),
			QuoteToken:         quoteToken.String(),
			QuoteTokenType:     mkInfo.QuoteTokenType,
			TradeTokenDecimals: mkInfo.TradeTokenDecimals,
			QuoteTokenDecimals: mkInfo.QuoteTokenDecimals,
			TakerBrokerFeeRate: mkInfo.TakerBrokerFeeRate,
			MakerBrokerFeeRate: mkInfo.MakerBrokerFeeRate,
			AllowMine:          mkInfo.AllowMine,
			Valid:              mkInfo.Valid,
			Owner:              owner.String(),
			Creator:            creator.String(),
			Stopped:            mkInfo.Stopped,
			Timestamp:          mkInfo.Timestamp,
		}
	}
	return rmk
}

type RpcDexTokenInfo struct {
	TokenSymbol    string            `json:"tokenSymbol"`
	Decimals       int32             `json:"decimals"`
	TokenId        types.TokenTypeId `json:"tokenId"`
	Index          int32             `json:"index"`
	Owner          types.Address     `json:"owner"`
	QuoteTokenType int32             `json:"quoteTokenType"`
}

func TokenInfoToRpc(tinfo *dex.TokenInfo, tti types.TokenTypeId) *RpcDexTokenInfo {
	var rt *RpcDexTokenInfo = nil
	if tinfo != nil {
		owner, _ := types.BytesToAddress(tinfo.Owner)
		rt = &RpcDexTokenInfo{
			TokenSymbol:    tinfo.Symbol,
			Decimals:       tinfo.Decimals,
			TokenId:        tti,
			Index:          tinfo.Index,
			Owner:          owner,
			QuoteTokenType: tinfo.QuoteTokenType,
		}
	}
	return rt
}

type RpcFeeSumForDividend struct {
	Token              string `json:"token"`
	DividendPoolAmount string `json:"dividendPoolAmount"`
}

type RpcFeeSumForMine struct {
	QuoteTokenType    int32  `json:"quoteTokenType"`
	BaseAmount        string `json:"baseAmount"`
	InviteBonusAmount string `json:"inviteBonusAmount"`
}

type RpcFeeSumByPeriod struct {
	FeesForDividend   []*RpcFeeSumForDividend `json:"feesForDividend"`
	FeesForMine       []*RpcFeeSumForMine     `json:"feesForMine"`
	LastValidPeriod   uint64                  `json:"lastValidPeriod"`
	FinishFeeDividend bool                    `json:"finishFeeDividend"`
	FinishVxMine      bool                    `json:"finishVxMine"`
}

func FeeSumByPeriodToRpc(feeSum *dex.FeeSumByPeriod) *RpcFeeSumByPeriod {
	if feeSum == nil {
		return nil
	}
	rpcFeeSum := &RpcFeeSumByPeriod{}
	for _, dividend := range feeSum.FeesForDividend {
		rpcDividend := &RpcFeeSumForDividend{}
		rpcDividend.Token = tokenBytesToString(dividend.Token)
		rpcDividend.DividendPoolAmount = amountBytesToString(dividend.DividendPoolAmount)
		rpcFeeSum.FeesForDividend = append(rpcFeeSum.FeesForDividend, rpcDividend)
	}
	for _, mine := range feeSum.FeesForMine {
		rpcMine := &RpcFeeSumForMine{}
		rpcMine.QuoteTokenType = mine.QuoteTokenType
		rpcMine.BaseAmount = amountBytesToString(mine.BaseAmount)
		rpcMine.InviteBonusAmount = amountBytesToString(mine.InviteBonusAmount)
		rpcFeeSum.FeesForMine = append(rpcFeeSum.FeesForMine, rpcMine)
	}
	rpcFeeSum.LastValidPeriod = feeSum.LastValidPeriod
	rpcFeeSum.FinishFeeDividend = feeSum.FinishFeeDividend
	rpcFeeSum.FinishVxMine = feeSum.FinishVxMine
	return rpcFeeSum
}

type RpcUserFeeAccount struct {
	QuoteTokenType    int32  `json:"quoteTokenType"`
	BaseAmount        string `json:"baseAmount"`
	InviteBonusAmount string `json:"inviteBonusAmount"`
}

type RpcUserFeeByPeriod struct {
	UserFees []*RpcUserFeeAccount `json:"userFees"`
	Period   uint64               `json:"period"`
}

type RpcUserFees struct {
	Fees []*RpcUserFeeByPeriod `json:"fees"`
}

func UserFeesToRpc(userFees *dex.UserFees) *RpcUserFees {
	if userFees == nil {
		return nil
	}
	rpcUserFees := &RpcUserFees{}
	for _, fee := range userFees.Fees {
		rpcFee := &RpcUserFeeByPeriod{}
		for _, acc := range fee.UserFees {
			rpcAcc := &RpcUserFeeAccount{}
			rpcAcc.QuoteTokenType = acc.QuoteTokenType
			rpcAcc.BaseAmount = amountBytesToString(acc.BaseAmount)
			rpcAcc.InviteBonusAmount = amountBytesToString(acc.InviteBonusAmount)
			rpcFee.UserFees = append(rpcFee.UserFees, rpcAcc)
		}
		rpcFee.Period = fee.Period
		rpcUserFees.Fees = append(rpcUserFees.Fees, rpcFee)
	}
	return rpcUserFees
}

type RpcVxFundByPeriod struct {
	Amount string `json:"amount"`
	Period uint64 `json:"period"`
}

type RpcVxFunds struct {
	Funds []*RpcVxFundByPeriod `json:"funds"`
}

func VxFundsToRpc(funds *dex.VxFunds) *RpcVxFunds {
	if funds == nil {
		return nil
	}
	rpcFunds := &RpcVxFunds{}
	for _, fund := range funds.Funds {
		rpcFund := &RpcVxFundByPeriod{}
		rpcFund.Period = fund.Period
		rpcFund.Amount = amountBytesToString(fund.Amount)
		rpcFunds.Funds = append(rpcFunds.Funds, rpcFund)
	}
	return rpcFunds
}

func amountBytesToString(amt []byte) string {
	return new(big.Int).SetBytes(amt).String()
}

func tokenBytesToString(token []byte) string {
	tk, _ := types.BytesToTokenTypeId(token)
	return tk.String()
}
