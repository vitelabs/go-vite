package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func AddMinedVxForTradeFeeEventLog(db vm_db.VmDb, address types.Address, quoteTokenType int, feeAmount []byte, vxMined *big.Int) {
	event := &MinedVxForTradeFeeEvent{}
	event.Address = address.Bytes()
	event.QuoteTokenType = int32(quoteTokenType)
	event.FeeAmount = feeAmount
	event.MinedAmount = vxMined.Bytes()
	DoEmitEventLog(db, event)
}

func AddTokenEventLog(db vm_db.VmDb, tokenInfo *TokenInfo) {
	event := &TokenEvent{}
	event.TokenInfo = tokenInfo.TokenInfo
	DoEmitEventLog(db, event)
}

func AddMarketEventLog(db vm_db.VmDb, marketInfo *MarketInfo) {
	event := &MarketEvent{}
	event.MarketInfo = marketInfo.MarketInfo
	DoEmitEventLog(db, event)
}

func AddBrokerFeeDividendLog(db vm_db.VmDb, address types.Address, brokerMarketFee *dexproto.BrokerMarketFee) {
	event := &BrokerFeeDividendEvent{}
	event.Address = address.Bytes()
	event.MarketId = brokerMarketFee.MarketId
	event.TakerBrokerFeeRate = brokerMarketFee.TakerBrokerFeeRate
	event.MakerBrokerFeeRate = brokerMarketFee.MakerBrokerFeeRate
	event.Amount = brokerMarketFee.Amount
	DoEmitEventLog(db, event)
}

func DoEmitEventLog(db vm_db.VmDb, event DexEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}
