package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func AddTokenEvent(db vm_db.VmDb, tokenInfo *TokenInfo) {
	event := &TokenEvent{}
	event.TokenInfo = tokenInfo.TokenInfo
	doEmitEventLog(db, event)
}

func AddMarketEvent(db vm_db.VmDb, marketInfo *MarketInfo) {
	event := &MarketEvent{}
	event.MarketInfo = marketInfo.MarketInfo
	doEmitEventLog(db, event)
}

func AddPeriodWithBizEvent(db vm_db.VmDb, periodId uint64, bizType uint8) {
	event := &PeriodWithBizEvent{}
	event.Period = periodId
	event.BizType = int32(bizType)
	doEmitEventLog(db, event)
}

func AddFeeDividendEvent(db vm_db.VmDb, address types.Address, feeToken types.TokenTypeId, vxAmount, feeDividend *big.Int) {
	event := &FeeDividendEvent{}
	event.Address = address.Bytes()
	event.VxAmount = vxAmount.Bytes()
	event.FeeToken = feeToken.Bytes()
	event.FeeDividend = feeDividend.Bytes()
	doEmitEventLog(db, event)
}

func AddBrokerFeeDividendEvent(db vm_db.VmDb, address types.Address, brokerMarketFee *dexproto.BrokerMarketFee) {
	event := &BrokerFeeDividendEvent{}
	event.Address = address.Bytes()
	event.MarketId = brokerMarketFee.MarketId
	event.TakerBrokerFeeRate = brokerMarketFee.TakerBrokerFeeRate
	event.MakerBrokerFeeRate = brokerMarketFee.MakerBrokerFeeRate
	event.Amount = brokerMarketFee.Amount
	doEmitEventLog(db, event)
}

func AddMinedVxForTradeFeeEvent(db vm_db.VmDb, address types.Address, quoteTokenType int32, feeAmount []byte, vxMined *big.Int) {
	event := &MinedVxForTradeFeeEvent{}
	event.Address = address.Bytes()
	event.QuoteTokenType = quoteTokenType
	event.FeeAmount = feeAmount
	event.MinedAmount = vxMined.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForInviteeFeeEvent(db vm_db.VmDb, address types.Address, quoteTokenType int32, feeAmount []byte, vxMined *big.Int) {
	event := &MinedVxForInviteeFeeEvent{}
	event.Address = address.Bytes()
	event.QuoteTokenType = quoteTokenType
	event.FeeAmount = feeAmount
	event.MinedAmount = vxMined.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForPledgeEvent(db vm_db.VmDb, address types.Address, pledgeAmt, minedAmt *big.Int) {
	event := &MinedVxForPledgeEvent{}
	event.Address = address.Bytes()
	event.PledgeAmount = pledgeAmt.Bytes()
	event.MinedAmount = minedAmt.Bytes()
	doEmitEventLog(db, event)
}

func AddMinedVxForOperationEvent(db vm_db.VmDb, bizType int32, address types.Address, amount *big.Int) {
	event := &MinedVxForOperationEvent{}
	event.BizType = bizType
	event.Address = address.Bytes()
	event.Amount = amount.Bytes()
	doEmitEventLog(db, event)
}

func AddInviteRelationEvent(db vm_db.VmDb, inviter, invitee types.Address, inviteCode uint32) {
	event := &InviteRelationEvent{}
	event.Inviter = inviter.Bytes()
	event.Invitee = invitee.Bytes()
	event.InviteCode = inviteCode
	doEmitEventLog(db, event)
}

func AddSettleMakerMinedVxEvent(db vm_db.VmDb, periodId uint64, page int32, finish bool) {
	event := &SettleMakerMinedVxEvent{}
	event.PeriodId = periodId
	event.Page = page
	event.Finish = finish
	doEmitEventLog(db, event)
}

func AddErrEvent(db vm_db.VmDb, err error) {
	event := &ErrEvent{}
	event.error = err
	doEmitEventLog(db, event)
}

func doEmitEventLog(db vm_db.VmDb, event DexEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}
