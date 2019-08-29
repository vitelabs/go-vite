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
	MarketId             int32  `json:"marketId"`
	MarketSymbol         string `json:"marketSymbol"`
	TradeToken           string `json:"tradeToken"`
	QuoteToken           string `json:"quoteToken"`
	QuoteTokenType       int32  `json:"quoteTokenType"`
	TradeTokenDecimals   int32  `json:"tradeTokenDecimals,omitempty"`
	QuoteTokenDecimals   int32  `json:"quoteTokenDecimals"`
	TakerOperatorFeeRate int32  `json:"takerOperatorFeeRate"`
	MakerOperatorFeeRate int32  `json:"makerOperatorFeeRate"`
	AllowMine            bool   `json:"allowMine"`
	Valid                bool   `json:"valid"`
	Owner                string `json:"owner"`
	Creator              string `json:"creator"`
	Stopped              bool   `json:"stopped"`
	Timestamp            int64  `json:"timestamp"`
}

func MarketInfoToRpc(mkInfo *dex.MarketInfo) *RpcMarketInfo {
	var rmk *RpcMarketInfo = nil
	if mkInfo != nil {
		tradeToken, _ := types.BytesToTokenTypeId(mkInfo.TradeToken)
		quoteToken, _ := types.BytesToTokenTypeId(mkInfo.QuoteToken)
		owner, _ := types.BytesToAddress(mkInfo.Owner)
		creator, _ := types.BytesToAddress(mkInfo.Creator)
		rmk = &RpcMarketInfo{
			MarketId:             mkInfo.MarketId,
			MarketSymbol:         mkInfo.MarketSymbol,
			TradeToken:           tradeToken.String(),
			QuoteToken:           quoteToken.String(),
			QuoteTokenType:       mkInfo.QuoteTokenType,
			TradeTokenDecimals:   mkInfo.TradeTokenDecimals,
			QuoteTokenDecimals:   mkInfo.QuoteTokenDecimals,
			TakerOperatorFeeRate: mkInfo.TakerOperatorFeeRate,
			MakerOperatorFeeRate: mkInfo.MakerOperatorFeeRate,
			AllowMine:            mkInfo.AllowMining,
			Valid:                mkInfo.Valid,
			Owner:                owner.String(),
			Creator:              creator.String(),
			Stopped:              mkInfo.Stopped,
			Timestamp:            mkInfo.Timestamp,
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

type RpcFeesForDividend struct {
	Token              string `json:"token"`
	DividendPoolAmount string `json:"dividendPoolAmount"`
}

type RpcFeesForMine struct {
	QuoteTokenType    int32  `json:"quoteTokenType"`
	BaseAmount        string `json:"baseAmount"`
	InviteBonusAmount string `json:"inviteBonusAmount"`
}

type RpcDexFeesByPeriod struct {
	FeesForDividend []*RpcFeesForDividend `json:"feesForDividend"`
	FeesForMine     []*RpcFeesForMine     `json:"feesForMine"`
	LastValidPeriod uint64                `json:"lastValidPeriod"`
	FinishDividend  bool                  `json:"finishFeeDividend"`
	FinishMine      bool                  `json:"finishVxMine"`
}

func FeeSumByPeriodToRpc(feeSum *dex.DexFeesByPeriod) *RpcDexFeesByPeriod {
	if feeSum == nil {
		return nil
	}
	rpcDexFeesByPeriod := &RpcDexFeesByPeriod{}
	for _, dividend := range feeSum.FeesForDividend {
		rpcDividend := &RpcFeesForDividend{}
		rpcDividend.Token = TokenBytesToString(dividend.Token)
		rpcDividend.DividendPoolAmount = AmountBytesToString(dividend.DividendPoolAmount)
		rpcDexFeesByPeriod.FeesForDividend = append(rpcDexFeesByPeriod.FeesForDividend, rpcDividend)
	}
	for _, mine := range feeSum.FeesForMine {
		rpcMine := &RpcFeesForMine{}
		rpcMine.QuoteTokenType = mine.QuoteTokenType
		rpcMine.BaseAmount = AmountBytesToString(mine.BaseAmount)
		rpcMine.InviteBonusAmount = AmountBytesToString(mine.InviteBonusAmount)
		rpcDexFeesByPeriod.FeesForMine = append(rpcDexFeesByPeriod.FeesForMine, rpcMine)
	}
	rpcDexFeesByPeriod.LastValidPeriod = feeSum.LastValidPeriod
	rpcDexFeesByPeriod.FinishDividend = feeSum.FinishDividend
	rpcDexFeesByPeriod.FinishMine = feeSum.FinishMine
	return rpcDexFeesByPeriod
}

type RpcOperatorMarketFee struct {
	MarketId             int32  `json:"marketId"`
	TakerOperatorFeeRate int32  `json:"takerOperatorFeeRate"`
	MakerOperatorFeeRate int32  `json:"makerOperatorFeeRate"`
	Amount               string `json:"amount"`
}

type RpcOperatorFeeAccount struct {
	Token      string                  `json:"token"`
	MarketFees []*RpcOperatorMarketFee `json:"marketFees"`
}

type RpcOperatorFeesByPeriod struct {
	OperatorFees []*RpcOperatorFeeAccount `json:"operatorFees"`
}

func OperatorFeesByPeriodToRpc(operatorFees *dex.OperatorFeesByPeriod) *RpcOperatorFeesByPeriod {
	if operatorFees == nil {
		return nil
	}
	rpcOperatorFees := &RpcOperatorFeesByPeriod{}
	for _, fee := range operatorFees.OperatorFees {
		rpcFee := &RpcOperatorFeeAccount{}
		rpcFee.Token = TokenBytesToString(fee.Token)
		for _, acc := range fee.MarketFees {
			rpcAcc := &RpcOperatorMarketFee{}
			rpcAcc.MarketId = acc.MarketId
			rpcAcc.TakerOperatorFeeRate = acc.TakerOperatorFeeRate
			rpcAcc.MakerOperatorFeeRate = acc.MakerOperatorFeeRate
			rpcAcc.Amount = AmountBytesToString(acc.Amount)
			rpcFee.MarketFees = append(rpcFee.MarketFees, rpcAcc)
		}
		rpcOperatorFees.OperatorFees = append(rpcOperatorFees.OperatorFees, rpcFee)
	}
	return rpcOperatorFees
}

type RpcFeeAccount struct {
	QuoteTokenType    int32  `json:"quoteTokenType"`
	BaseAmount        string `json:"baseAmount"`
	InviteBonusAmount string `json:"inviteBonusAmount"`
}

type RpcFeesByPeriod struct {
	UserFees []*RpcFeeAccount `json:"userFees"`
	Period   uint64           `json:"period"`
}

type RpcUserFees struct {
	Fees []*RpcFeesByPeriod `json:"fees"`
}

func UserFeesToRpc(userFees *dex.UserFees) *RpcUserFees {
	if userFees == nil {
		return nil
	}
	rpcUserFees := &RpcUserFees{}
	for _, fee := range userFees.Fees {
		rpcFee := &RpcFeesByPeriod{}
		for _, acc := range fee.Fees {
			rpcAcc := &RpcFeeAccount{}
			rpcAcc.QuoteTokenType = acc.QuoteTokenType
			rpcAcc.BaseAmount = AmountBytesToString(acc.BaseAmount)
			rpcAcc.InviteBonusAmount = AmountBytesToString(acc.InviteBonusAmount)
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
		rpcFund.Amount = AmountBytesToString(fund.Amount)
		rpcFunds.Funds = append(rpcFunds.Funds, rpcFund)
	}
	return rpcFunds
}

type RpcThresholdForTradeAndMine struct {
	TradeThreshold string `json:"tradeThreshold"`
	MineThreshold  string `json:"mineThreshold"`
}

type RpcStackedForVxByPeriod struct {
	Period uint64 `json:"period"`
	Amount string `json:"amount"`
}

type RpcPledgesForVx struct {
	Pledges []*RpcStackedForVxByPeriod `json:"Pledges"`
}

func PledgesForVxToRpc(miningStakings *dex.MiningStakings) *RpcPledgesForVx {
	rpcPledges := &RpcPledgesForVx{}
	for _, pledge := range miningStakings.Stakings {
		rpcPledge := &RpcStackedForVxByPeriod{}
		rpcPledge.Period = pledge.Period
		rpcPledge.Amount = AmountBytesToString(pledge.Amount)
		rpcPledges.Pledges = append(rpcPledges.Pledges, rpcPledge)
	}
	return rpcPledges
}

func AmountBytesToString(amt []byte) string {
	return new(big.Int).SetBytes(amt).String()
}

func TokenBytesToString(token []byte) string {
	tk, _ := types.BytesToTokenTypeId(token)
	return tk.String()
}

type SimpleAccountInfo struct {
	Token     string `json:"token"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
}

type SimpleUserFund struct {
	Address  string               `json:"address"`
	Accounts []*SimpleAccountInfo `json:"accounts"`
}

type UserFunds struct {
	Funds []*SimpleUserFund `json:"funds"`
}

type RpcVxMineInfo struct {
	HistoryMinedSum string           `json:"historyMinedSum"`
	Total           string           `json:"total"`
	FeeMineTotal    string           `json:"feeMineTotal"`
	FeeMineDetail   map[int32]string `json:"feeMineDetail"`
	PledgeMine      string           `json:"pledgeMine"`
	MakerMine       string           `json:"makerMine"`
}
