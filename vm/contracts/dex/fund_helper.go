package dex

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var (
	fundKeyPrefix = []byte("fd:") // fund:types.Address

	feeAccKeyPrefix          = []byte("fee:") // fee:periodId
	lastFeePeriodKey         = []byte("lFPId:")

	VxFundKeyPrefix     = []byte("vxF:")    // vxFund:types.Address
	vxSumFundsKey       = []byte("vxFS:") // vxFundSum:periodId
	lastDividendIdKey   = []byte("divId:")
	VxTokenBytes        = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	VxDividendThreshold = new(big.Int).Mul(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil), big.NewInt(10)) // 18 : vx decimals, 10 amount
)

type ParamDexFundWithDraw struct {
	Token  types.TokenTypeId
	Amount *big.Int
}

type ParamDexFundNewOrder struct {
	OrderId    []byte
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
	Side       bool
	OrderType  uint32
	Price      string
	Quantity   *big.Int
}

type ParamDexFundDividend struct {
	PeriodId uint64
}

type ParamDexSerializedData struct {
	Data []byte
}

type UserFund struct {
	dexproto.Fund
}

func (df *UserFund) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.Fund)
}

func (df *UserFund) DeSerialize(fundData []byte) (dexFund *UserFund, err error) {
	protoFund := dexproto.Fund{}
	if err := proto.Unmarshal(fundData, &protoFund); err != nil {
		return nil, err
	} else {
		return &UserFund{protoFund}, nil
	}
}

type Fee struct {
	dexproto.FeeByPeriod
}

func (df *Fee) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.FeeByPeriod)
}

func (df *Fee) DeSerialize(feeData []byte) (dexFee *Fee, err error) {
	protoFee := dexproto.FeeByPeriod{}
	if err := proto.Unmarshal(feeData, &protoFee); err != nil {
		return nil, err
	} else {
		return &Fee{protoFee}, nil
	}
}

type VxFunds struct {
	dexproto.VxFunds
}

func (dvf *VxFunds) Serialize() (data []byte, err error) {
	return proto.Marshal(&dvf.VxFunds)
}

func (dvf *VxFunds) DeSerialize(vxFundsData []byte) (*VxFunds, error) {
	protoVxFunds := dexproto.VxFunds{}
	if err := proto.Unmarshal(vxFundsData, &protoVxFunds); err != nil {
		return nil, err
	} else {
		return &VxFunds{protoVxFunds}, nil
	}
}

