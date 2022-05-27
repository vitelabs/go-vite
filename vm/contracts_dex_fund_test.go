package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/v2/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/v2/vm/util"
)

type DexFundCase struct {
	Name           string
	GlobalEnv      GlobalEnv
	DexFundStorage *DexFundStorage
	// environment
	PreStorage    map[string]string
	PreBalanceMap map[types.TokenTypeId]string
	AssetActions  []*AssetAction

	InitContract   *InitContract
	SetQuoteTokens []*SetQuoteToken

	NewMarkets      []*NewMarket
	NewMarketTokens []*NewMarketToken
	PlaceOrders     []*PlaceOrder

	CheckBalances []*CheckBalance
	CheckMarkets  []*CheckMarket
	CheckEvents   []*CheckEvent
}

type GlobalEnv struct {
	// global status
	SbHeight uint64
	SbTime   int64
	SbHash   string
	Quotas   []*StakeQuota
	Balances []*BalanceStorage
	//CsDetail map[uint64]map[string]*ConsensusDetail
}

type StakeQuota struct {
	Address types.Address
	Quota   *big.Int
}

type DexFundStorage struct {
	Funds     []*FundStorage
	Tokens    []*TokenStorage
	Markets   []*MarketStorage
	Timestamp int64
}

type FundStorage struct {
	Address  types.Address
	Accounts []AccountStorage
}

type AccountStorage struct {
	Token           types.TokenTypeId
	Available       *big.Int
	Locked          *big.Int
	VxUnlocking     *big.Int
	CancellingStake *big.Int
}

type BalanceStorage struct {
	Address  types.Address
	Balances []*Balance
}

type Balance struct {
	TokenId types.TokenTypeId
	Amount  *big.Int
}

type TokenStorage struct {
	TokenId        types.TokenTypeId
	Decimals       int32
	Symbol         string
	Index          int32
	Owner          types.Address
	QuoteTokenType int32
}

type MarketStorage struct {
	MarketId             int32
	TradeToken           types.TokenTypeId
	QuoteToken           types.TokenTypeId
	Symbol               string
	QuoteTokenType       int32
	TradeTokenDecimals   int32
	QuoteTokenDecimals   int32
	TakerOperatorFeeRate int32
	MakerOperatorFeeRate int32
	AllowMining          bool
	Valid                bool
	Owner                types.Address
	Creator              types.Address
	Stopped              bool
	Timestamp            int64
	StableMarket         bool
}

type AssetAction struct {
	ActionTye int32 // 0 deposit, 1 withdraw, 2 transfer
	Address   types.Address
	Target    types.Address
	Token     types.TokenTypeId
	Amount    *big.Int
}

type CheckBalance struct {
	Address  types.Address
	Balances []AccountStorage
}

type InitContract struct {
	InitOwner       types.Address
	Owner           types.Address
	Timer           types.Address
	Trigger         types.Address
	MineProxy       types.Address
	MaintainerProxy types.Address
}

type SetQuoteToken struct {
	TokenId     types.TokenTypeId
	Bid         uint8
	Exist       bool
	Decimals    uint8
	TokenSymbol string
	Index       uint16
	DexOwner    types.Address
	Owner       types.Address
}

type NewTokenEvent struct {
	TokenId types.TokenTypeId
}

type NewMarket struct {
	Owner      types.Address
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
}

type NewMarketToken struct {
	TokenId     types.TokenTypeId
	Bid         uint8
	Exist       bool
	Decimals    uint8
	TokenSymbol string
	Index       uint16
	Owner       types.Address
}

type PlaceOrder struct {
	Address          types.Address
	TradeToken       types.TokenTypeId
	QuoteToken       types.TokenTypeId
	Side             bool
	OrderType        uint8
	Price            string
	Quantity         *big.Int
	CheckPlacedOrder *PlacedOrder
}

type PlacedOrder struct {
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
	MarketOrderAmtThreshold *big.Int
}

