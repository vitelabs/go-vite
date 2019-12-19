package dex

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/chain"
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
	TakerBrokerFeeRate int32  `json:"takerBrokerFeeRate"`
	MakerBrokerFeeRate int32  `json:"makerBrokerFeeRate"`
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
			TakerBrokerFeeRate: mkInfo.TakerOperatorFeeRate,
			MakerBrokerFeeRate: mkInfo.MakerOperatorFeeRate,
			AllowMine:          mkInfo.AllowMining,
			Valid:              mkInfo.Valid,
			Owner:              owner.String(),
			Creator:            creator.String(),
			Stopped:            mkInfo.Stopped,
			Timestamp:          mkInfo.Timestamp,
		}
	}
	return rmk
}

type NewRpcMarketInfo struct {
	MarketId             int32  `json:"marketId"`
	MarketSymbol         string `json:"marketSymbol"`
	TradeToken           string `json:"tradeToken"`
	QuoteToken           string `json:"quoteToken"`
	QuoteTokenType       int32  `json:"quoteTokenType"`
	TradeTokenDecimals   int32  `json:"tradeTokenDecimals,omitempty"`
	QuoteTokenDecimals   int32  `json:"quoteTokenDecimals"`
	TakerOperatorFeeRate int32  `json:"takerOperatorFeeRate"`
	MakerOperatorFeeRate int32  `json:"makerOperatorFeeRate"`
	AllowMining          bool   `json:"allowMining"`
	Valid                bool   `json:"valid"`
	Owner                string `json:"owner"`
	Creator              string `json:"creator"`
	Stopped              bool   `json:"stopped"`
	Timestamp            int64  `json:"timestamp"`
}

