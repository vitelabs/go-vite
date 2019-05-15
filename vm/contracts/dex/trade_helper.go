package dex

import (
	"bytes"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
)

//Note: the 4th byte of trade db key must not be 0 or 1, in order to diff from orderId side value
var marketByMarketIdPrefix = []byte("mkIf:")

type ParamDexCancelOrder struct {
	OrderId    []byte
}

func GetMarketInfoById(db vm_db.VmDb, marketId int32) (marketInfo *MarketInfo, ok bool) {
	marketInfo = &MarketInfo{}
	ok = deserializeFromDb(db, GetMarketInfoKeyById(marketId), marketInfo)
	return
}

func SaveMarketInfoById(db vm_db.VmDb, marketInfo *MarketInfo) {
	serializeToDb(db, GetMarketInfoKeyById(marketInfo.MarketId), marketInfo)
}

func GetMarketInfoKeyById(marketId int32) []byte {
	return append(marketByMarketIdPrefix, Uint32ToBytes(uint32(marketId))...)
}

type FundSettleSorter []*dexproto.FundSettle

func (st FundSettleSorter) Len() int {
	return len(st)
}

func (st FundSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st FundSettleSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].Token, st[j].Token)
	if tkCmp < 0 {
		return true
	} else {
		return false
	}
}

type UserFundSettleSorter []*dexproto.UserFundSettle

func (st UserFundSettleSorter) Len() int {
	return len(st)
}

func (st UserFundSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st UserFundSettleSorter) Less(i, j int) bool {
	addCmp := bytes.Compare(st[i].Address, st[j].Address)
	if addCmp < 0 {
		return true
	} else {
		return false
	}
}

type UserFeeSettleSorter []*dexproto.UserFeeSettle

func (st UserFeeSettleSorter) Len() int {
	return len(st)
}

func (st UserFeeSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st UserFeeSettleSorter) Less(i, j int) bool {
	addCmp := bytes.Compare(st[i].Address, st[j].Address)
	if addCmp < 0 {
		return true
	} else {
		return false
	}
}

type FeeSettleSorter []*dexproto.FeeSettle

func (st FeeSettleSorter) Len() int {
	return len(st)
}

func (st FeeSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st FeeSettleSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].Token, st[j].Token)
	if tkCmp < 0 {
		return true
	} else {
		return false
	}
}
