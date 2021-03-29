package vm

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
)

type DexTradeCase struct {
	Name            string
	GlobalEnv       GlobalEnv
	DexTradeStorage *DexTradeStorage

	MarketActions []*MarketStorage
	OrderActions  []*OrderStorage

	CheckBooks  []*CheckBook
	CheckOrders []*CheckOrder
	CheckEvents []*CheckTradeEvent
}

type DexTradeStorage struct {
	Timestamp int64
	Markets   []*MarketStorage
	Orders    []*OrderStorage
}

type OrderStorage struct {
	Id                      []byte
	Address                 *types.Address
	MarketId                int32
	Side                    bool
	Type                    int32
	Price                   string
	TakerFeeRate            int32
	MakerFeeRate            int32
	TakerOperatorFeeRate    int32
	MakerOperatorFeeRate    int32
	Quantity                *big.Int
	Amount                  *big.Int
	LockedBuyFee            *big.Int
	Status                  int32
	CancelReason            int32
	ExecutedQuantity        *big.Int
	ExecutedAmount          *big.Int
	ExecutedBaseFee         *big.Int
	ExecutedOperatorFee     *big.Int
	RefundToken             *types.TokenTypeId
	RefundQuantity          *big.Int
	Timestamp               int64
	Agent                   *types.Address
	SendHash                *types.Hash
	MarketOrderAmtThreshold *big.Int
}

type CheckBook struct {
	MarketId int32
	Side     bool
	Size     int
}

type CheckOrder struct {
	MarketId   int32
	BuyOrders  []*OrderStorage
	SellOrders []*OrderStorage
}

type CheckTradeEvent struct {
	TopicName   string
	NewOrder    *NewOrderEvent
	OrderUpdate *OrderUpdateEvent
	Transaction *TransactionEvent
}

type NewOrderEvent struct {
	OrderStorage *OrderStorage
	TradeToken   *types.TokenTypeId
	QuoteToken   *types.TokenTypeId
}

type OrderUpdateEvent struct {
	TradeToken          *types.TokenTypeId
	QuoteToken          *types.TokenTypeId
	Status              int32
	CancelReason        int32
	ExecutedQuantity    *big.Int
	ExecutedAmount      *big.Int
	ExecutedBaseFee     *big.Int
	ExecutedOperatorFee *big.Int
	RefundToken         *types.TokenTypeId
	RefundQuantity      *big.Int
}

type TransactionEvent struct {
	TakerSide        bool
	TakerId          *types.Address
	MakerId          *types.Address
	Price            string
	Quantity         *big.Int
	Amount           *big.Int
	TakerFee         *big.Int
	MakerFee         *big.Int
	TakerOperatorFee *big.Int
	MakerOperatorFee *big.Int
}

func TestDexTrade(t *testing.T) {
	testDir := "./contracts/dex/test/trade/"
	testFiles, ok := ioutil.ReadDir(testDir)
	if ok != nil {
		t.Fatalf("read dir failed, %v", ok)
	}
	for _, testFile := range testFiles {
		if testFile.IsDir() {
			continue
		}
		file, ok := os.Open(testDir + testFile.Name())
		if ok != nil {
			t.Fatalf("open test file failed, %v", ok)
		}
		testCaseMap := new(map[string]*DexTradeCase)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file %v failed, %v", testFile.Name(), ok)
		}
		for k, testCase := range *testCaseMap {
			fmt.Println(testFile.Name() + ":" + k)
			db := initTradeDb(testCase, t)
			reader := util.NewVMConsensusReader(newConsensusReaderTest(db.GetGenesisSnapshotBlock().Timestamp.Unix(), 24*3600, nil))
			vm := NewVM(reader)
			executeTradeActions(testCase, vm, db, t)
			executeTradeChecks(testCase, db, t)
		}
	}
}