type TransferAssetEvent struct {
	BizType int32
	From    types.Address
	To      types.Address
	Token   types.TokenTypeId
	Amount  *big.Int
	Extra   string
}

type CheckMarket struct {
	MarketId           int32
	MarketSymbol       string
	TradeToken         types.TokenTypeId
	QuoteToken         types.TokenTypeId
	QuoteTokenType     int32
	TradeTokenDecimals int32
	QuoteTokenDecimals int32
	Owner              types.Address
	Creator            types.Address
}

type CheckEvent struct {
	TopicName string
	Transfer  *TransferAssetEvent
	NewToken  *NewTokenEvent
	NewMarket *NewMarket
}

func TestDexFund(t *testing.T) {
	upgrade.CleanupUpgradeBox()
	upgrade.InitUpgradeBox(upgrade.NewLatestUpgradeBox())

	testDir := "./contracts/dex/test/fund/"
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
		testCaseMap := new(map[string]*DexFundCase)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file %v failed, %v", testFile.Name(), ok)
		}
		for k, testCase := range *testCaseMap {
			fmt.Println(testFile.Name() + ":" + k)
			db := initFundDb(testCase, t)
			reader := util.NewVMConsensusReader(newConsensusReaderTest(db.GetGenesisSnapshotBlock().Timestamp.Unix(), 24*3600, nil))
			vm := NewVM(reader)
			executeActions(testCase, vm, db, t)
			executeChecks(testCase, db, t)
		}
	}
}

func executeActions(testCase *DexFundCase, vm *VM, db *testDatabase, t *testing.T) {
	if testCase.AssetActions != nil {
		for _, action := range testCase.AssetActions {
			sendBlock := newSendBlock(action.Address, types.AddressDexFund)
			db.addr = action.Address

			var (
				vmSendBlock *interfaces.VmAccountBlock
				err         error
			)
			if action.ActionTye == dex.TransferAssetDeposit { // deposit
				sendBlock.TokenId = action.Token
				sendBlock.Amount = action.Amount
				sendBlock.Data, err = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundDeposit)
				helper.ErrFailf(t, err, "ABI pack err %v", err)
				vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)
			} else if action.ActionTye == dex.TransferAssetWithdraw {
				sendBlock.Data, err = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundWithdraw, action.Token, action.Amount)
				helper.ErrFailf(t, err, "ABI pack err %v", err)
				vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)
			} else { // transfer\
				sendBlock.Data, err = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundTransfer, action.Target, action.Token, action.Amount)
				helper.ErrFailf(t, err, "ABI pack err %v", err)
				vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)
			}
			helper.ErrFailf(t, err, "vm.RunV2 handle send result err %v", err)
			db.addr = types.AddressDexFund
			_, _, err = vm.RunV2(db, newRecBlock(types.AddressDexFund), vmSendBlock.AccountBlock, nil)
			//fmt.Printf("handle receive runVm err %v\n", err)
			helper.ErrFailf(t, err, "vm.RunV2 handle receive result err %v", err)
		}
	}
	if testCase.InitContract != nil {
		ic := *testCase.InitContract
		data, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundOwnerConfig, uint8(dex.AdminConfigOwner+dex.AdminConfigTimeOracle+dex.AdminConfigPeriodJobTrigger+dex.AdminConfigMakerMiningAdmin+dex.AdminConfigMaintainer), ic.Owner, ic.Timer, ic.Trigger, false, ic.MineProxy, ic.MaintainerProxy)
		doAction("initContract", db, vm, ic.InitOwner, types.AddressDexFund, data, t)
	}
	if testCase.SetQuoteTokens != nil {
		for _, st := range testCase.SetQuoteTokens {
			data, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundTradeAdminConfig, uint8(dex.TradeAdminConfigNewQuoteToken), ledger.ViteTokenId, ledger.ViteTokenId, false, st.TokenId, uint8(dex.ViteTokenType), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
			doAction("setQuoteToken", db, vm, st.DexOwner, types.AddressDexFund, data, t)
		}
		for _, st := range testCase.SetQuoteTokens {
			data, _ := cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundGetTokenInfoCallback, st.TokenId, st.Bid, st.Exist, st.Decimals, st.TokenSymbol, st.Index, st.Owner)
			doAction("setQuoteTokenCallback", db, vm, types.AddressAsset, types.AddressDexFund, data, t)
		}
	}
	if testCase.NewMarkets != nil {
		for _, nm := range testCase.NewMarkets {
			data, _ := cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundOpenNewMarket, nm.TradeToken, nm.QuoteToken)
			doAction("newMarket", db, vm, nm.Owner, types.AddressDexFund, data, t)
		}
	}
	if testCase.NewMarketTokens != nil {
		for _, nmt := range testCase.NewMarketTokens {
			data, _ := cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundGetTokenInfoCallback, nmt.TokenId, nmt.Bid, nmt.Exist, nmt.Decimals, nmt.TokenSymbol, nmt.Index, nmt.Owner)
			doAction("newMarketToken", db, vm, types.AddressAsset, types.AddressDexFund, data, t)
		}
	}
	if testCase.PlaceOrders != nil {
		for _, od := range testCase.PlaceOrders {
			data, _ := cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundPlaceOrder, od.TradeToken, od.QuoteToken, od.Side, od.OrderType, od.Price, od.Quantity)
			sendBlocks := doAction("placeOrder", db, vm, od.Address, types.AddressDexFund, data, t)
			assert.Equal(t, 1, len(sendBlocks))
			if od.CheckPlacedOrder != nil {
				param := new(dex.ParamSerializedData)
				err := cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradePlaceOrder, sendBlocks[0].Data)
				assert.Nil(t, err)
				order := &dex.Order{}
				err = order.DeSerialize(param.Data)
				assert.Nil(t, err)
				if od.CheckPlacedOrder != nil {
					checkPlaceOrder(t, od.CheckPlacedOrder, order)
				}
			}
		}
	}
}

