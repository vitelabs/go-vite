package dex

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
)

//Note: the 4th byte of trade db key must not be 0 or 1, in order to diff from orderId side value
var marketByMarketIdPrefix = []byte("mkIf:")

var tradeTimestampKey = []byte("ttmsp:")
var sendHashMapOrderIdPrefixKey = []byte("hmId:")

const CleanExpireOrdersMaxCount = 200

type ParamDexCancelOrder struct {
	OrderId []byte
}

func CleanExpireOrders(db vm_db.VmDb, orderIds []byte) (map[types.Address]map[bool]*dexproto.AccountSettle, *MarketInfo, error) {
	var (
		matcher       *Matcher
		marketId      int32
		currentTime   int64
		expiredMakers = make([]*Order, 0, len(orderIds)/OrderIdBytesLength)
		err           error
	)
	if currentTime = GetTradeTimestamp(db); currentTime == 0 {
		return nil, nil, NotSetTimestampErr
	}

	for i := 0; i < len(orderIds)/OrderIdBytesLength; i++ {
		var (
			order *Order
			mkId  int32
		)
		orderId := orderIds[i*OrderIdBytesLength : (i+1)*OrderIdBytesLength]
		if mkId, _, _, _, err = DeComposeOrderId(orderId); err != nil {
			return nil, nil, err
		} else {
			if marketId == 0 {
				marketId = mkId
			} else if mkId != marketId {
				return nil, nil, MultiMarketsInOneActionErr
			}
		}
		if matcher == nil {
			if matcher, err = NewMatcher(db, marketId); err != nil {
				return nil, nil, err
			}
		}
		if order, err = matcher.GetOrderById(orderId); err == OrderNotExistsErr {
			continue
		} else if err != nil {
			return nil, nil, err
		}
		if filterTimeout(db, order) {
			expiredMakers = append(expiredMakers, order)
		}
	}
	if matcher != nil && len(expiredMakers) > 0 {
		matcher.handleModifiedMakers(expiredMakers)
		return matcher.GetFundSettles(), matcher.MarketInfo, nil
	} else {
		return nil, nil, nil
	}
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

func SetTradeTimestamp(db vm_db.VmDb, timestamp int64) {
	setValueToDb(db, tradeTimestampKey, Uint64ToBytes(uint64(timestamp)))
}

func GetTradeTimestamp(db vm_db.VmDb) int64 {
	if bs := getValueFromDb(db, tradeTimestampKey); len(bs) == 8 {
		return int64(BytesToUint64(bs))
	} else {
		return 0
	}
}

func SaveHashMapOrderId(db vm_db.VmDb, sendHash []byte, orderId []byte) {
	setValueToDb(db, GetHashMapOrderIdKey(sendHash), orderId)
}

func GetOrderIdByHash(db vm_db.VmDb, sendHash []byte) ([]byte, bool) {
	if orderId := getValueFromDb(db, GetHashMapOrderIdKey(sendHash)); len(orderId) == OrderIdBytesLength {
		return orderId, true
	} else {
		return nil, false
	}
}

func DeleteHashMapOrderId(db vm_db.VmDb, sendHash []byte) {
	setValueToDb(db, GetHashMapOrderIdKey(sendHash), nil)
}

func GetHashMapOrderIdKey(sendHash []byte) []byte {
	return append(sendHashMapOrderIdPrefixKey, sendHash[len(sendHashMapOrderIdPrefixKey):]...)
}

func TryUpdateTimestamp(db vm_db.VmDb, timestamp int64, preHash types.Hash) {
	header := uint8(preHash[0])
	if header < 32 {
		SetTradeTimestamp(db, timestamp)
	}
}

type AccountSettleSorter []*dexproto.AccountSettle

func (st AccountSettleSorter) Len() int {
	return len(st)
}

func (st AccountSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st AccountSettleSorter) Less(i, j int) bool {
	return st[i].IsTradeToken
}

type FundSettleSorter []*dexproto.FundSettle

func (st FundSettleSorter) Len() int {
	return len(st)
}

func (st FundSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st FundSettleSorter) Less(i, j int) bool {
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
	addCmp := bytes.Compare(st[i].Address, st[j].Address)
	if addCmp < 0 {
		return true
	} else {
		return false
	}
}