func executeTradeActions(testCase *DexTradeCase, vm *VM, db *testDatabase, t *testing.T) {
	db.addr = types.AddressDexTrade
	if testCase.MarketActions != nil {
		for _, ma := range testCase.MarketActions {
			mk := toDexMarketStorage(ma)
			mkdata, _ := mk.Serialize()
			data, _ := cabi.ABIDexTrade.PackMethod(cabi.MethodNameDexTradeSyncNewMarket, mkdata)
			doAction("syncNewMarket", db, vm, types.AddressDexFund, types.AddressDexTrade, data, t)
		}
	}
	if testCase.OrderActions != nil {
		for _, oa := range testCase.OrderActions {
			od := toDexOrder(db, oa)
			odData, _ := od.Serialize()
			data, _ := cabi.ABIDexTrade.PackMethod(cabi.MethodNameDexTradePlaceOrder, odData)
			doAction("placeTradeOrder", db, vm, types.AddressDexFund, types.AddressDexTrade, data, t)
		}
	}
}

func executeTradeChecks(testCase *DexTradeCase, db *testDatabase, t *testing.T) {
	db.addr = types.AddressDexTrade
	if testCase.CheckBooks != nil {
		for _, cb := range testCase.CheckBooks {
			mc, err := dex.NewMatcher(db, cb.MarketId)
			assert.Nil(t, err)
			_, size, err := mc.GetOrdersFromMarket(cb.Side, 0, cb.Size)
			assert.Equal(t, cb.Size, size)
		}
	}
	if testCase.CheckOrders != nil {
		for _, co := range testCase.CheckOrders {
			mc, err := dex.NewMatcher(db, co.MarketId)
			assert.Nil(t, err)
			if co.BuyOrders != nil {
				buyOrders, size, err := mc.GetOrdersFromMarket(false, 0, len(co.BuyOrders))
				assert.Nil(t, err)
				for i := 0; i < size; i++ {
					assertOrder(t, co.BuyOrders[i], &buyOrders[i].Order)
				}
			}
			if co.SellOrders != nil {
				sellOrders, size, err := mc.GetOrdersFromMarket(true, 0, len(co.SellOrders))
				assert.Nil(t, err)
				for i := 0; i < size; i++ {
					assertOrder(t, co.SellOrders[i], &sellOrders[i].Order)
				}
			}
		}
	}
	if testCase.CheckEvents != nil {
		assert.Equal(t, len(testCase.CheckEvents), len(db.logList))
		for i, cev := range testCase.CheckEvents {
			log := db.logList[i]
			assert.Equal(t, getTopicId(cev.TopicName), log.Topics[0])
			if cev.NewOrder != nil {
				no := &dex.NewOrderEvent{}
				no.FromBytes(log.Data)
				assertTokenIdEqual(t, cev.NewOrder.TradeToken, no.TradeToken)
				assertTokenIdEqual(t, cev.NewOrder.QuoteToken, no.QuoteToken)
			}
			if cev.OrderUpdate != nil {
				ou := &dex.OrderUpdateEvent{}
				ou.FromBytes(log.Data)
				assertTokenIdEqual(t, cev.OrderUpdate.TradeToken, ou.OrderUpdateInfo.TradeToken)
				assertTokenIdEqual(t, cev.OrderUpdate.QuoteToken, ou.OrderUpdateInfo.QuoteToken)
				assert.Equal(t, cev.OrderUpdate.Status, ou.OrderUpdateInfo.Status)
				assertAmountEqual(t, cev.OrderUpdate.ExecutedQuantity, ou.OrderUpdateInfo.ExecutedQuantity)
				assertAmountEqual(t, cev.OrderUpdate.ExecutedAmount, ou.OrderUpdateInfo.ExecutedAmount)
				assertAmountEqual(t, cev.OrderUpdate.ExecutedBaseFee, ou.OrderUpdateInfo.ExecutedBaseFee)
				assertAmountEqual(t, cev.OrderUpdate.ExecutedOperatorFee, ou.OrderUpdateInfo.ExecutedOperatorFee)
				assertTokenIdEqual(t, cev.OrderUpdate.RefundToken, ou.OrderUpdateInfo.RefundToken)
				assertAmountEqual(t, cev.OrderUpdate.RefundQuantity, ou.OrderUpdateInfo.RefundQuantity)
			}
			if cev.Transaction != nil {
				tx := &dex.TransactionEvent{}
				tx.FromBytes(log.Data)
				assert.True(t, cev.Transaction.TakerSide, tx.TakerSide)
				assertAddressEqual(t, cev.Transaction.TakerId, tx.TakerId)
				assertAddressEqual(t, cev.Transaction.MakerId, tx.MakerId)
				assertPriceEqual(t, cev.Transaction.Price, tx.Price)
				assertAmountEqual(t, cev.Transaction.Quantity, tx.Quantity)
				assertAmountEqual(t, cev.Transaction.Amount, tx.Amount)
				assertAmountEqual(t, cev.Transaction.TakerFee, tx.TakerFee)
				assertAmountEqual(t, cev.Transaction.MakerFee, tx.MakerFee)
				assertAmountEqual(t, cev.Transaction.TakerOperatorFee, tx.TakerOperatorFee)
				assertAmountEqual(t, cev.Transaction.MakerOperatorFee, tx.MakerOperatorFee)
			}
		}
	}
}

