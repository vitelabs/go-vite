package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const (
	jsonDexFund = `
	[
		{"type":"function","name":"DexFundUserDeposit", "inputs":[]},
		{"type":"function","name":"DexFundUserWithdraw", "inputs":[{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"orderId","type":"bytes"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"uint32"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]}
	]`

	MethodNameDexFundUserDeposit  = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder     = "DexFundNewOrder"
	MethodNameDexFundSettleOrders = "DexFundSettleOrders"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
	fundKeyPrefix = []byte("fund:")
)

type ParamDexFundWithDraw struct {
	Token   types.TokenTypeId
	Amount  *big.Int
}

type ParamDexFundNewOrder struct {
	OrderId []byte
	TradeToken   types.TokenTypeId
	QuoteToken   types.TokenTypeId
	Side bool
	OrderType uint32
	Price string
	Quantity *big.Int
}

type ParamDexSerializedData struct {
	Data []byte
}

type DexFund struct {
	dexproto.Fund
}

func (df *DexFund) serialize() (data []byte, err error) {
	return proto.Marshal(&df.Fund)
}

func (df *DexFund) deSerialize(fundData []byte) (dexFund *DexFund, err error) {
	protoFund := dexproto.Fund{}
	if err := proto.Unmarshal(fundData, &protoFund); err != nil {
		return nil, err
	} else {
		return &DexFund{protoFund}, nil
	}
}

type MethodDexFundUserDeposit struct {
}

func (md *MethodDexFundUserDeposit) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserDeposit) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, dexFundDepositGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() <= 0 {
		return quotaLeft, fmt.Errorf("deposit amount is zero")
	}
	if err = getTokenInfo(db, block.TokenId); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundUserDeposit) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund = &DexFund{}
		err     error
	)
	if dexFund, err = GetFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	walletAvailable := db.GetBalance(&sendBlock.AccountAddress, &sendBlock.TokenId)
	if walletAvailable.Cmp(sendBlock.Amount) < 0 {
		return []*SendBlock{}, fmt.Errorf("deposit amount exceed token balance")
	}
	dexAccount, exists := getAccountByTokeIdFromFund(dexFund, sendBlock.TokenId)
	dexAvailable := new(big.Int).SetBytes(dexAccount.Available)
	dexAccount.Available = dexAvailable.Add(dexAvailable, sendBlock.Amount).Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, dexAccount)
	}
	return []*SendBlock{}, saveFundToStorage(db, sendBlock.AccountAddress, dexFund)
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserWithdraw) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundWithdrawGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, block.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Sign() <= 0 {
		return quotaLeft, fmt.Errorf("withdraw amount is zero")
	}
	if tokenInfo := cabi.GetTokenById(db, param.Token); tokenInfo == nil {
		return quotaLeft, fmt.Errorf("token to withdraw is invalid")
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundUserWithdraw) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(ParamDexFundWithDraw)
	var (
		dexFund = &DexFund{}
		err     error
	)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	if dexFund, err = GetFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	account, _ := getAccountByTokeIdFromFund(dexFund, param.Token)
	available := big.NewInt(0).SetBytes(account.Available)
	if available.Cmp(param.Amount) < 0 {
		return []*SendBlock{}, fmt.Errorf("withdraw amount exceed fund available")
	}
	available = available.Sub(available, param.Amount)
	account.Available = available.Bytes()
	if err = saveFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return []*SendBlock{}, err
	}
	return []*SendBlock{
		{
				block,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				param.Amount,
				param.Token,
				[]byte{},
		},
	},nil
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundNewOrderGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.Data); err != nil {
		return quotaLeft, err
	}
	if err = checkOrderParam(db, param); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundNewOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund = &DexFund{}
		tradeBlockData []byte
		err error
		orderBytes []byte
	)
	param := new(ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	order := &dexproto.Order{}
	renderOrder(order, param, db, sendBlock.AccountAddress, db.GetSnapshotBlockByHash(&block.SnapshotHash).Timestamp)
	if dexFund, err = GetFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	if _, err = tryLockFundForNewOrder(db, dexFund, order); err != nil {
		return []*SendBlock{}, err
	}
	if err = saveFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return []*SendBlock{}, err
	}
	if orderBytes, err = proto.Marshal(order); err != nil {
		return []*SendBlock{}, err
	}
	if tradeBlockData, err = ABIDexTrade.PackMethod(MethodNameDexTradeNewOrder, orderBytes); err != nil {
		return []*SendBlock{}, err
	}
	return []*SendBlock{
		{
			block,
			types.AddressDexTrade,
			ledger.BlockTypeSendCall,
			big.NewInt(0),
			ledger.ViteTokenId, // no need send token
			tradeBlockData,
		},
	},nil
}

