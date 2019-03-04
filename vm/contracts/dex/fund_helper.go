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
	"strings"
	"time"
)

var (
	fundKeyPrefix = []byte("fd:") // fund:types.Address

	UserFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	feeSumKeyPrefix     = []byte("fS:")    // feeSum:periodId
	lastFeeSumPeriodKey = []byte("lFSPId:") //

	VxFundKeyPrefix      = []byte("vxF:")    // vxFund:types.Address
	vxSumFundsKey        = []byte("vxFS:") // vxFundSum
	lastFeeDividendIdKey = []byte("lDId:")
	lastMinedVxDividendIdKey = []byte("lMVDId:")

	VxTokenBytes        = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	vxTokenPow = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)
	VxDividendThreshold = new(big.Int).Mul(vxTokenPow, big.NewInt(10))     // 10
	VxMinedAmtPerPeriod = new(big.Int).Mul(vxTokenPow, big.NewInt(137000)) // 100,000,000/(365*2) = 136986
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

type FeeSumByPeriod struct {
	dexproto.FeeSumByPeriod
}

func (df *FeeSumByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.FeeSumByPeriod)
}

func (df *FeeSumByPeriod) DeSerialize(feeSumData []byte) (dexFeeSum *FeeSumByPeriod, err error) {
	protoFeeSum := dexproto.FeeSumByPeriod{}
	if err := proto.Unmarshal(feeSumData, &protoFeeSum); err != nil {
		return nil, err
	} else {
		return &FeeSumByPeriod{protoFeeSum}, nil
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

type UserFees struct {
	dexproto.UserFees
}

func (ufs *UserFees) Serialize() (data []byte, err error) {
	return proto.Marshal(&ufs.UserFees)
}

func (ufs *UserFees) DeSerialize(userFeesData []byte) (*UserFees, error) {
	protoUserFees := dexproto.UserFees{}
	if err := proto.Unmarshal(userFeesData, &protoUserFees); err != nil {
		return nil, err
	} else {
		return &UserFees{protoUserFees}, nil
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
	// TODO add market order support
	if orderParam.OrderType != Limited {
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
	for _, fund := range actions.FundActions {
		if len(fund.Address) != 20 {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(fund.FundSettles) == 0 {
			return fmt.Errorf("no user funds to settle")
		}
	}

	for _, fee := range actions.FeeActions {
		if len(fee.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for fee settle")
		}
		if len(fee.UserFeeSettles) == 0 {
			return fmt.Errorf("no userFees to settle")
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
	if dexFund == nil || len(dexFund.Accounts) == 0 {
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
	dexFund = &UserFund{}
	if fundBytes := db.GetStorage(&types.AddressDexFund, GetUserFundKey(address)); len(fundBytes) > 0 {
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

func BatchSaveUserFund(db vmctxt_interface.VmDatabase, address types.Address, funds map[types.TokenTypeId]*big.Int) error {
	if userFund, err := GetUserFundFromStorage(db, address); err != nil {
		return err
	} else {
		for _, acc := range userFund.Accounts {
			if tk, err := types.BytesToTokenTypeId(acc.Token); err != nil {
				return err
			} else {
				if amt, ok := funds[tk]; ok {
					acc.Available = AddBigInt(acc.Available, amt.Bytes())
					delete(funds, tk)
				}
			}
		}
		for tokenId, amt := range funds {
			acc := &dexproto.Account{}
			acc.Token = tokenId.Bytes()
			acc.Available = amt.Bytes()
			userFund.Accounts = append(userFund.Accounts, acc)
		}
		if err := SaveUserFundToStorage(db, address, userFund); err != nil {
			return err
		}
	}
	return nil
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetCurrentFeeSumFromStorage(db vmctxt_interface.VmDatabase) (feeSum *FeeSumByPeriod, err error) {
	if feeKey, err := GetFeeSumCurrentKeyFromStorage(db); err != nil {
		return nil, err
	} else {
		return getFeeSumByKeyFromStorage(db, feeKey)
	}
}

func GetFeeSumByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) (feeSum *FeeSumByPeriod, err error) {
	return getFeeSumByKeyFromStorage(db, GetFeeSumKeyByPeriodId(periodId))
}

func getFeeSumByKeyFromStorage(db vmctxt_interface.VmDatabase, feeKey []byte) (feeSum *FeeSumByPeriod, err error) {
	feeSum = &FeeSumByPeriod{}
	if feeBytes := db.GetStorage(&types.AddressDexFund, feeKey); len(feeBytes) > 0 {
		if feeSum, err = feeSum.DeSerialize(feeBytes); err != nil {
			return nil, err
		} else {
			return feeSum, nil
		}
	} else {
		return nil, nil
	}
}

//get all feeSums that not divided yet
func GetNotDividedFeeSumsByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) (map[uint64]*FeeSumByPeriod, error) {
	var (
		dexFeeSums  = make(map[uint64]*FeeSumByPeriod)
		dexFeeSum  *FeeSumByPeriod
		err        error
	)
	for {
		if dexFeeSum, err = GetFeeSumByPeriodIdFromStorage(db, periodId); err != nil {
			return nil, err
		} else {
			if dexFeeSum == nil {
				if periodId > 0 {
					periodId --
					continue
				} else {
					return nil, nil
				}
			} else {
				if !dexFeeSum.FeeDivided {
					dexFeeSums[periodId] = dexFeeSum
				} else {
					return dexFeeSums, nil
				}
			}
		}
		periodId = dexFeeSum.LastValidPeriod
		if periodId == 0 {
			return dexFeeSums, nil
		}
	}
}

func SaveCurrentFeeSumToStorage(db vmctxt_interface.VmDatabase, feeSum *FeeSumByPeriod) error {
	if feeSumKey, err := GetFeeSumCurrentKeyFromStorage(db); err != nil {
		return err
	} else {
		if feeSumBytes, err := feeSum.Serialize(); err == nil {
			db.SetStorage(feeSumKey, feeSumBytes)
			return nil
		} else {
			return err
		}
	}
}
//fee sum used both by fee dividend and mined vx dividend
func MarkFeeSumAsFeeDivided(db vmctxt_interface.VmDatabase, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.MinedVxDivided {
		db.SetStorage(GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FeeDivided = true
		sumBytes, _ := feeSum.Serialize()
		db.SetStorage(GetFeeSumKeyByPeriodId(periodId), sumBytes)
	}
}

func MarkFeeSumAsMinedVxDivided(db vmctxt_interface.VmDatabase, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.FeeDivided {
		db.SetStorage(GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.MinedVxDivided = true
		sumBytes, _ := feeSum.Serialize()
		db.SetStorage(GetFeeSumKeyByPeriodId(periodId), sumBytes)
	}
}

func GetFeeSumKeyByPeriodId(periodId uint64) []byte {
	return append(feeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetFeeSumCurrentKeyFromStorage(db vmctxt_interface.VmDatabase) ([]byte, error) {
	if periodId, err := GetCurrentPeriodIdFromStorage(db); err != nil {
		return nil, err
	} else {
		return GetFeeSumKeyByPeriodId(periodId), nil
	}
}

func GetFeeSumLastPeriodIdForRoll(db vmctxt_interface.VmDatabase) uint64 {
	if lastPeriodIdBytes := db.GetStorage(&types.AddressDexFund, lastFeeSumPeriodKey); len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeSumLastPeriodIdForRoll(db vmctxt_interface.VmDatabase) error {
	if periodId, err := GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	} else {
		db.SetStorage(lastFeeSumPeriodKey, Uint64ToBytes(periodId))
		return nil
	}
}

func GetUserFeesFromStorage(db vmctxt_interface.VmDatabase, address []byte) (userFees *UserFees, err error) {
	userFees = &UserFees{}
	if userFeesBytes := db.GetStorage(&types.AddressDexFund, GetUserFeesKey(address)); len(userFeesBytes) > 0 {
		if userFees, err = userFees.DeSerialize(userFeesBytes); err != nil {
			return nil, err
		} else {
			return userFees, nil
		}
	} else {
		return nil, nil
	}
}

func SaveUserFeesToStorage(db vmctxt_interface.VmDatabase, address []byte, userFees *UserFees) error {
	if userFeesBytes, err := userFees.Serialize(); err == nil {
		db.SetStorage(GetUserFeesKey(address), userFeesBytes)
		return nil
	} else {
		return err
	}
}

func DeleteUserFeesFromStorage(db vmctxt_interface.VmDatabase, address []byte) {
	db.SetStorage(GetUserFeesKey(address), nil)
}

func GetUserFeesKey(address []byte) []byte {
	return append(UserFeeKeyPrefix, address...)
}


func GetVxFundsFromStorage(db vmctxt_interface.VmDatabase, address []byte) (vxFunds *VxFunds, err error) {
	if vxFundsBytes := db.GetStorage(&types.AddressDexFund, GetVxFundsKey(address)); len(vxFundsBytes) > 0 {
		if vxFunds, err = vxFunds.DeSerialize(vxFundsBytes); err != nil {
			return nil, err
		} else {
			return vxFunds, nil
		}
	} else {
		return nil, nil
	}
}

func SaveVxFundsToStorage(db vmctxt_interface.VmDatabase, address []byte, vxFunds *VxFunds) error {
	if vxFundsBytes, err := vxFunds.Serialize(); err == nil {
		db.SetStorage(GetVxFundsKey(address), vxFundsBytes)
		return nil
	} else {
		return err
	}
}

func MatchVxFundsByPeriod(vxFunds *VxFunds, periodId uint64, checkDelete bool) (bool, []byte, bool, bool) {
	var (
		vxAmtBytes []byte
		matchIndex int
		needUpdateVxFunds bool
	)
	for i, fund := range vxFunds.Funds {
		if periodId >= fund.Period {
			vxAmtBytes = fund.Amount
			matchIndex = i
		} else {
			break
		}
	}
	if len(vxAmtBytes) == 0 {
		return false, nil, false, checkDelete && CheckUserVxFundsCanBeDelete(vxFunds)
	}
	if matchIndex > 0 {//remove obsolete items, but leave current matched item
		vxFunds.Funds = vxFunds.Funds[matchIndex:]
		needUpdateVxFunds = true
	}
	if len(vxFunds.Funds) > 1 && vxFunds.Funds[1].Period == periodId + 1 {
		vxFunds.Funds = vxFunds.Funds[1:]
		needUpdateVxFunds = true
	}
	return true, vxAmtBytes, needUpdateVxFunds, checkDelete && CheckUserVxFundsCanBeDelete(vxFunds)
}

func CheckUserVxFundsCanBeDelete(vxFunds *VxFunds) bool {
	return len(vxFunds.Funds) == 1 && !IsValidVxAmountBytesForDividend(vxFunds.Funds[0].Amount)
}

func DeleteVxFundsFromStorage(db vmctxt_interface.VmDatabase, address []byte) {
	db.SetStorage(GetVxFundsKey(address), nil)
}

func GetVxFundsKey(address []byte) []byte {
	return append(VxFundKeyPrefix, address...)
}

func GetVxSumFundsFromStorage(db vmctxt_interface.VmDatabase) (vxSumFunds *VxFunds, err error) {
	if vxSumFundsBytes := db.GetStorage(&types.AddressDexFund, vxSumFundsKey); len(vxSumFundsBytes) > 0 {
		if vxSumFunds, err = vxSumFunds.DeSerialize(vxSumFundsBytes); err != nil {
			return nil, err
		} else {
			return vxSumFunds, nil
		}
	} else {
		return nil, nil
	}
}

func SaveVxSumFundsToStorage(db vmctxt_interface.VmDatabase, vxSumFunds *VxFunds) error {
	if vxSumFundsBytes, err := vxSumFunds.Serialize(); err == nil {
		db.SetStorage(vxSumFundsKey, vxSumFundsBytes)
		return nil
	} else {
		return err
	}
}

func GetLastFeeDividendIdFromStorage(db vmctxt_interface.VmDatabase) uint64 {
	if lastFeeDividendIdBytes := db.GetStorage(&types.AddressDexFund, lastFeeDividendIdKey); len(lastFeeDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastFeeDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastDividendIdToStorage(db vmctxt_interface.VmDatabase, periodId uint64) {
	db.SetStorage(lastFeeDividendIdKey, Uint64ToBytes(periodId))
}

func GetLastMinedVxDividendIdFromStorage(db vmctxt_interface.VmDatabase) uint64 {
	if lastMinedVxDividendIdBytes := db.GetStorage(&types.AddressDexFund, lastMinedVxDividendIdKey); len(lastMinedVxDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastMinedVxDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastMinedVxDividendIdToStorage(db vmctxt_interface.VmDatabase, periodId uint64) {
	db.SetStorage(lastMinedVxDividendIdKey, Uint64ToBytes(periodId))
}

func IsValidVxAmountBytesForDividend(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(VxDividendThreshold) >= 0
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func GetCurrentPeriodIdFromStorage(db vmctxt_interface.VmDatabase) (uint64, error) {
	groupInfo := cabi.GetConsensusGroup(db, types.SNAPSHOT_GID)
	reader := core.NewReader(*db.GetGenesisSnapshotBlock().Timestamp, groupInfo)
	return reader.TimeToIndex(*db.CurrentSnapshotBlock().Timestamp)
}

func GetTokenInfo(db vmctxt_interface.VmDatabase, tokenId types.TokenTypeId) (error, *types.TokenInfo) {
	if tokenInfo := cabi.GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid"), nil
	} else {
		return nil, tokenInfo
	}
}

func GetMindedVxAmt(vxBalance *big.Int) (amtFroFeePerMarket, amtForPledge, amtForViteLabs *big.Int, success bool) {
	var toDivideTotal *big.Int
	if vxBalance.Sign() > 0 {
		if vxBalance.Cmp(VxMinedAmtPerPeriod) < 0 {
			toDivideTotal = vxBalance
		} else {
			toDivideTotal = VxMinedAmtPerPeriod
		}
		toDivideTotalF := new(big.Float).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetString("0.2")
		amtFroFeePerMarket = RoundAmount(new(big.Float).Mul(toDivideTotalF, proportion))
		amtForFeeTotal := new(big.Int).Mul(amtFroFeePerMarket, big.NewInt(4))
		proportion, _ = new(big.Float).SetString("0.1")
		amtForViteLabs = RoundAmount(new(big.Float).Mul(toDivideTotalF, proportion))
		amtForPledge = new(big.Int).Sub(toDivideTotal, amtForFeeTotal)
		amtForPledge.Sub(amtForPledge, amtForViteLabs)
		return amtFroFeePerMarket, amtForPledge, amtForViteLabs, true
	} else {
		return nil,nil,nil, false
	}
}

func DivideByProportion(totalReferAmt, partReferAmt, dividedReferAmt, toDivideTotalAmt, toDivideLeaveAmt *big.Int) (proportionAmt *big.Int, finished bool) {
	dividedReferAmt.Add(dividedReferAmt, partReferAmt)
	proportion := new(big.Float).Quo(new(big.Float).SetInt(partReferAmt), new(big.Float).SetInt(totalReferAmt))
	proportionAmt = RoundAmount(new(big.Float).Mul(new(big.Float).SetInt(toDivideTotalAmt), proportion))
	toDivideLeaveNewAmt := new(big.Int).Sub(toDivideLeaveAmt, proportionAmt)
	if toDivideLeaveNewAmt.Sign() <= 0 || dividedReferAmt.Cmp(totalReferAmt) >= 0 {
		proportionAmt.Set(toDivideLeaveAmt)
		finished = true
	} else {
		toDivideLeaveAmt.Set(toDivideLeaveNewAmt)
	}
	return proportionAmt, finished
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
		if idx > 0 && len(price) - idx >= 10 { // max price precision is 8 decimals
			return false
		}
	}
	return true
}