func initTradeDb(dexTradeCase *DexTradeCase, t *testing.T) *testDatabase {
	db := generateDb(dexTradeCase.Name, &dexTradeCase.GlobalEnv, t)
	if dexTradeCase.DexTradeStorage != nil {
		db.storageMap[types.AddressDexTrade] = make(map[string][]byte, 0)
		db.addr = types.AddressDexTrade
		if dexTradeCase.DexTradeStorage.Markets != nil {
			for _, mk := range dexTradeCase.DexTradeStorage.Markets {
				mkInfo := toDexMarketStorage(mk)
				dex.SaveMarketInfoById(db, mkInfo)
			}
		}
		if dexTradeCase.DexTradeStorage.Timestamp > 0 {
			dex.SetTradeTimestamp(db, dexTradeCase.DexTradeStorage.Timestamp)
			// save dexFund storage to dexTrade storage only for orderId compose
			db.storageMap[types.AddressDexTrade][ToKey([]byte("tts:"))] = dex.Uint64ToBytes(uint64(dexTradeCase.DexTradeStorage.Timestamp))
		}
		if dexTradeCase.DexTradeStorage.Orders != nil {
			for _, o := range dexTradeCase.DexTradeStorage.Orders {
				od := toDexOrder(db, o)
				saveOrder(db, od, true)
			}
		}
	}
	return db
}

func toDexOrder(db *testDatabase, o *OrderStorage) *dex.Order {
	od := &dex.Order{}
	od.Id = dex.ComposeOrderId(db, o.MarketId, o.Side, o.Price)
	od.Address = o.Address.Bytes()
	od.Side = o.Side
	od.Type = o.Type
	od.Price = dex.PriceToBytes(o.Price)
	od.TakerFeeRate = o.TakerFeeRate
	od.MakerFeeRate = o.MakerFeeRate
	od.TakerOperatorFeeRate = o.TakerOperatorFeeRate
	od.MakerOperatorFeeRate = o.MakerOperatorFeeRate
	od.Quantity = o.Quantity.Bytes()
	if o.Amount != nil {
		od.Amount = o.Amount.Bytes()
	}
	if o.LockedBuyFee != nil {
		od.LockedBuyFee = o.LockedBuyFee.Bytes()
	}
	od.Status = o.Status
	od.CancelReason = o.CancelReason
	if o.ExecutedQuantity != nil {
		od.ExecutedQuantity = o.ExecutedQuantity.Bytes()
	}
	if o.ExecutedAmount != nil {
		od.ExecutedAmount = o.ExecutedAmount.Bytes()
	}
	if o.ExecutedBaseFee != nil {
		od.ExecutedBaseFee = o.ExecutedBaseFee.Bytes()
	}
	if o.ExecutedOperatorFee != nil {
		od.ExecutedOperatorFee = o.ExecutedOperatorFee.Bytes()
	}
	if o.RefundToken != nil {
		od.RefundToken = o.RefundToken.Bytes()
	}
	if o.RefundQuantity != nil {
		od.RefundQuantity = o.RefundQuantity.Bytes()
	}
	od.Timestamp = o.Timestamp
	if o.Agent != nil {
		od.Agent = o.Agent.Bytes()
	}
	if o.SendHash != nil {
		od.SendHash = o.SendHash.Bytes()
	}
	if o.MarketOrderAmtThreshold != nil {
		od.MarketOrderAmtThreshold = o.MarketOrderAmtThreshold.Bytes()
	}
	return od
}