func checkPlaceOrder(t *testing.T, po *PlacedOrder, od *dex.Order) {
	assertAddressEqual(t, po.Address, od.Address, "checkPlaceOrder.Address")
	marketId, side, price, _, _ := dex.DeComposeOrderId(od.Id)
	assert.Equal(t, po.MarketId, marketId)
	assert.Equal(t, po.Side, side)
	assert.Equal(t, po.Type, od.Type)
	assert.Equal(t, po.Price, dex.BytesToPrice(price))
	assert.Equal(t, po.TakerFeeRate, od.TakerFeeRate)
	assert.Equal(t, po.MakerFeeRate, od.MakerFeeRate)
	assert.Equal(t, po.TakerOperatorFeeRate, od.TakerOperatorFeeRate)
	assert.Equal(t, po.MakerOperatorFeeRate, od.MakerOperatorFeeRate)
	assertAmountEqual(t, po.Quantity, od.Quantity, "checkPlaceOrder.Quantity")
	assertAmountEqual(t, po.Amount, od.Amount, "checkPlaceOrder.Amount")
	assertAmountEqual(t, po.LockedBuyFee, od.LockedBuyFee, "checkPlaceOrder.LockedBuyFee")
	assertAmountEqual(t, po.MarketOrderAmtThreshold, od.MarketOrderAmtThreshold, "checkPlaceOrder.MarketOrderAmtThreshold")
	assert.Equal(t, po.Status, od.Status)
}

