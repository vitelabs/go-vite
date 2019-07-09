package abi

import (
	"github.com/vitelabs/go-vite/vm/abi"
	"strings"
)

const (
	jsonDexFund = `
	[
        {"type":"function","name":"DexFundUserDeposit", "inputs":[]},
        {"type":"function","name":"DexFundUserWithdraw", "inputs":[{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
        {"type":"function","name":"DexFundNewMarket", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
        {"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"uint8"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
        {"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]},
        {"type":"function","name":"DexFundPeriodJob", "inputs":[{"name":"periodId","type":"uint64"},{"name":"bizType","type":"uint8"}]},
        {"type":"function","name":"DexFundPledgeForVx", "inputs":[{"name":"actionType","type":"uint8"}, {"name":"amount","type":"uint256"}]},
        {"type":"function","name":"DexFundPledgeForVip", "inputs":[{"name":"actionType","type":"uint8"}]},
        {"type":"function","name":"AgentPledgeCallback", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
        {"type":"function","name":"AgentCancelPledgeCallback", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
        {"type":"function","name":"GetTokenInfoCallback", "inputs":[{"name":"tokenId","type":"tokenId"},{"name":"bid","type":"uint8"},{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"},{"name":"owner","type":"address"}]},
        {"type":"function","name":"DexFundOwnerConfig", "inputs":[{"name":"operationCode","type":"uint8"},{"name":"owner","type":"address"},{"name":"timer","type":"address"},{"name":"trigger","type":"address"},{"name":"stopViteX","type":"bool"},{"name":"makerMineProxy","type":"address"},{"name":"maintainer","type":"address"}]},
        {"type":"function","name":"DexFundOwnerConfigTrade", "inputs":[{"name":"operationCode","type":"uint8"},{"name":"tradeToken","type":"tokenId"},{"name":"quoteToken","type":"tokenId"},{"name":"allowMine","type":"bool"},{"name":"newQuoteToken","type":"tokenId"},{"name":"quoteTokenType","type":"uint8"},{"name":"tokenType4TradeThr","type":"uint8"},{"name":"tradeThreshold","type":"uint256"},{"name":"tokenType4MineThr","type":"uint8"},{"name":"mineThreshold","type":"uint256"}]},
        {"type":"function","name":"DexFundMarketOwnerConfig", "inputs":[{"name":"operationCode","type":"uint8"},{"name":"tradeToken","type":"tokenId"},{"name":"quoteToken","type":"tokenId"},{"name":"owner","type":"address"},{"name":"takerFeeRate","type":"int32"},{"name":"makerFeeRate","type":"int32"},{"name":"stopMarket","type":"bool"}]},
		{"type":"function","name":"DexFundTransferTokenOwner", "inputs":[{"name":"token","type":"tokenId"}, {"name":"owner","type":"address"}]},
		{"type":"function","name":"NotifyTime", "inputs":[{"name":"timestamp","type":"int64"}]},
		{"type":"function","name":"DexFundNewInviter", "inputs":[]},
		{"type":"function","name":"DexFundBindInviteCode", "inputs":[{"name":"code","type":"uint32"}]},
		{"type":"function","name":"DexFundEndorseVx", "inputs":[]},
		{"type":"function","name":"DexFunSettleMakerMinedVx", "inputs":[{"name":"data","type":"bytes"}]}
    ]`

	MethodNameDexFundUserDeposit          = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw         = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder             = "DexFundNewOrder"
	MethodNameDexFundSettleOrders         = "DexFundSettleOrders"
	MethodNameDexFundPeriodJob            = "DexFundPeriodJob"
	MethodNameDexFundNewMarket            = "DexFundNewMarket"
	MethodNameDexFundPledgeForVx          = "DexFundPledgeForVx"
	MethodNameDexFundPledgeForVip         = "DexFundPledgeForVip"
	MethodNameDexFundPledgeCallback       = "AgentPledgeCallback"
	MethodNameDexFundCancelPledgeCallback = "AgentCancelPledgeCallback"
	MethodNameDexFundGetTokenInfoCallback = "GetTokenInfoCallback"
	MethodNameDexFundOwnerConfig          = "DexFundOwnerConfig"
	MethodNameDexFundOwnerConfigTrade     = "DexFundOwnerConfigTrade"
	MethodNameDexFundMarketOwnerConfig    = "DexFundMarketOwnerConfig"
	MethodNameDexFundTransferTokenOwner   = "DexFundTransferTokenOwner"
	MethodNameDexFundNotifyTime           = "NotifyTime"
	MethodNameDexFundNewInviter           = "DexFundNewInviter"
	MethodNameDexFundBindInviteCode       = "DexFundBindInviteCode"
	MethodNameDexFundEndorseVxMinePool    = "DexFundEndorseVx"
	MethodNameDexFunSettleMakerMinedVx    = "DexFunSettleMakerMinedVx"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)
