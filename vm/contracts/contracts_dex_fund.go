package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math"
	"math/big"
	"strings"
	"time"
)

const (
	jsonDexFund = `
	[
		{"type":"function","name":"DexFundUserDeposit", "inputs":[{"name":"address","type":"address"},{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundUserWithdraw", "inputs":[{"name":"address","type":"address"},{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"data","type":"bytes"}]},
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

type ParamDexFundDepositAndWithDraw struct {
	Address types.Address
	Token   types.TokenTypeId
	Amount  *big.Int
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

func (md *MethodDexFundUserDeposit) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, dexFundDepositGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundDepositAndWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserDeposit, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Uint64() == 0 {
		return quotaLeft, fmt.Errorf("deposit amount is zero")
	}
	if err = checkToken(block.VmContext, param.Token); err != nil {
		return quotaLeft, err
	}
	block.AccountBlock.TokenId = param.Token
	block.AccountBlock.Amount = param.Amount
	block.AccountBlock.ToAddress = AddressDexFund
	return util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
}

func (md *MethodDexFundUserDeposit) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	var (
		dexFund = &DexFund{}
		err     error
	)
	if dexFund, err = GetFundFromStorage(block.VmContext, sendBlock.AccountAddress); err != nil {
		return err
	}
	available := block.VmContext.GetBalance(&sendBlock.AccountAddress, &sendBlock.TokenId)
	if available.Cmp(sendBlock.Amount) < 0 {
		return fmt.Errorf("deposit amount exceed token balance")
	}
	account, exists := getAccountByTokeIdFromFund(dexFund, sendBlock.TokenId)
	bigAmt := big.NewInt(0).SetUint64(account.Available)
	bigAmt = bigAmt.Add(bigAmt, sendBlock.Amount)
	account.Available += bigAmt.Uint64()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return saveFundToStorage(block.VmContext, sendBlock.AccountAddress, dexFund)
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundWithdrawGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundDepositAndWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Uint64() == 0 {
		return quotaLeft, fmt.Errorf("withdraw amount is zero")
	}
	if tokenInfo := GetTokenById(block.VmContext, param.Token); tokenInfo == nil {
		return quotaLeft, fmt.Errorf("token to withdraw is invalid")
	}
	return util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
}

func (md *MethodDexFundUserWithdraw) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamDexFundDepositAndWithDraw)
	ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, sendBlock.Data)
	var (
		dexFund = &DexFund{}
		err     error
	)
	if dexFund, err = GetFundFromStorage(block.VmContext, sendBlock.AccountAddress); err != nil {
		return err
	}
	account, _ := getAccountByTokeIdFromFund(dexFund, param.Token)
	available := big.NewInt(0).SetUint64(account.Available)
	if available.Cmp(param.Amount) < 0 {
		return fmt.Errorf("withdraw amount exceed fund available")
	}
	available = available.Sub(available, param.Amount)
	account.Available = available.Uint64()
	if err = saveFundToStorage(block.VmContext, sendBlock.AccountAddress, dexFund); err != nil {
		return err
	}
	context.AppendBlock(
		&vm_context.VmAccountBlock{
			util.MakeSendBlock(
				block.AccountBlock,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				param.Amount,
				param.Token,
				context.GetNewBlockHeight(block),
				[]byte{}),
			nil})
	return nil
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundNewOrderGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	order := &dexproto.Order{}
	if err = proto.Unmarshal(param.Data, order); err != nil {
		return quotaLeft, fmt.Errorf("input data format of order is invalid")
	}
	if err = checkOrderParam(block.VmContext, order); err != nil {
		return quotaLeft, err
	}
	renderOrder(order, block.AccountBlock.AccountAddress)
	var dexFund = &DexFund{}
	if dexFund, err = GetFundFromStorage(block.VmContext, block.AccountBlock.AccountAddress); err != nil {
		return quotaLeft, err
	}
	if err = checkFundForNewOrder(dexFund, order); err != nil {
		return quotaLeft, err
	}
	param.Data, _ = proto.Marshal(order)
	block.AccountBlock.Data, _ = ABIDexFund.PackMethod(MethodNameDexFundNewOrder, param.Data)
	return util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
}

func (md *MethodDexFundNewOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (err error) {
	param := new(ParamDexSerializedData)
	ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, sendBlock.Data)
	order := &dexproto.Order{}
	proto.Unmarshal(param.Data, order)
	var (
		dexFund = &DexFund{}
	)
	if dexFund, err = GetFundFromStorage(block.VmContext, sendBlock.AccountAddress); err != nil {
		return err
	}
	if err = tryLockFundForNewOrder(dexFund, order); err != nil {
		return err
	}
	if err = saveFundToStorage(block.VmContext, sendBlock.AccountAddress, dexFund); err != nil {
		return err
	}
	tradeBlockData, _ := ABIDexTrade.PackMethod(MethodNameDexTradeNewOrder, param.Data)
	context.AppendBlock(
		&vm_context.VmAccountBlock{
			util.MakeSendBlock(
				block.AccountBlock,
				AddressDexTrade,
				ledger.BlockTypeSendCall,
				big.NewInt(0),
				ledger.ViteTokenId, // no need send token
				context.GetNewBlockHeight(block),
				tradeBlockData),
			nil})
	return nil
}

type MethodDexFundSettleOrders struct {
}

func (md *MethodDexFundSettleOrders) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundSettleOrdersGas); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), AddressDexTrade.Bytes()) {
		return quotaLeft, fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return quotaLeft, err
	}
	if err = checkActions(settleActions); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
}

func (md MethodDexFundSettleOrders) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), AddressDexTrade.Bytes()) {
		return fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	var err error
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, sendBlock.Data); err != nil {
		return err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return err
	}
	for _, action := range settleActions.Actions {
		if err = doSettleAction(block.VmContext, action); err != nil {
			return err
		}
	}
	return nil
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func checkFundForNewOrder(dexFund *DexFund, order *dexproto.Order) error {
	return checkAndLockFundForNewOrder(dexFund, order, true)
}
func tryLockFundForNewOrder(dexFund *DexFund, order *dexproto.Order) error {
	return checkAndLockFundForNewOrder(dexFund, order, false)
}

func checkAndLockFundForNewOrder(dexFund *DexFund, order *dexproto.Order, onlyCheck bool) error {
	var (
		lockToken   []byte
		lockAmount  uint64
		lockTokenId *types.TokenTypeId
		err         error
	)
	switch order.Side {
	case false: //buy
		lockToken = order.QuoteToken
		lockAmount = order.Amount
	case true: // sell
		lockToken = order.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = fromBytesToTokenTypeId(lockToken); err != nil {
		return err
	}
	account, exists := getAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := big.NewInt(0).SetUint64(account.Available)
	lockAmountToInc := big.NewInt(0).SetUint64(lockAmount)
	if available.Cmp(lockAmountToInc) < 0 {
		return fmt.Errorf("order lock amount exceed fund available")
	}
	if onlyCheck {
		return nil
	}
	available = available.Sub(available, lockAmountToInc)
	lockedInBig := big.NewInt(0).SetUint64(account.Locked)
	lockedInBig = lockedInBig.Add(lockedInBig, lockAmountToInc)
	account.Available = available.Uint64()
	account.Locked = lockedInBig.Uint64()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return nil
}

func GetFundFromStorage(storage vmctxt_interface.VmDatabase, address types.Address) (dexFund *DexFund, err error) {
	fundKey := GetUserFundKey(address)
	dexFund = &DexFund{}
	if fundBytes := storage.GetStorage(&AddressDexFund, fundKey); len(fundBytes) > 0 {
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
	account.Available = 0
	account.Locked = 0
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

func checkTokenByProto(db StorageDatabase, protoBytes []byte) error {
	if tokenId, err := fromBytesToTokenTypeId(protoBytes); err != nil {
		return err
	} else {
		return checkToken(db, *tokenId)
	}
}

func checkToken(db StorageDatabase, tokenId types.TokenTypeId) error {
	if tokenInfo := GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid")
	} else {
		return nil
	}
}

func checkOrderParam(db StorageDatabase, order *dexproto.Order) error {
	var err error
	if order.Id <= 0 {
		return fmt.Errorf("invalid order id")
	}
	if err = checkTokenByProto(db, order.TradeToken); err != nil {
		return err
	}
	if err = checkTokenByProto(db, order.QuoteToken); err != nil {
		return err
	}
	if order.Type != dex.Market && order.Type != dex.Limited {
		return fmt.Errorf("invalid order type")
	}
	if order.Price < 0 || math.Abs(order.Price) < dex.MinPricePermit {
		return fmt.Errorf("invalid order price")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("invalid trade quantity for order")
	}
	return nil
}

func renderOrder(order *dexproto.Order, address types.Address) {
	order.Address = string(address.Bytes())
	order.Amount = dex.CalculateAmount(order.Quantity, order.Price)
	order.Status = dex.Pending
	order.Timestamp = time.Now().UnixNano() / 1000
	order.ExecutedQuantity = 0
	order.ExecutedAmount = 0
	order.RefundToken = []byte{}
	order.RefundQuantity = 0
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
		if v.IncAvailable < 0 {
			return fmt.Errorf("negative incrAvailable for settle")
		}
		if v.DeduceLocked < 0 {
			return fmt.Errorf("negative deduceLocked for settle")
		}
		if v.ReleaseLocked < 0 {
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
			if err = checkToken(db, *tokenId); err != nil {
				return err
			}
			account, exists := getAccountByTokeIdFromFund(dexFund, *tokenId)
			if action.DeduceLocked > 0 {
				if action.DeduceLocked > account.Locked {
					return fmt.Errorf("try deduce locked amount execeed locked")
				}
				account.Locked -= action.DeduceLocked
			}
			if action.ReleaseLocked > 0 {
				if action.ReleaseLocked > account.Locked {
					return fmt.Errorf("try release locked amount execeed locked")
				}
				account.Locked -= action.ReleaseLocked
			}
			if action.IncAvailable > 0 {
				account.Available += action.IncAvailable
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			if err = saveFundToStorage(db, *address, dexFund); err != nil {
				return err
			}
		}
		return err
	}
	return nil
}