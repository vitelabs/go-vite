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

func (p *MethodDexFundUserDeposit) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodDexFundUserDeposit) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserDeposit) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, dexFundDepositGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundDepositAndWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserDeposit, block.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Uint64() == 0 {
		return quotaLeft, fmt.Errorf("deposit amount is zero")
	}
	if err = checkToken(db, param.Token); err != nil {
		return quotaLeft, err
	}
	block.TokenId = param.Token
	block.Amount = param.Amount
	block.ToAddress = types.AddressDexFund
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
	available := db.GetBalance(&sendBlock.AccountAddress, &sendBlock.TokenId)
	if available.Cmp(sendBlock.Amount) < 0 {
		return []*SendBlock{}, fmt.Errorf("deposit amount exceed token balance")
	}
	account, exists := getAccountByTokeIdFromFund(dexFund, sendBlock.TokenId)
	bigAmt := big.NewInt(0).SetBytes(account.Available)
	bigAmt = bigAmt.Add(bigAmt, sendBlock.Amount)
	account.Available = dex.AddBigInt(account.Available, bigAmt.Bytes())
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return []*SendBlock{}, saveFundToStorage(db, sendBlock.AccountAddress, dexFund)
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodDexFundUserWithdraw) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserWithdraw) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundWithdrawGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexFundDepositAndWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, block.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Uint64() == 0 {
		return quotaLeft, fmt.Errorf("withdraw amount is zero")
	}
	if tokenInfo := cabi.GetTokenById(db, param.Token); tokenInfo == nil {
		return quotaLeft, fmt.Errorf("token to withdraw is invalid")
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundUserWithdraw) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(ParamDexFundDepositAndWithDraw)
	ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, sendBlock.Data)
	var (
		dexFund = &DexFund{}
		err     error
	)
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
	param := new(ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.Data); err != nil {
		return quotaLeft, err
	}
	order := &dexproto.Order{}
	if err = proto.Unmarshal(param.Data, order); err != nil {
		return quotaLeft, fmt.Errorf("input data format of order is invalid")
	}
	if err = checkOrderParam(db, order); err != nil {
		return quotaLeft, err
	}
	renderOrder(order, block.AccountAddress)
	var dexFund = &DexFund{}
	if dexFund, err = GetFundFromStorage(db, block.AccountAddress); err != nil {
		return quotaLeft, err
	}
	if err = checkFundForNewOrder(dexFund, order); err != nil {
		return quotaLeft, err
	}
	param.Data, _ = proto.Marshal(order)
	block.Data, _ = ABIDexFund.PackMethod(MethodNameDexFundNewOrder, param.Data)
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundNewOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(ParamDexSerializedData)
	ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, sendBlock.Data)
	order := &dexproto.Order{}
	proto.Unmarshal(param.Data, order)
	var (
		dexFund = &DexFund{}
		needUpdate bool
		err error
	)
	if dexFund, err = GetFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	if needUpdate, err = tryLockFundForNewOrder(dexFund, order); err != nil {
		return []*SendBlock{}, err
	}
	if err = saveFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return []*SendBlock{}, err
	}
	if needUpdate {// update for market buy order amount setting
		param.Data, _ = proto.Marshal(order)
	}
	tradeBlockData, _ := ABIDexTrade.PackMethod(MethodNameDexTradeNewOrder, param.Data)
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

func checkFundForNewOrder(dexFund *DexFund, order *dexproto.Order) error {
	_, err := checkAndLockFundForNewOrder(dexFund, order, true)
	return err
}

func tryLockFundForNewOrder(dexFund *DexFund, order *dexproto.Order) (needUpdate bool, err error) {
	return checkAndLockFundForNewOrder(dexFund, order, false)
}

func checkAndLockFundForNewOrder(dexFund *DexFund, order *dexproto.Order, onlyCheck bool) (needUpdate bool, err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId *types.TokenTypeId
		lockAmountToInc *big.Int
	)
	switch order.Side {
	case false: //buy
		lockToken = order.QuoteToken
		if order.Type == dex.Limited {
			lockAmount = order.Amount
		}
	case true: // sell
		lockToken = order.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = fromBytesToTokenTypeId(lockToken); err != nil {
		return false, err
	}
	account, exists := getAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := big.NewInt(0).SetBytes(account.Available)
	if order.Type != dex.Market || order.Side {// limited or sell order
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
		if available.Cmp(lockAmountToInc) < 0 {
			return false, fmt.Errorf("order lock amount exceed fund available")
		}
	}

	if onlyCheck {
		return false, nil
	}
	if !order.Side && order.Type == dex.Market {
		if available.Cmp(big.NewInt(0)) == 0 {
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

func checkTokenByProto(db vmctxt_interface.VmDatabase, protoBytes []byte) error {
	if tokenId, err := fromBytesToTokenTypeId(protoBytes); err != nil {
		return err
	} else {
		return checkToken(db, *tokenId)
	}
}

func checkToken(db vmctxt_interface.VmDatabase, tokenId types.TokenTypeId) error {
	if tokenInfo := cabi.GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid")
	} else {
		return nil
	}
}

func checkOrderParam(db vmctxt_interface.VmDatabase, order *dexproto.Order) error {
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
	if order.Type == dex.Limited {
		if !dex.ValidPrice(order.Price) {
			return fmt.Errorf("invalid format for price")
		}
	}
	if dex.CmpToBigZero(order.Quantity) <= 0 {
		return fmt.Errorf("invalid trade quantity for order")
	}
	return nil
}

func renderOrder(order *dexproto.Order, address types.Address) {
	order.Address = address.Bytes()
	if order.Type == dex.Limited {
		order.Amount = dex.CalculateAmount(order.Quantity, order.Price)
	}
	order.Status = dex.Pending
	order.Timestamp = time.Now().UnixNano() / 1000
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
			if err = checkToken(db, *tokenId); err != nil {
				return err
			}
			account, exists := getAccountByTokeIdFromFund(dexFund, *tokenId)
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
		}
		return err
	}
	return nil
}