func doAction(name string, db *testDatabase, vm *VM, from, to types.Address, data []byte, t *testing.T) []*ledger.AccountBlock {
	sendBlock := newSendBlock(from, to)
	db.addr = from
	var (
		vmSendBlock, sendCreateBlock *interfaces.VmAccountBlock
		err                          error
	)
	sendBlock.Data = data
	vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)

	//fmt.Printf("executeActions send runVm err %v\n", err)
	assert.True(t, err == nil, name+" vm.RunV2 handle send result err not nil")
	db.addr = to
	sendCreateBlock, _, err = vm.RunV2(db, newRecBlock(to), vmSendBlock.AccountBlock, nil)

	//fmt.Printf("handle receive runVm err %v\n", err)
	assert.True(t, err == nil, name+" vm.RunV2 handle receive result err not nil")
	return sendCreateBlock.AccountBlock.SendBlockList
}

func newSendBlock(from, to types.Address) *ledger.AccountBlock {
	sendBlock := &ledger.AccountBlock{}
	sendBlock.AccountAddress = from
	sendBlock.BlockType = ledger.BlockTypeSendCall
	sendBlock.ToAddress = to
	sendBlock.TokenId = ledger.ViteTokenId
	sendBlock.Amount = big.NewInt(0)
	return sendBlock
}

func newRecBlock(to types.Address) *ledger.AccountBlock {
	rcBlock := &ledger.AccountBlock{}
	rcBlock.AccountAddress = to
	rcBlock.BlockType = ledger.BlockTypeReceive
	return rcBlock
}

func executeChecks(testCase *DexFundCase, db *testDatabase, t *testing.T) {
	db.addr = types.AddressDexFund
	if testCase.CheckBalances != nil {
		for idx, bc := range testCase.CheckBalances {
			fund, ok := dex.GetFund(db, bc.Address)
			assert.True(t, ok, fmt.Sprintf("fund not exist for %s, %s, %d", testCase.Name, bc.Address.String(), idx))
			for idx1, bl := range bc.Balances {
				acc, ok1 := dex.GetAccountByToken(fund, bl.Token)
				assert.True(t, ok1, fmt.Sprintf("account not exist for %s, %s, %s, %d", testCase.Name, bc.Address.String(), bl.Token.String(), idx1))
				assert.Equal(t, bl.Available.String(), new(big.Int).SetBytes(acc.Available).String(), fmt.Sprintf("account available equals for %s, %s, %s, %d", testCase.Name, bc.Address.String(), bl.Token.String(), idx1))
				assert.Equal(t, bl.Locked.String(), new(big.Int).SetBytes(acc.Locked).String(), fmt.Sprintf("account locked equals for %s, %s, %s, %d", testCase.Name, bc.Address.String(), bl.Token.String(), idx1))
			}
		}
	}
	if testCase.CheckMarkets != nil {
		for _, cm := range testCase.CheckMarkets {
			//mk, ok := getMarketInfo(db, cm.TradeToken, cm.QuoteToken)
			mk, ok := dex.GetMarketInfo(db, cm.TradeToken, cm.QuoteToken)
			assert.True(t, ok, fmt.Sprintf("market not exist for %s, %s, %s", testCase.Name, cm.TradeToken.String(), cm.QuoteToken.String()))
			assert.Equal(t, mk.MarketId, cm.MarketId, "quote token not equal")
			assert.Equal(t, mk.MarketSymbol, cm.MarketSymbol, "symbol not equal")
			assert.True(t, bytes.Equal(mk.TradeToken, cm.TradeToken.Bytes()), "trade token not equal")
			assert.True(t, bytes.Equal(mk.QuoteToken, cm.QuoteToken.Bytes()), "quote token not equal")
			assert.Equal(t, mk.TradeTokenDecimals, cm.TradeTokenDecimals, "trade token decimals not equal")
			assert.Equal(t, mk.QuoteTokenDecimals, cm.QuoteTokenDecimals, "quote token decimals not equal")
			assert.True(t, bytes.Equal(mk.Owner, cm.Owner.Bytes()), "owner not equal")
			assert.True(t, bytes.Equal(mk.Creator, cm.Creator.Bytes()), "creator not equal")
			assert.Equal(t, mk.QuoteTokenType, cm.QuoteTokenType, "QuoteTokenType not equal")
		}
	}
	if testCase.CheckEvents != nil {
		assert.Equal(t, len(testCase.CheckEvents), len(db.logList))
		for i, ev := range testCase.CheckEvents {
			log := db.logList[i]
			assert.Equal(t, getTopicId(ev.TopicName), log.Topics[0])
			if ev.Transfer != nil {
				ae := &dex.TransferAssetEvent{}
				ae.FromBytes(log.Data)
				assert.Equal(t, ev.Transfer.BizType, ae.BizType)
				assert.True(t, bytes.Equal(ev.Transfer.From.Bytes(), ae.From))
				assert.True(t, bytes.Equal(ev.Transfer.To.Bytes(), ae.To))
				assert.True(t, bytes.Equal(ev.Transfer.Token.Bytes(), ae.Token))
				assert.True(t, bytes.Equal(ev.Transfer.Amount.Bytes(), ae.Amount))
				assert.True(t, len(ev.Transfer.Extra) == 0 && len(ae.Extra) == 0 || ev.Transfer.Extra == string(ae.Extra))
			}
			if ev.NewToken != nil {
				te := &dex.TokenEvent{}
				te.FromBytes(log.Data)
				assert.True(t, bytes.Equal(ev.NewToken.TokenId.Bytes(), te.TokenId))
			}
			if ev.NewMarket != nil {
				me := &dex.MarketEvent{}
				me.FromBytes(log.Data)
				assert.True(t, bytes.Equal(ev.NewMarket.Owner.Bytes(), me.Owner))
				assert.True(t, bytes.Equal(ev.NewMarket.TradeToken.Bytes(), me.TradeToken))
				assert.True(t, bytes.Equal(ev.NewMarket.QuoteToken.Bytes(), me.QuoteToken))
			}
		}
	}
}