func MarketInfoToNewRpc(mkInfo *dex.MarketInfo) *NewRpcMarketInfo {
	var rmk *NewRpcMarketInfo = nil
	if mkInfo != nil {
		tradeToken, _ := types.BytesToTokenTypeId(mkInfo.TradeToken)
		quoteToken, _ := types.BytesToTokenTypeId(mkInfo.QuoteToken)
		owner, _ := types.BytesToAddress(mkInfo.Owner)
		creator, _ := types.BytesToAddress(mkInfo.Creator)
		rmk = &NewRpcMarketInfo{
			MarketId:             mkInfo.MarketId,
			MarketSymbol:         mkInfo.MarketSymbol,
			TradeToken:           tradeToken.String(),
			QuoteToken:           quoteToken.String(),
			QuoteTokenType:       mkInfo.QuoteTokenType,
			TradeTokenDecimals:   mkInfo.TradeTokenDecimals,
			QuoteTokenDecimals:   mkInfo.QuoteTokenDecimals,
			TakerOperatorFeeRate: mkInfo.TakerOperatorFeeRate,
			MakerOperatorFeeRate: mkInfo.MakerOperatorFeeRate,
			AllowMining:          mkInfo.AllowMining,
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
	NotRoll            bool   `json:"notRoll"`
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
	FinishDividend  bool                  `json:"finishDividend"`
	FinishMine      bool                  `json:"finishMine"`
}

func DexFeesByPeriodToRpc(dexFeesByPeriod *dex.DexFeesByPeriod) *RpcDexFeesByPeriod {
	if dexFeesByPeriod == nil {
		return nil
	}
	rpcDexFeesByPeriod := &RpcDexFeesByPeriod{}
	for _, dividend := range dexFeesByPeriod.FeesForDividend {
		rpcDividend := &RpcFeesForDividend{}
		rpcDividend.Token = TokenBytesToString(dividend.Token)
		rpcDividend.DividendPoolAmount = AmountBytesToString(dividend.DividendPoolAmount)
		rpcDividend.NotRoll = dividend.NotRoll
		rpcDexFeesByPeriod.FeesForDividend = append(rpcDexFeesByPeriod.FeesForDividend, rpcDividend)
	}
	for _, mine := range dexFeesByPeriod.FeesForMine {
		rpcMine := &RpcFeesForMine{}
		rpcMine.QuoteTokenType = mine.QuoteTokenType
		rpcMine.BaseAmount = AmountBytesToString(mine.BaseAmount)
		rpcMine.InviteBonusAmount = AmountBytesToString(mine.InviteBonusAmount)
		rpcDexFeesByPeriod.FeesForMine = append(rpcDexFeesByPeriod.FeesForMine, rpcMine)
	}
	rpcDexFeesByPeriod.LastValidPeriod = dexFeesByPeriod.LastValidPeriod
	rpcDexFeesByPeriod.FinishDividend = dexFeesByPeriod.FinishDividend
	rpcDexFeesByPeriod.FinishMine = dexFeesByPeriod.FinishMine
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

type RpcMiningStakingByPeriod struct {
	Period uint64 `json:"period"`
	Amount string `json:"amount"`
}

type RpcMiningStakings struct {
	Pledges []*RpcMiningStakingByPeriod `json:"Pledges"`
}

func MiningStakingsToRpc(miningStakings *dex.MiningStakings) *RpcMiningStakings {
	rpcPledges := &RpcMiningStakings{}
	for _, staking := range miningStakings.Stakings {
		rpcStaking := &RpcMiningStakingByPeriod{}
		rpcStaking.Period = staking.Period
		rpcStaking.Amount = AmountBytesToString(staking.Amount)
		rpcPledges.Pledges = append(rpcPledges.Pledges, rpcStaking)
	}
	return rpcPledges
}

type SimpleAccountInfo struct {
	Token     string `json:"token"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
}

type SimpleFund struct {
	Address  string               `json:"address"`
	Accounts []*SimpleAccountInfo `json:"accounts"`
}

type Funds struct {
	Funds []*SimpleFund `json:"funds"`
}

type RpcVxMineInfo struct {
	HistoryMinedSum string           `json:"historyMinedSum"`
	Total           string           `json:"total"`
	FeeMineTotal    string           `json:"feeMineTotal"`
	FeeMineDetail   map[int32]string `json:"feeMineDetail"`
	PledgeMine      string           `json:"pledgeMine"`
	MakerMine       string           `json:"makerMine"`
}

type NewRpcVxMineInfo struct {
	HistoryMinedSum string           `json:"historyMinedSum"`
	Total           string           `json:"total"`
	FeeMineTotal    string           `json:"feeMineTotal"`
	FeeMineDetail   map[int32]string `json:"feeMineDetail"`
	StakingMine     string           `json:"stakingMine"`
	MakerMine       string           `json:"makerMine"`
}

type RpcOrder struct {
	Id                   string `json:"Id"`
	Address              string `json:"Address"`
	MarketId             int32  `json:"MarketId"`
	Side                 bool   `json:"Side"`
	Type                 int32  `json:"Type"`
	Price                string `json:"Price"`
	TakerFeeRate         int32  `json:"TakerFeeRate"`
	MakerFeeRate         int32  `json:"MakerFeeRate"`
	TakerOperatorFeeRate int32  `json:"TakerOperatorFeeRate"`
	MakerOperatorFeeRate int32  `json:"MakerOperatorFeeRate"`
	Quantity             string `json:"Quantity"`
	Amount               string `json:"Amount"`
	LockedBuyFee         string `json:"LockedBuyFee,omitempty"`
	Status               int32  `json:"Status"`
	CancelReason         int32  `json:"CancelReason,omitempty"`
	ExecutedQuantity     string `json:"ExecutedQuantity,omitempty"`
	ExecutedAmount       string `json:"ExecutedAmount,omitempty"`
	ExecutedBaseFee      string `json:"ExecutedBaseFee,omitempty"`
	ExecutedOperatorFee  string `json:"ExecutedOperatorFee,omitempty"`
	RefundToken          string `json:"RefundToken,omitempty"`
	RefundQuantity       string `json:"RefundQuantity,omitempty"`
	Timestamp            int64  `json:"Timestamp"`
	Agent                string `json:"Agent,omitempty"`
	SendHash             string `json:"SendHash,omitempty"`
}

type OrdersRes struct {
	Orders []*RpcOrder `json:"orders,omitempty"`
	Size   int         `json:"size"`
}

func OrderToRpc(order *dex.Order) *RpcOrder {
	if order == nil {
		return nil
	}
	address, _ := types.BytesToAddress(order.Address)
	rpcOrder := &RpcOrder{}
	rpcOrder.Id = hex.EncodeToString(order.Id)
	rpcOrder.Address = address.String()
	rpcOrder.MarketId = order.MarketId
	rpcOrder.Side = order.Side
	rpcOrder.Type = order.Type
	rpcOrder.Price = dex.BytesToPrice(order.Price)
	rpcOrder.TakerFeeRate = order.TakerFeeRate
	rpcOrder.MakerFeeRate = order.MakerFeeRate
	rpcOrder.TakerOperatorFeeRate = order.TakerOperatorFeeRate
	rpcOrder.MakerOperatorFeeRate = order.MakerOperatorFeeRate
	rpcOrder.Quantity = AmountBytesToString(order.Quantity)
	rpcOrder.Amount = AmountBytesToString(order.Amount)
	if len(order.LockedBuyFee) > 0 {
		rpcOrder.LockedBuyFee = AmountBytesToString(order.LockedBuyFee)
	}
	rpcOrder.Status = order.Status
	rpcOrder.CancelReason = order.CancelReason
	if len(order.ExecutedQuantity) > 0 {
		rpcOrder.ExecutedQuantity = AmountBytesToString(order.ExecutedQuantity)
	}
	if len(order.ExecutedAmount) > 0 {
		rpcOrder.ExecutedAmount = AmountBytesToString(order.ExecutedAmount)
	}
	if len(order.ExecutedBaseFee) > 0 {
		rpcOrder.ExecutedBaseFee = AmountBytesToString(order.ExecutedBaseFee)
	}
	if len(order.ExecutedOperatorFee) > 0 {
		rpcOrder.ExecutedOperatorFee = AmountBytesToString(order.ExecutedOperatorFee)
	}
	if len(order.RefundToken) > 0 {
		tk, _ := types.BytesToTokenTypeId(order.RefundToken)
		rpcOrder.RefundToken = tk.String()
	}
	if len(order.RefundQuantity) > 0 {
		rpcOrder.RefundQuantity = AmountBytesToString(order.RefundQuantity)
	}
	if len(order.Agent) > 0 {
		agent, _ := types.BytesToAddress(order.Agent)
		rpcOrder.Agent = agent.String()
	}
	if len(order.SendHash) > 0 {
		sendHash, _ := types.BytesToHash(order.SendHash)
		rpcOrder.SendHash = sendHash.String()
	}
	rpcOrder.Timestamp = order.Timestamp
	return rpcOrder
}

func OrdersToRpc(orders []*dex.Order) []*RpcOrder {
	if len(orders) == 0 {
		return nil
	} else {
		rpcOrders := make([]*RpcOrder, len(orders))
		for i := 0; i < len(orders); i++ {
			rpcOrders[i] = OrderToRpc(orders[i])
		}
		return rpcOrders
	}
}

type StakeInfoList struct {
	StakeAmount string       `json:"totalStakeAmount"`
	Count       int          `json:"totalStakeCount"`
	StakeList   []*StakeInfo `json:"stakeList"`
}

type StakeInfo struct {
	Amount           string `json:"stakeAmount"`
	Beneficiary      string `json:"beneficiary"`
	ExpirationHeight string `json:"expirationHeight"`
	ExpirationTime   int64  `json:"expirationTime"`
	IsDelegated      bool   `json:"isDelegated"`
	DelegateAddress  string `json:"delegateAddress"`
	StakeAddress     string `json:"stakeAddress"`
	Bid              uint8  `json:"bid"`
	Id               string `json:"id,omitempty"`
	Principal        string `json:"principal,omitempty"`
}

type VxUnlockList struct {
	UnlockingAmount string      `json:"unlockingAmount"`
	Count           int         `json:"count"`
	Unlocks         []*VxUnlock `json:"unlocks"`
}

type VxUnlock struct {
	Amount           string `json:"amount"`
	ExpirationTime   int64  `json:"expirationTime"`
	ExpirationPeriod uint64 `json:"expirationPeriod"`
}

type CancelStakeList struct {
	CancellingAmount string         `json:"cancellingAmount"`
	Count            int            `json:"count"`
	Cancels          []*CancelStake `json:"cancels"`
}

type CancelStake struct {
	Amount           string `json:"amount"`
	ExpirationTime   int64  `json:"expirationTime"`
	ExpirationPeriod uint64 `json:"expirationPeriod"`
}

func UnlockListToRpc(unlocks *dex.VxUnlocks, pageIndex int, pageSize int, chain chain.Chain) *VxUnlockList {
	genesisTime := chain.GetGenesisSnapshotBlock().Timestamp.Unix()
	total := new(big.Int)
	vxUnlockList := new(VxUnlockList)
	var count = 0
	for i, ul := range unlocks.Unlocks {
		amt := new(big.Int).SetBytes(ul.Amount)
		if i >= pageIndex*pageSize && i < (pageIndex+1)*pageSize {
			unlock := new(VxUnlock)
			unlock.Amount = amt.String()
			unlock.ExpirationTime = genesisTime + int64((ul.PeriodId+1+uint64(dex.SchedulePeriods))*3600*24)
			unlock.ExpirationPeriod = ul.PeriodId + 1 + uint64(dex.SchedulePeriods)
			vxUnlockList.Unlocks = append(vxUnlockList.Unlocks, unlock)
		}
		total.Add(total, amt)
		count++
	}
	vxUnlockList.UnlockingAmount = total.String()
	vxUnlockList.Count = count
	return vxUnlockList
}

func CancelStakeListToRpc(cancelStakes *dex.CancelStakes, pageIndex int, pageSize int, chain chain.Chain) *CancelStakeList {
	genesisTime := chain.GetGenesisSnapshotBlock().Timestamp.Unix()
	total := new(big.Int)
	cancelStakeList := new(CancelStakeList)
	var count = 0
	for i, ul := range cancelStakes.Cancels {
		amt := new(big.Int).SetBytes(ul.Amount)
		if i >= pageIndex*pageSize && i < (pageIndex+1)*pageSize {
			cancel := new(CancelStake)
			cancel.Amount = amt.String()
			cancel.ExpirationTime = genesisTime + int64((ul.PeriodId+1+uint64(dex.SchedulePeriods))*3600*24) + 1200
			cancel.ExpirationPeriod = ul.PeriodId + 1 + uint64(dex.SchedulePeriods)
			cancelStakeList.Cancels = append(cancelStakeList.Cancels, cancel)
		}
		total.Add(total, amt)
		count++
	}
	cancelStakeList.CancellingAmount = total.String()
	cancelStakeList.Count = count
	return cancelStakeList
}

type DelegateStakeInfo struct {
	StakeType int    `json:"stakeType"`
	Address   string `json:"address"`
	Principal string `json:"principal"`
	Amount    string `json:"amount"`
	Status    int    `json:"status"`
}

func DelegateStakeInfoToRpc(info *dex.DelegateStakeInfo) *DelegateStakeInfo {
	rpcInfo := new(DelegateStakeInfo)
	rpcInfo.StakeType = int(info.StakeType)
	addr, _ := types.BytesToAddress(info.Address)
	rpcInfo.Address = addr.String()
	if len(info.Principal) > 0 {
		prAddr, _ := types.BytesToAddress(info.Principal)
		rpcInfo.Principal = prAddr.String()
	}
	rpcInfo.Amount = AmountBytesToString(info.Amount)
	rpcInfo.Status = int(info.Status)
	return rpcInfo
}

type VIPStakingRpc struct {
	Amount           string `json:"stakeAmount"`
	ExpirationHeight string `json:"expirationHeight"`
	ExpirationTime   int64  `json:"expirationTime"`
	Id               string `json:"id,omitempty"`
}