func saveOrder(db *testDatabase, order *dex.Order, isTaker bool) {
	orderId := order.Id
	if data, err := order.SerializeCompact(); err != nil {
		panic(err)
	} else {
		db.storageMap[types.AddressDexTrade][ToKey(orderId)] = data
	}
	if isTaker && len(order.SendHash) > 0 {
		dex.SaveHashMapOrderId(db, order.SendHash, orderId)
	}
}

func assertOrder(t *testing.T, checkOrder *OrderStorage, dbOrder *dexproto.Order) {
	assertAddressEqual(t, checkOrder.Address, dbOrder.Address)
	assert.Equal(t, checkOrder.Type, dbOrder.Type)
	assertPriceEqual(t, checkOrder.Price, dbOrder.Price)
	assert.Equal(t, checkOrder.TakerFeeRate, dbOrder.TakerFeeRate)
	assert.Equal(t, checkOrder.MakerFeeRate, dbOrder.MakerFeeRate)
	assert.Equal(t, checkOrder.TakerOperatorFeeRate, dbOrder.TakerOperatorFeeRate)
	assert.Equal(t, checkOrder.MakerOperatorFeeRate, dbOrder.MakerOperatorFeeRate)
	assertAmountEqual(t, checkOrder.Quantity, dbOrder.Quantity)
	assertAmountEqual(t, checkOrder.Amount, dbOrder.Amount)
	assertAmountEqual(t, checkOrder.LockedBuyFee, dbOrder.LockedBuyFee)
	assert.Equal(t, checkOrder.Status, dbOrder.Status)
	assert.Equal(t, checkOrder.CancelReason, dbOrder.CancelReason)
	assertAmountEqual(t, checkOrder.ExecutedQuantity, dbOrder.ExecutedQuantity)
	assertAmountEqual(t, checkOrder.ExecutedAmount, dbOrder.ExecutedAmount)
	assertAmountEqual(t, checkOrder.ExecutedBaseFee, dbOrder.ExecutedBaseFee)
	assertAmountEqual(t, checkOrder.ExecutedOperatorFee, dbOrder.ExecutedOperatorFee)
	assertAddressEqual(t, checkOrder.Agent, dbOrder.Agent)
	assertHashEqual(t, checkOrder.SendHash, dbOrder.SendHash)
	assertAmountEqual(t, checkOrder.MarketOrderAmtThreshold, dbOrder.MarketOrderAmtThreshold)
}

func assertAmountEqual(t *testing.T, amount *big.Int, amountBytes []byte) {
	if amount == nil {
		amount = big.NewInt(0)
	}
	amountCmp := big.NewInt(0).SetBytes(amountBytes)
	assert.True(t, amount.Cmp(amountCmp) == 0, fmt.Sprintf("assertAmountEqual expected %s, actual %s", amount.String(), amountCmp.String()))
}

func assertAddressEqual(t *testing.T, address *types.Address, addressBytes []byte) {
	if address == nil {
		address = &types.ZERO_ADDRESS
	}
	var addressCmp = &types.Address{}
	if len(addressBytes) > 0 {
		addressCmp.SetBytes(addressBytes)
	} else {
		addressCmp = &types.ZERO_ADDRESS
	}
	assert.Equal(t, address, addressCmp)
}

func assertTokenIdEqual(t *testing.T, token *types.TokenTypeId, tokenBytes []byte) {
	if token == nil {
		token = &types.ZERO_TOKENID
	}
	var tokenCmp = &types.TokenTypeId{}
	if len(tokenBytes) > 0 {
		tokenCmp.SetBytes(tokenBytes)
	} else {
		tokenCmp = &types.ZERO_TOKENID
	}
	assert.Equal(t, token, tokenCmp)
}

func assertPriceEqual(t *testing.T, price string, priceBytes []byte) {
	if len(price) == 0 {
		price = "0";
	}
	var priceCmp = "0"
	if len(priceBytes) > 0 {
		priceCmp = dex.BytesToPrice(priceBytes)
	}
	assert.Equal(t, price, priceCmp)
}

func assertHashEqual(t *testing.T, hash *types.Hash, hashBytes []byte) {
	if hash == nil {
		hash = &types.ZERO_HASH
	}
	var hashCmp = &types.Hash{}
	if len(hashBytes) > 0 {
		hashCmp.SetBytes(hashBytes)
	} else {
		hashCmp = &types.ZERO_HASH
	}
	assert.Equal(t, hash, hashCmp)
}