func CheckOrderParam(db vmctxt_interface.VmDatabase, orderParam *ParamDexFundNewOrder) error {
	var (
		orderId OrderId
		err     error
	)
	if orderId, err = NewOrderId(orderParam.OrderId); err != nil {
		return err
	}
	if !orderId.IsNormal() {
		return fmt.Errorf("invalid order id")
	}
	if err, _ = GetTokenInfo(db, orderParam.TradeToken); err != nil {
		return err
	}
	if err, _ = GetTokenInfo(db, orderParam.QuoteToken); err != nil {
		return err
	}
	if orderParam.OrderType != Market && orderParam.OrderType != Limited {
		return fmt.Errorf("invalid order type")
	}
	if orderParam.OrderType == Limited {
		if !ValidPrice(orderParam.Price) {
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

func RenderOrder(order *dexproto.Order, param *ParamDexFundNewOrder, db vmctxt_interface.VmDatabase, address types.Address, snapshotTM *time.Time) {
	order.Id = param.OrderId
	order.Address = address.Bytes()
	order.TradeToken = param.TradeToken.Bytes()
	order.QuoteToken = param.QuoteToken.Bytes()
	_, tradeTokenInfo := GetTokenInfo(db, param.TradeToken)
	order.TradeTokenDecimals = int32(tradeTokenInfo.Decimals)
	_, quoteTokenInfo := GetTokenInfo(db, param.QuoteToken)
	order.QuoteTokenDecimals = int32(quoteTokenInfo.Decimals)
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = param.Price
	order.Quantity = param.Quantity.Bytes()
	if order.Type == Limited {
		order.Amount = CalculateRawAmount(order.Quantity, order.Price, order.TradeTokenDecimals, order.QuoteTokenDecimals)
		if !order.Side { //buy
			order.LockedBuyFee = CalculateRawFee(order.Amount, MaxFeeRate())
		}
	}
	order.Status = Pending
	order.Timestamp = snapshotTM.Unix()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
}

func CheckSettleActions(actions *dexproto.SettleActions) error {
	if actions == nil || len(actions.FundActions) == 0 && len(actions.FeeActions) == 0 {
		return fmt.Errorf("settle actions is emtpy")
	}
	for _, v := range actions.FundActions {
		if len(v.Address) != 20 {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(v.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for settle")
		}
		if CmpToBigZero(v.IncAvailable) < 0 {
			return fmt.Errorf("negative incrAvailable for settle")
		}
		if CmpToBigZero(v.ReduceLocked) < 0 {
			return fmt.Errorf("negative reduceLocked for settle")
		}
		if CmpToBigZero(v.ReleaseLocked) < 0 {
			return fmt.Errorf("negative releaseLocked for settle")
		}
	}

	for _, fee := range actions.FeeActions {
		if len(fee.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for fee settle")
		}
		if CmpToBigZero(fee.Amount) <= 0 {
			return fmt.Errorf("negative feeAmount for settle")
		}
	}
	return nil
}

func GetAccountByTokeIdFromFund(dexFund *UserFund, token types.TokenTypeId) (account *dexproto.Account, exists bool) {
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

func GetAccountFundInfo(dexFund *UserFund, tokenId *types.TokenTypeId) ([]*Account, error) {
	if dexFund == nil {
		return nil, errors.New("fund user doesn't exist.")
	}
	var dexAccount = make([]*Account, 0)
	if tokenId != nil {
		for _, v := range dexFund.Accounts {
			if bytes.Equal(tokenId.Bytes(), v.Token) {
				var acc = &Account{}
				acc.Deserialize(v)
				dexAccount = append(dexAccount, acc)
				break
			}
		}
	} else {
		for _, v := range dexFund.Accounts {
			var acc = &Account{}
			acc.Deserialize(v)
			dexAccount = append(dexAccount, acc)
		}
	}
	return dexAccount, nil
}

func GetUserFundFromStorage(db vmctxt_interface.VmDatabase, address types.Address) (dexFund *UserFund, err error) {
	fundKey := GetUserFundKey(address)
	dexFund = &UserFund{}
	if fundBytes := db.GetStorage(&types.AddressDexFund, fundKey); len(fundBytes) > 0 {
		if dexFund, err = dexFund.DeSerialize(fundBytes); err != nil {
			return nil, err
		}
	}
	return dexFund, nil
}

func SaveUserFundToStorage(db vmctxt_interface.VmDatabase, address types.Address, dexFund *UserFund) error {
	if fundRes, err := dexFund.Serialize(); err == nil {
		db.SetStorage(GetUserFundKey(address), fundRes)
		return nil
	} else {
		return err
	}
}

func GetAccountByAddressAndTokenId(db vmctxt_interface.VmDatabase, address types.Address, token types.TokenTypeId) (account *dexproto.Account, exists bool, err error) {
	if dexFund, err := GetUserFundFromStorage(db, address); err != nil {
		return nil, false, err
	} else {
		account, exists = GetAccountByTokeIdFromFund(dexFund, token)
		return account, exists, nil
	}
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetCurrentFeeFromStorage(db vmctxt_interface.VmDatabase) (dexFee *Fee, err error) {
	if feeKey, err := GetFeeCurrentKeyFromStorage(db); err != nil {
		return nil, err
	} else {
		return getFeeByKeyFromStorage(db, feeKey)
	}
}

func GetFeeByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) (dexFee *Fee, err error) {
	return getFeeByKeyFromStorage(db, GetFeeKeyByPeriodId(periodId))
}

func getFeeByKeyFromStorage(db vmctxt_interface.VmDatabase, feeKey []byte) (dexFee *Fee, err error) {
	dexFee = &Fee{}
	if feeBytes := db.GetStorage(&types.AddressDexFund, feeKey); len(feeBytes) > 0 {
		if dexFee, err = dexFee.DeSerialize(feeBytes); err != nil {
			return nil, err
		} else {
			return dexFee, nil
		}
	} else {
		return nil, nil
	}
}

func GetNotDividedFeesByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) (map[uint64]*Fee, error) {
	var (
		dexFees = make(map[uint64]*Fee)
		dexFee *Fee
		err error
	)
	for {
		if dexFee, err = GetFeeByPeriodIdFromStorage(db, periodId); err != nil {
			return nil, err
		} else {
			if dexFee != nil {
				dexFees[periodId] = dexFee
			} else {
				return dexFees, nil
			}
		}
		periodId = dexFee.LastValidPeriod
	}
}

func SaveCurrentFeeToStorage(db vmctxt_interface.VmDatabase, fee *Fee) error {
	if feeKey, err := GetFeeCurrentKeyFromStorage(db); err != nil {
		return err
	} else {
		if feeBytes, err := proto.Marshal(fee); err == nil {
			db.SetStorage(feeKey, feeBytes)
			return nil
		} else {
			return err
		}
	}
}

func DeleteFeeByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) {
	db.SetStorage(GetFeeKeyByPeriodId(periodId), nil)
}

func GetFeeKeyByPeriodId(periodId uint64) []byte {
	return append(feeAccKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetFeeCurrentKeyFromStorage(db vmctxt_interface.VmDatabase) ([]byte, error) {
	if periodId, err := GetCurrentPeriodIdFromStorage(db); err != nil {
		return nil, err
	} else {
		return append(feeAccKeyPrefix, Uint64ToBytes(periodId)...), nil
	}
}

func GetFeeLastPeriodIdForRoll(db vmctxt_interface.VmDatabase) uint64 {
	if lastPeriodIdBytes := db.GetStorage(&types.AddressDexFund, lastFeePeriodKey); len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeLastPeriodIdForRoll(db vmctxt_interface.VmDatabase) error {
	if periodId, err := GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	} else {
		db.SetStorage(lastFeePeriodKey, Uint64ToBytes(periodId))
		return nil
	}
}

func GetVxFundsFromStorage(db vmctxt_interface.VmDatabase, address []byte) (vxFunds *VxFunds, err error) {
	vxFundsKey := GetVxFundsKey(address)
	vxFunds = &VxFunds{}
	if vxFundsBytes := db.GetStorage(&types.AddressDexFund, vxFundsKey); len(vxFundsBytes) > 0 {
		if vxFunds, err = vxFunds.DeSerialize(vxFundsBytes); err != nil {
			return nil, err
		} else {
			return vxFunds, nil
		}
	} else {
		return vxFunds, nil
	}
}

func SaveVxFundsToStorage(db vmctxt_interface.VmDatabase, address []byte, vxFunds *VxFunds) error {
	vxFundsKey := GetVxFundsKey(address)
	if vxFundsBytes, err := proto.Marshal(vxFunds); err == nil {
		db.SetStorage(vxFundsKey, vxFundsBytes)
		return nil
	} else {
		return err
	}
}

func MatchVxFundsByPeriod(vxFunds *VxFunds, periodId uint64) (bool, []byte, bool) {
	var (
		vxAmtBytes []byte
		matchIndex int
		needUpdateVxFunds bool
	)
	for i, fund := range vxFunds.Funds {
		if periodId >= fund.Period {
			vxAmtBytes = fund.Amount
			matchIndex = i
			if periodId == fund.Period {
				break
			}
		} else {
			break
		}
	}
	if len(vxAmtBytes) == 0 {
		return false, nil, false
	}
	if matchIndex > 0 {
		vxFunds.Funds = vxFunds.Funds[matchIndex:]
		needUpdateVxFunds = true
	}
	if len(vxFunds.Funds) > 1 && vxFunds.Funds[1].Period == periodId + 1 {
		vxFunds.Funds = vxFunds.Funds[1:]
		needUpdateVxFunds = true
	}
	return true, vxAmtBytes, needUpdateVxFunds
}

func GetCurrentPeriodIdFromStorage(db vmctxt_interface.VmDatabase) (uint64, error) {
	groupInfo := cabi.GetConsensusGroup(db, types.SNAPSHOT_GID)
	reader := core.NewReader(*db.GetGenesisSnapshotBlock().Timestamp, groupInfo)
	return reader.TimeToIndex(*db.CurrentSnapshotBlock().Timestamp)
}

func DeleteVxFundsFromStorage(db vmctxt_interface.VmDatabase, address []byte) {
	db.SetStorage(GetVxFundsKey(address), nil)
}

func GetVxFundsKey(address []byte) []byte {
	return append(VxFundKeyPrefix, address...)
}

func GetVxSumFundsFromStorage(db vmctxt_interface.VmDatabase) (vxSumFunds *VxFunds, err error) {
	vxSumFundsKey := GetVxSumFundsKey()
	vxSumFunds = &VxFunds{}
	if vxSumFundsBytes := db.GetStorage(&types.AddressDexFund, vxSumFundsKey); len(vxSumFundsBytes) > 0 {
		if vxSumFunds, err = vxSumFunds.DeSerialize(vxSumFundsBytes); err != nil {
			return nil, err
		} else {
			return vxSumFunds, nil
		}
	} else {
		return vxSumFunds, nil
	}
}

func SaveVxSumFundsToStorage(db vmctxt_interface.VmDatabase, vxSumFunds *VxFunds) error {
	vxSumFundsKey := GetVxSumFundsKey()
	if vxSumFundsBytes, err := proto.Marshal(vxSumFunds); err == nil {
		db.SetStorage(vxSumFundsKey, vxSumFundsBytes)
		return nil
	} else {
		return err
	}
}

func GetLastDividendIdFromStorage(db vmctxt_interface.VmDatabase) uint64 {
	if lastDividendIdBytes := db.GetStorage(&types.AddressDexFund, lastDividendIdKey); len(lastDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastDividendIdToStorage(db vmctxt_interface.VmDatabase, periodId uint64) {
	db.SetStorage(lastDividendIdKey, Uint64ToBytes(periodId))
}

func GetVxSumFundsKey() []byte {
	return vxSumFundsKey
}

func IsValidVxAmountBytesForDividend(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(VxDividendThreshold) >= 0
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func FromBytesToTokenTypeId(bytes []byte) (tokenId *types.TokenTypeId, err error) {
	tokenId = &types.TokenTypeId{}
	if err := tokenId.SetBytes(bytes); err == nil {
		return tokenId, nil
	} else {
		return nil, err
	}
}

func GetTokenInfo(db vmctxt_interface.VmDatabase, tokenId types.TokenTypeId) (error, *types.TokenInfo) {
	if tokenInfo := cabi.GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid"), nil
	} else {
		return nil, tokenInfo
	}
}

func Uint64ToBytes(value uint64) []byte {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, value)
	return bs
}

func ValidPrice(price string) bool {
	if len(price) == 0 {
		return false
	} else if pr, ok := new(big.Float).SetString(price); !ok || pr.Cmp(big.NewFloat(0)) <= 0 {
		return false
	} else {
		idx := strings.Index(price, ".")
		if idx > 0 && len(price)-idx >= 12 { // price max precision is 10 decimal
			return false
		}
	}
	return true
}
