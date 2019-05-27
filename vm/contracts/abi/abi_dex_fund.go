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
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"int8"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexFundFeeDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundMinedVxDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundNewMarket", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundSetOwner", "inputs":[{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"DexFundConfigMineMarket", "inputs":[{"name":"allowMine","type":"bool"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundPledgeForVx", "inputs":[{"name":"actionType","type":"int8"}, {"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundPledgeForVip", "inputs":[{"name":"actionType","type":"int8"}]},
		{"type":"function","name":"AgentPledgeCallback", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"function","name":"AgentCancelPledgeCallback", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"function","name":"GetTokenInfoCallback", "inputs":[{"name":"tokenId","type":"tokenId"},{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"}]},
		{"type":"function","name":"DexFundConfigTimerAddress", "inputs":[{"name":"address","type":"address"}]},
		{"type":"function","name":"NotifyTime", "inputs":[{"name":"timestamp","type":"int64"}]}
	]`

	MethodNameDexFundUserDeposit          = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw         = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder             = "DexFundNewOrder"
	MethodNameDexFundSettleOrders         = "DexFundSettleOrders"
	MethodNameDexFundFeeDividend          = "DexFundFeeDividend"
	MethodNameDexFundMinedVxDividend      = "DexFundMinedVxDividend"
	MethodNameDexFundNewMarket            = "DexFundNewMarket"
	MethodNameDexFundSetOwner             = "DexFundSetOwner"
	MethodNameDexFundConfigMineMarket     = "DexFundConfigMineMarket"
	MethodNameDexFundPledgeForVx          = "DexFundPledgeForVx"
	MethodNameDexFundPledgeForVip         = "DexFundPledgeForVip"
	MethodNameDexFundPledgeCallback       = "AgentPledgeCallback"
	MethodNameDexFundCancelPledgeCallback = "AgentCancelPledgeCallback"
	MethodNameDexFundGetTokenInfoCallback = "GetTokenInfoCallback"
	MethodNameDexFundConfigTimerAddress   = "DexFundConfigTimerAddress"
	MethodNameDexFundNotifyTime           = "NotifyTime"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)