type MethodDexFundSettleOrders struct {
}

func (md *MethodDexFundSettleOrders) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundSettleOrders) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundSettleOrdersGas); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()) {
		return quotaLeft, fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, block.Data); err != nil {
		return quotaLeft, err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return quotaLeft, err
	}
	if err = checkActions(settleActions); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md MethodDexFundSettleOrders) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()) {
		return []*SendBlock{}, fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	var err error
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return []*SendBlock{}, err
	}
	for _, action := range settleActions.Actions {
		if err = doSettleAction(db, action); err != nil {
			return []*SendBlock{}, err
		}
	}
	return []*SendBlock{}, nil
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func tryLockFundForNewOrder(db vmctxt_interface.VmDatabase, dexFund *DexFund, order *dexproto.Order) (needUpdate bool, err error) {
	return checkAndLockFundForNewOrder(db, dexFund, order, false)
}

func checkAndLockFundForNewOrder(db vmctxt_interface.VmDatabase, dexFund *DexFund, order *dexproto.Order, onlyCheck bool) (needUpdate bool, err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId *types.TokenTypeId
		lockAmountToInc *big.Int
	)
	switch order.Side {
	case false: //buy
		lockToken = order.QuoteToken
		if order.Type == dex.Limited {
			lockAmount = dex.AddBigInt(order.Amount, order.LockedBuyFee)
		}
	case true: // sell
		lockToken = order.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = fromBytesToTokenTypeId(lockToken); err != nil {
		return false, err
	}
	//var tokenName string
	//if tokenInfo := cabi.GetTokenById(db, *lockTokenId); tokenInfo != nil {
	//	tokenName = tokenInfo.TokenName
	//}
	account, exists := getAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := big.NewInt(0).SetBytes(account.Available)
	if order.Type != dex.Market || order.Side {// limited or sell order
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
		//fmt.Printf("token %s, available %s , lockAmountToInc %s\n", tokenName, available.String(), lockAmountToInc.String())
		if available.Cmp(lockAmountToInc) < 0 {
			return false, fmt.Errorf("order lock amount exceed fund available")
		}
	}

	if onlyCheck {
		return false, nil
	}
	if !order.Side && order.Type == dex.Market {// buy or market order
		if available.Sign() <= 0 {
			return false, fmt.Errorf("no quote amount available for market sell order")
		} else {
			lockAmount = available.Bytes()
			//NOTE: use amount available for order amount to full fill
			order.Amount = lockAmount
			lockAmountToInc = available
			needUpdate = true
		}
	}
	available = available.Sub(available, lockAmountToInc)
	lockedInBig := big.NewInt(0).SetBytes(account.Locked)
	lockedInBig = lockedInBig.Add(lockedInBig, lockAmountToInc)
	account.Available = available.Bytes()
	account.Locked = lockedInBig.Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return needUpdate, nil
}

func GetFundFromStorage(storage vmctxt_interface.VmDatabase, address types.Address) (dexFund *DexFund, err error) {
	fundKey := GetUserFundKey(address)
	dexFund = &DexFund{}
	if fundBytes := storage.GetStorage(&types.AddressDexFund, fundKey); len(fundBytes) > 0 {
		if dexFund, err = dexFund.deSerialize(fundBytes); err != nil {
			return nil, err
		}
	}
	return dexFund, nil
}

func saveFundToStorage(storage vmctxt_interface.VmDatabase, address types.Address, dexFund *DexFund) error {
	if fundRes, err := dexFund.serialize(); err == nil {
		storage.SetStorage(GetUserFundKey(address), fundRes)
		return nil
	} else {
		return err
	}
}

func getAccountByTokeIdFromFund(dexFund *DexFund, token types.TokenTypeId) (account *dexproto.Account, exists bool) {
	for _, a := range dexFund.Accounts {
		if bytes.Equal(token.Bytes(), a.Token) {
			return a, true
		}
	}
	account = &dexproto.Account{}
	account.Token = token.Bytes()
	account.Available = big.NewInt(0).Bytes()
	account.Locked = big.NewInt(0).Bytes()
	return account, false
}

func fromBytesToTokenTypeId(bytes []byte) (tokenId *types.TokenTypeId, err error) {
	tokenId = &types.TokenTypeId{}
	if err := tokenId.SetBytes(bytes); err == nil {
		return tokenId, nil
	} else {
		return nil, err
	}
}

func getTokenInfo(db vmctxt_interface.VmDatabase, tokenId types.TokenTypeId) (error, *types.TokenInfo) {
	if tokenInfo := cabi.GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid"), nil
	} else {
		return nil, tokenInfo
	}
}