func generateDb(caseName string, globalEnv *GlobalEnv, t *testing.T) *testDatabase {
	var currentTime time.Time
	if globalEnv.SbTime > 0 {
		currentTime = time.Unix(globalEnv.SbTime, 0)
	} else {
		currentTime = time.Now()
	}
	latestSnapshotBlock := &ledger.SnapshotBlock{
		Height:    globalEnv.SbHeight,
		Timestamp: &currentTime,
	}
	if len(globalEnv.SbHash) > 0 {
		sbHash, parseErr := types.HexToHash(globalEnv.SbHash)
		if parseErr != nil {
			t.Fatal("invalid test case sbHash", "sbHash", globalEnv.SbHash)
		}
		latestSnapshotBlock.Hash = sbHash
	}
	var db *testDatabase
	var newDbErr error
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, _, _, _, _, _ = prepareDb(viteTotalSupply)
	t2 := time.Unix(1600663514, 0)
	snapshot20 := &ledger.SnapshotBlock{Height: 2000, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot20)
	if newDbErr != nil {
		t.Fatal("new mock db failed", "name", caseName)
	}

	if len(globalEnv.Quotas) > 0 {
		db.storageMap[types.AddressQuota] = make(map[string][]byte, 0)
		for _, qt := range globalEnv.Quotas {
			data, packErr := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
			assert.True(t, packErr == nil)
			key := ToKey(abi.GetStakeBeneficialKey(qt.Address))
			db.storageMap[types.AddressQuota][key] = data
		}
	}

	if len(globalEnv.Balances) > 0 {
		for _, balance := range globalEnv.Balances {
			db.balanceMap[balance.Address] = make(map[types.TokenTypeId]*big.Int)
			for _, bl := range balance.Balances {
				db.balanceMap[balance.Address][bl.TokenId] = bl.Amount
			}
		}
	}
	return db
}

