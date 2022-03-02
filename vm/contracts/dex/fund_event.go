package dex

import (
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	dexproto "github.com/vitelabs/go-vite/v2/vm/contracts/dex/proto"
)

func AddTokenEvent(db interfaces.VmDb, tokenInfo *TokenInfo) {
	event := &TokenEvent{}
	event.TokenInfo = tokenInfo.TokenInfo
	doEmitEventLog(db, event)
}

func AddMarketEvent(db interfaces.VmDb, marketInfo *MarketInfo) {
	event := &MarketEvent{}
	event.MarketInfo = marketInfo.MarketInfo
	doEmitEventLog(db, event)
}

func AddPeriodWithBizEvent(db interfaces.VmDb, periodId uint64, bizType uint8) {
	event := &PeriodJobWithBizEvent{}
	event.Period = periodId
	event.BizType = int32(bizType)
	doEmitEventLog(db, event)
}

func AddFeeDividendEvent(db interfaces.VmDb, address types.Address, feeToken types.TokenTypeId, vxAmount, feeDividend *big.Int) {
	event := &FeeDividendEvent{}
	event.Address = address.Bytes()
	event.VxAmount = vxAmount.Bytes()
	event.FeeToken = feeToken.Bytes()
	event.FeeDividend = feeDividend.Bytes()
	doEmitEventLog(db, event)
}

func AddOperatorFeeDividendEvent(db interfaces.VmDb, address types.Address, operatorMarketFee *dexproto.OperatorMarketFee) {
	event := &OperatorFeeDividendEvent{}
	event.Address = address.Bytes()
	event.MarketId = operatorMarketFee.MarketId
	event.TakerOperatorFeeRate = operatorMarketFee.TakerOperatorFeeRate
	event.MakerOperatorFeeRate = operatorMarketFee.MakerOperatorFeeRate
	event.Amount = operatorMarketFee.Amount
	doEmitEventLog(db, event)
}

func AddMinedVxForTradeFeeEvent(db interfaces.VmDb, address types.Address, quoteTokenType int32, feeAmount []byte, vxMined *big.Int) {
	event := &MinedVxForTradeFeeEvent{}
	event.Address = address.Bytes()
	event.QuoteTokenType = quoteTokenType
	event.FeeAmount = feeAmount
	event.MinedAmount = vxMined.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForInviteeFeeEvent(db interfaces.VmDb, address types.Address, quoteTokenType int32, feeAmount []byte, vxMined *big.Int) {
	event := &MinedVxForInviteeFeeEvent{}
	event.Address = address.Bytes()
	event.QuoteTokenType = quoteTokenType
	event.FeeAmount = feeAmount
	event.MinedAmount = vxMined.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForStakingEvent(db interfaces.VmDb, address types.Address, stakedAmt, minedAmt *big.Int) {
	event := &MinedVxForStakingEvent{}
	event.Address = address.Bytes()
	event.StakedAmount = stakedAmt.Bytes()
	event.MinedAmount = minedAmt.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForOperationEvent(db interfaces.VmDb, bizType int32, address types.Address, amount *big.Int) {
	event := &MinedVxForOperationEvent{}
	event.BizType = bizType
	event.Address = address.Bytes()
	event.Amount = amount.Bytes()
	doEmitEventLog(db, event)
}

func AddInviteRelationEvent(db interfaces.VmDb, inviter, invitee types.Address, inviteCode uint32) {
	event := &InviteRelationEvent{}
	event.Inviter = inviter.Bytes()
	event.Invitee = invitee.Bytes()
	event.InviteCode = inviteCode
	doEmitEventLog(db, event)
}

func AddSettleMakerMinedVxEvent(db interfaces.VmDb, periodId uint64, page int32, finish bool) {
	event := &SettleMakerMinedVxEvent{}
	event.PeriodId = periodId
	event.Page = page
	event.Finish = finish
	doEmitEventLog(db, event)
}

func AddGrantMarketToAgentEvent(db interfaces.VmDb, principal, agent types.Address, marketId int32) {
	event := &GrantMarketToAgentEvent{}
	event.Principal = principal.Bytes()
	event.Agent = agent.Bytes()
	event.MarketId = marketId
	doEmitEventLog(db, event)
}

func AddRevokeMarketFromAgentEvent(db interfaces.VmDb, principal, agent types.Address, marketId int32) {
	event := &RevokeMarketFromAgentEvent{}
	event.Principal = principal.Bytes()
	event.Agent = agent.Bytes()
	event.MarketId = marketId
	doEmitEventLog(db, event)
}

func AddBurnViteEvent(db interfaces.VmDb, bizType int, amount *big.Int) {
	event := &BurnViteEvent{}
	event.BizType = int32(bizType)
	event.Amount = amount.Bytes()
	doEmitEventLog(db, event)
}

func AddTransferAssetEvent(db interfaces.VmDb, bizType int, from, to types.Address, token types.TokenTypeId, amount *big.Int, extra []byte) {
	event := &TransferAssetEvent{}
	event.BizType = int32(bizType)
	event.From = from.Bytes()
	event.To = to.Bytes()
	event.Token = token.Bytes()
	event.Amount = amount.Bytes()
	event.Extra = extra
	doEmitEventLog(db, event)
}

func AddErrEvent(db interfaces.VmDb, err error) {
	event := &ErrEvent{}
	event.error = err
	doEmitEventLog(db, event)
}

func doEmitEventLog(db interfaces.VmDb, event DexEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}