func checkOrderParam(db vmctxt_interface.VmDatabase, orderParam *ParamDexFundNewOrder) error {
	var (
		orderId dex.OrderId
		err error
	)
	if orderId, err = dex.NewOrderId(orderParam.OrderId); err != nil {
		return err
	}
	if !orderId.IsNormal() {
		return fmt.Errorf("invalid order id")
	}
	if err, _ = getTokenInfo(db, orderParam.TradeToken); err != nil {
		return err
	}
	if err, _ = getTokenInfo(db, orderParam.QuoteToken); err != nil {
		return err
	}
	if orderParam.OrderType != dex.Market && orderParam.OrderType != dex.Limited {
		return fmt.Errorf("invalid order type")
	}
	if orderParam.OrderType == dex.Limited {
		if !dex.ValidPrice(orderParam.Price) {
			return fmt.Errorf("invalid format for price")
		}
	}
	if orderParam.Quantity.Sign() <= 0 {
		return fmt.Errorf("invalid trade quantity for order")
	}
	if _, err = strconv.ParseFloat(orderParam.Price, 64); err != nil {
		return fmt.Errorf("invalid price format")
	}
	return nil
}

func renderOrder(order *dexproto.Order, param *ParamDexFundNewOrder, db vmctxt_interface.VmDatabase, address types.Address, snapshotTM *time.Time) {
	order.Id = param.OrderId
	order.Address = address.Bytes()
	order.TradeToken = param.TradeToken.Bytes()
	order.QuoteToken = param.QuoteToken.Bytes()
	_, tradeTokenInfo := getTokenInfo(db, param.TradeToken)
	order.TradeTokenDecimals = int32(tradeTokenInfo.Decimals)
	_, quoteTokenInfo := getTokenInfo(db, param.QuoteToken)
	order.QuoteTokenDecimals = int32(quoteTokenInfo.Decimals)
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = param.Price
	order.Quantity = param.Quantity.Bytes()
	if order.Type == dex.Limited {
		order.Amount = dex.CalculateRawAmount(order.Quantity, order.Price, order.TradeTokenDecimals, order.QuoteTokenDecimals)
		if !order.Side { //buy
			order.LockedBuyFee = dex.CalculateRawFee(order.Amount, maxFeeRate())
		}
	}
	order.Status = dex.Pending
	order.Timestamp = snapshotTM.Unix()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
}

func checkActions(actions *dexproto.SettleActions) error {
	if actions == nil || len(actions.Actions) == 0 {
		return fmt.Errorf("settle orders is emtpy")
	}
	for _ , v := range actions.Actions {
		if len(v.Address) != 20 {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(v.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for settle")
		}
		if dex.CmpToBigZero(v.IncAvailable) < 0 {
			return fmt.Errorf("negative incrAvailable for settle")
		}
		if dex.CmpToBigZero(v.DeduceLocked) < 0 {
			return fmt.Errorf("negative deduceLocked for settle")
		}
		if dex.CmpToBigZero(v.ReleaseLocked) < 0 {
			return fmt.Errorf("negative releaseLocked for settle")
		}
	}
	return nil
}

func doSettleAction(db vmctxt_interface.VmDatabase, action *dexproto.SettleAction) error {
	address := &types.Address{}
	address.SetBytes([]byte(action.Address))
	if dexFund, err := GetFundFromStorage(db, *address); err != nil {
		return err
	} else {
		if tokenId, err := fromBytesToTokenTypeId(action.Token); err != nil {
			return err
		} else {
			if err, _ = getTokenInfo(db, *tokenId); err != nil {
				return err
			}
			account, exists := getAccountByTokeIdFromFund(dexFund, *tokenId)
			//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
			if dex.CmpToBigZero(action.DeduceLocked) > 0 {
				if dex.CmpForBigInt(action.DeduceLocked, account.Locked) > 0 {
					return fmt.Errorf("try deduce locked amount execeed locked")
				}
				account.Locked = dex.SubBigInt(account.Locked, action.DeduceLocked)
			}
			if dex.CmpToBigZero(action.ReleaseLocked) > 0 {
				if dex.CmpForBigInt(action.ReleaseLocked, account.Locked) > 0 {
					return fmt.Errorf("try release locked amount execeed locked")
				}
				account.Locked = dex.SubBigInt(account.Locked, action.ReleaseLocked)
				account.Available = dex.AddBigInt(account.Locked, action.ReleaseLocked)
			}
			if dex.CmpToBigZero(action.IncAvailable) > 0 {
				account.Available = dex.AddBigInt(account.Available, action.IncAvailable)
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			if err = saveFundToStorage(db, *address, dexFund); err != nil {
				return err
			}
			//fmt.Printf("settle for :address %s, tokenId %s, DeduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.DeduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
		return err
	}
	return nil
}