func initFundDb(dexFundCase *DexFundCase, t *testing.T) *testDatabase {
	db := generateDb(dexFundCase.Name, &dexFundCase.GlobalEnv, t)
	if dexFundCase.DexFundStorage != nil {
		db.storageMap[types.AddressDexFund] = make(map[string][]byte, 0)
		db.addr = types.AddressDexFund
		if dexFundCase.DexFundStorage.Funds != nil {
			for _, fd := range dexFundCase.DexFundStorage.Funds {
				fund := &dex.Fund{}
				fund.Address = fd.Address.Bytes()
				for _, acc := range fd.Accounts {
					account := &dexproto.Account{}
					account.Token = acc.Token.Bytes()
					account.Available = acc.Available.Bytes()
					account.Locked = acc.Locked.Bytes()
					account.VxUnlocking = acc.VxUnlocking.Bytes()
					account.CancellingStake = acc.CancellingStake.Bytes()
					fund.Accounts = append(fund.Accounts, account)
				}
				dex.SaveFund(db, fd.Address, fund)
			}
		}
		if dexFundCase.DexFundStorage.Tokens != nil {
			for _, tk := range dexFundCase.DexFundStorage.Tokens {
				tokenInfo := &dex.TokenInfo{}
				tokenInfo.TokenId = tk.TokenId.Bytes()
				tokenInfo.Decimals = tk.Decimals
				tokenInfo.Symbol = tk.Symbol
				tokenInfo.Index = tk.Index
				tokenInfo.Owner = tk.Owner.Bytes()
				tokenInfo.QuoteTokenType = tk.QuoteTokenType
				dex.SaveTokenInfo(db, tk.TokenId, tokenInfo)
			}
		}
		if dexFundCase.DexFundStorage.Markets != nil {
			for _, mk := range dexFundCase.DexFundStorage.Markets {
				mkInfo := toDexMarketStorage(mk)
				dex.SaveMarketInfo(db, mkInfo, mk.TradeToken, mk.QuoteToken)
				//serializeToDb(db, dex.GetMarketInfoKey(mk.TradeToken, mk.QuoteToken), mkInfo)
			}
		}
		if dexFundCase.DexFundStorage.Timestamp > 0 {
			saveToStorage(db, []byte("tts:"), dex.Uint64ToBytes(uint64(dexFundCase.DexFundStorage.Timestamp)))
		}
	}
	return db
}

func toDexMarketStorage(mk *MarketStorage) *dex.MarketInfo {
	mkInfo := &dex.MarketInfo{}
	mkInfo.MarketId = mk.MarketId
	mkInfo.MarketSymbol = mk.Symbol
	mkInfo.TradeToken = mk.TradeToken.Bytes()
	mkInfo.QuoteToken = mk.QuoteToken.Bytes()
	mkInfo.QuoteTokenType = mk.QuoteTokenType
	mkInfo.TradeTokenDecimals = mk.TradeTokenDecimals
	mkInfo.QuoteTokenDecimals = mk.QuoteTokenDecimals
	mkInfo.QuoteTokenType = mk.QuoteTokenType
	mkInfo.TakerOperatorFeeRate = mk.TakerOperatorFeeRate
	mkInfo.MakerOperatorFeeRate = mk.MakerOperatorFeeRate
	mkInfo.AllowMining = mk.AllowMining
	mkInfo.Valid = mk.Valid
	mkInfo.Owner = mk.Owner.Bytes()
	mkInfo.Creator = mk.Creator.Bytes()
	mkInfo.Stopped = mk.Stopped
	mkInfo.Timestamp = mk.Timestamp
	mkInfo.StableMarket = mk.StableMarket
	return mkInfo
}

func saveToStorage(db *testDatabase, key []byte, value []byte) {
	db.storageMap[types.AddressDexFund][ToKey(key)] = value
}

func getTopicId(name string) types.Hash {
	hs := types.Hash{}
	hs.SetBytes(common.RightPadBytes([]byte(name), types.HashSize))
	return hs
}
