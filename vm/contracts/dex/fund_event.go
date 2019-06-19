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

func doEmitEventLog(db vm_db.VmDb, event DexEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}
