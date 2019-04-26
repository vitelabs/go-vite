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
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var (
	ownerKey = []byte("own:")
	fundKeyPrefix = []byte("fd:") // fund:types.Address
	timestampKey = []byte("tts") // timerTimestamp

	UserFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	feeSumKeyPrefix     = []byte("fS:")    // feeSum:periodId
	donateFeeSumKeyPrefix = []byte("dfS:")    // donateFeeSum:periodId, feeSum for new market fee exceed
	lastFeeSumPeriodKey = []byte("lFSPId:") //

	VxFundKeyPrefix          = []byte("vxF:")    // vxFund:types.Address
	vxSumFundsKey            = []byte("vxFS:") // vxFundSum
	lastFeeDividendIdKey     = []byte("lDId:")
	lastMinedVxDividendIdKey = []byte("lMVDId:") //
	marketKeyPrefix          = []byte("mk:") // market: types.TokenTypeId,types.TokenTypeId

	pledgeForVipPrefix          = []byte("pldVip:") // pledgeForVip: types.Address
	pledgeForVxPrefix          = []byte("pldVx:") // pledgeForVx: types.Address

	tokenDecimalsPrefix = []byte("tk:") // token:uint8

	VxTokenBytes        = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	commonTokenPow      = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)
	VxDividendThreshold = new(big.Int).Mul(commonTokenPow, big.NewInt(10))     // 10
	VxMinedAmtPerPeriod = new(big.Int).Mul(commonTokenPow, big.NewInt(137000)) // 100,000,000/(365*2) = 136986
	NewMarketFeeAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	NewMarketFeeDividendAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))
	NewMarketFeeDonateAmount = new(big.Int).Sub(NewMarketFeeAmount, NewMarketFeeDividendAmount)

	bitcoinToken, _ = types.HexToTokenTypeId("tti_4e88a475c675971dab7ec917")
	ethToken, _     = types.HexToTokenTypeId("tti_2152a3d33c5e2fc90073fad4")
	usdtToken, _    = types.HexToTokenTypeId("tti_77a7a54d540d5c587dd666d6")
	QuoteTokenDecimals = map[types.TokenTypeId]int32{ledger.ViteTokenId : 18, bitcoinToken: 8, ethToken: 18, usdtToken : 8}

	viteMinAmount       = commonTokenPow // 1 VITE
	bitcoinMinAmount    = big.NewInt(1000000) //0.01 BTC
	ethMinAmount        = big.NewInt(10000000)//0.1 ETH
	usdtMinAmount       = big.NewInt(100000)//1 USDT
	QuoteTokenMinAmount = map[types.TokenTypeId]*big.Int{ledger.ViteTokenId : viteMinAmount, bitcoinToken: bitcoinMinAmount, ethToken: ethMinAmount, usdtToken : usdtMinAmount}

	PledgeForVxMinAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	PledgeForVipAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	PledgeForVipDuration int64 = 3600*24*30
)

const (
	priceIntMaxLen = 12
	priceDecimalMaxLen = 12
)

const (
  PledgeForVx =	iota + 1
  PledgeForVip
)

const (
	Pledge = iota + 1
	CancelPledge
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
	OrderType  int8
	Price      string
	Quantity   *big.Int
}

type ParamDexFundDividend struct {
	PeriodId uint64
}

type ParamDexSerializedData struct {
	Data []byte
}

type ParamDexFundNewMarket struct {
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
}

type ParamDexFundSetOwner struct {
	NewOwner types.Address
}

type ParamDexFundConfigMineMarket struct {
	AllowMine  bool
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
}

type ParamDexFundPledgeForVx struct {
	ActionType int8 // 1: pledge 2: cancel pledge
	Amount *big.Int
}

type ParamDexFundPledgeForVip struct {
	ActionType int8 // 1: pledge 2: cancel pledge
}

type ParamDexFundPledgeCallBack struct {
	Success bool
}

type ParamDexFundPledge struct {
	Source types.Address
	Beneficial types.Address
	PledgeType int8
	Amount *big.Int
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

type TokenInfo struct {
	dexproto.TokenInfo
}

func (tk *TokenInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&tk.TokenInfo)
}

func (tk *TokenInfo) DeSerialize(data []byte) (*TokenInfo, error) {
	tokenInfo := dexproto.TokenInfo{}
	if err := proto.Unmarshal(data, &tokenInfo); err != nil {
		return nil, err
	} else {
		return &TokenInfo{tokenInfo}, nil
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

type MarketInfo struct {
	dexproto.MarketInfo
}

func (mi *MarketInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&mi.MarketInfo)
}

func (mi *MarketInfo) DeSerialize(data []byte) (*MarketInfo, error) {
	marketInfo := dexproto.MarketInfo{}
	if err := proto.Unmarshal(data, &marketInfo); err != nil {
		return nil, err
	} else {
		return &MarketInfo{marketInfo}, nil
	}
}

type PledgeVip struct {
	dexproto.PledgeVip
}

func (pv *PledgeVip) Serialize() (data []byte, err error) {
	return proto.Marshal(&pv.PledgeVip)
}

func (pv *PledgeVip) DeSerialize(data []byte) (*PledgeVip, error) {
	pledgeVip := dexproto.PledgeVip{}
	if err := proto.Unmarshal(data, &pledgeVip); err != nil {
		return nil, err
	} else {
		return &PledgeVip{pledgeVip}, nil
	}
}

func EmitOrderFailLog(db vmctxt_interface.VmDatabase, orderInfo *dexproto.OrderInfo, errCode int) {
	orderFail := dexproto.OrderFail{}
	orderFail.OrderInfo = orderInfo
	orderFail.ErrCode = strconv.Itoa(errCode)
	event := NewOrderFailEvent{orderFail}

	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}

func EmitErrLog(db vmctxt_interface.VmDatabase, err error) {
	event := ErrEvent{err}
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
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
func GetNotDividedFeeSumsByPeriodIdFromStorage(db vmctxt_interface.VmDatabase, periodId uint64) (map[uint64]*FeeSumByPeriod, map[uint64]*big.Int, error) {
	var (
		dexFeeSums  = make(map[uint64]*FeeSumByPeriod)
		dexFeeSum  *FeeSumByPeriod
		donateFeeSums  = make(map[uint64]*big.Int)
		err        error
	)
	for {
		if dexFeeSum, err = GetFeeSumByPeriodIdFromStorage(db, periodId); err != nil {
			return nil, nil, err
		} else {
			if dexFeeSum == nil {
				if periodId > 0 {
					periodId --
					continue
				} else {
					return nil, nil, nil
				}
			} else {
				if !dexFeeSum.FeeDivided {
					dexFeeSums[periodId] = dexFeeSum
					if donateFeeSum := GetDonateFeeSum(db, periodId); donateFeeSum.Sign() > 0 {// when donateFee exists feeSum must exists
						donateFeeSums[periodId] = donateFeeSum
					}
				} else {
					return dexFeeSums, donateFeeSums, nil
				}
			}
		}
		periodId = dexFeeSum.LastValidPeriod
		if periodId == 0 {
			return dexFeeSums, donateFeeSums, nil
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

func SaveLastFeeDividendIdToStorage(db vmctxt_interface.VmDatabase, periodId uint64) {
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
	return reader.TimeToIndex(GetTimestamp(db))
}

func GetDonateFeeSum(db vmctxt_interface.VmDatabase, periodId uint64) *big.Int {
	if amountBytes := db.GetStorage(&types.AddressDexFund, GetDonateFeeSumKey(periodId)); len(amountBytes) > 0 {
		return new(big.Int).SetBytes(amountBytes)
	} else {
		return big.NewInt(0)
	}
}

func AddDonateFeeSum(db vmctxt_interface.VmDatabase) error {
	if period, err := GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	} else {
		donateFeeSum := GetDonateFeeSum(db, period)
		db.SetStorage(GetDonateFeeSumKey(period), new(big.Int).Add(donateFeeSum, NewMarketFeeDonateAmount).Bytes())
		return nil
	}
}

func DeleteDonateFeeSum(db vmctxt_interface.VmDatabase, period uint64) {
	db.SetStorage(GetDonateFeeSumKey(period), nil)
}

func GetDonateFeeSumKey(periodId uint64) []byte {
	return append(donateFeeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetMindedVxAmt(vxBalance *big.Int) (amtFroFeePerMarket, amtForPledge, amtForViteLabs *big.Int, success bool) {
	var toDivideTotal *big.Int
	if vxBalance.Sign() > 0 {
		if vxBalance.Cmp(VxMinedAmtPerPeriod) < 0 {
			toDivideTotal = vxBalance
		} else {
			toDivideTotal = VxMinedAmtPerPeriod
		}
		toDivideTotalF := new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetPrec(bigFloatPrec).SetString("0.2")
		amtFroFeePerMarket = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		amtForFeeTotal := new(big.Int).Mul(amtFroFeePerMarket, big.NewInt(4))
		proportion, _ = new(big.Float).SetPrec(bigFloatPrec).SetString("0.1")
		amtForViteLabs = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		amtForPledge = new(big.Int).Sub(toDivideTotal, amtForFeeTotal)
		amtForPledge.Sub(amtForPledge, amtForViteLabs)
		return amtFroFeePerMarket, amtForPledge, amtForViteLabs, true
	} else {
		return nil,nil,nil, false
	}
}

func GetTokenInfo(db vmctxt_interface.VmDatabase, token types.TokenTypeId) (tokenInfo *TokenInfo, err error) {
	if tokenInfoBytes := db.GetStorage(&types.AddressDexFund, GetTokenInfoKey(token)); len(tokenInfoBytes) > 0 {
		if tokenInfo, err = tokenInfo.DeSerialize(tokenInfoBytes); err != nil {
			return nil, err
		} else {
			return tokenInfo, nil
		}
	} else {
		return nil, nil
	}
}

func SaveTokenInfo(db vmctxt_interface.VmDatabase, token types.TokenTypeId, tokenInfo *TokenInfo) error {
	if tokenInfoBytes, err := tokenInfo.Serialize(); err == nil {
		db.SetStorage(GetTokenInfoKey(token), tokenInfoBytes)
		return nil
	} else {
		return err
	}
}

func GetTokenInfoKey(token types.TokenTypeId) []byte {
	return append(tokenDecimalsPrefix, token.Bytes()...)
}

func GetMarketInfo(db vmctxt_interface.VmDatabase, tradeToken, quoteToken types.TokenTypeId) (marketInfo *MarketInfo, err error) {
	if marketInfoBytes := db.GetStorage(&types.AddressDexFund, GetMarketInfoKey(tradeToken, quoteToken)); len(marketInfoBytes) > 0 {
		if marketInfo, err = marketInfo.DeSerialize(marketInfoBytes); err != nil {
			return nil, err
		} else {
			return marketInfo, nil
		}
	} else {
		return nil, TradeMarketNotExistsError
	}
}

func SaveMarketInfo(db vmctxt_interface.VmDatabase, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId) error {
	if marketInfoBytes, err := marketInfo.Serialize(); err == nil {
		db.SetStorage(GetMarketInfoKey(tradeToken, quoteToken), marketInfoBytes)
		return nil
	} else {
		return err
	}
}

func AddNewMarketEventLog(db vmctxt_interface.VmDatabase, newMarketEvent *NewMarketEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, newMarketEvent.GetTopicId())
	log.Data = newMarketEvent.toDataBytes()
	db.AddLog(log)
}

func GetMarketInfoKey(tradeToken, quoteToken types.TokenTypeId) []byte {
	re := make([]byte, len(marketKeyPrefix) + 2 * types.TokenTypeIdSize)
	copy(re[0:len(marketKeyPrefix)], marketKeyPrefix)
	copy(re[len(marketKeyPrefix):], tradeToken.Bytes())
	copy(re[len(marketKeyPrefix)+ types.TokenTypeIdSize:], quoteToken.Bytes())
	return re
}

func IsOwner(db vmctxt_interface.VmDatabase, address types.Address) bool {
	if storeOwner := db.GetStorage(&types.AddressDexFund, ownerKey); len(storeOwner) == 20 {
		if bytes.Compare(storeOwner, address.Bytes()) == 0 {
			return true
		}
	}
	return false
}

func SetOwner(db vmctxt_interface.VmDatabase, address types.Address) {
	db.SetStorage(ownerKey, address.Bytes())
}

func GetPledgeForVx(db vmctxt_interface.VmDatabase, address types.Address) *big.Int {
	if bs := db.GetStorage(&types.AddressDexFund, GetPledgeForVxKey(address)); len(bs) == 0 {
		return big.NewInt(0)
	} else {
		return new(big.Int).SetBytes(bs)
	}
}

func SavePledgeForVx(db vmctxt_interface.VmDatabase, address types.Address, amount *big.Int) {
	db.SetStorage(GetPledgeForVxKey(address), amount.Bytes())
}

func DeletePledgeForVx(db vmctxt_interface.VmDatabase, address types.Address) {
	db.SetStorage(GetPledgeForVxKey(address), nil)
}

func GetPledgeForVxKey(address types.Address) []byte {
	return append(pledgeForVxPrefix, address.Bytes()...)
}

func GetPledgeForVip(db vmctxt_interface.VmDatabase, address types.Address) (pledgeVip *PledgeVip, err error) {
	if pledgeVipBytes := db.GetStorage(&types.AddressDexFund, GetPledgeForVipKey(address)); len(pledgeVipBytes) > 0 {
		if pledgeVip, err = pledgeVip.DeSerialize(pledgeVipBytes); err != nil {
			return nil, err
		} else {
			return pledgeVip, nil
		}
	} else {
		return nil, PledgeForVipNotExistsErr
	}
}

func SavePledgeForVip(db vmctxt_interface.VmDatabase, address types.Address, pledgeVip *PledgeVip) error {
	if pledgeVipBytes, err := pledgeVip.Serialize(); err != nil {
		return err
	} else {
		db.SetStorage(GetPledgeForVipKey(address), pledgeVipBytes)
		return nil
	}
}

func DeletePledgeForVip(db vmctxt_interface.VmDatabase, address types.Address) {
	db.SetStorage(GetPledgeForVipKey(address), nil)
}

func GetPledgeForVipKey(address types.Address) []byte {
	return append(pledgeForVipPrefix, address.Bytes()...)
}

func GetTimestamp(db vmctxt_interface.VmDatabase) time.Time {
	//	GetTimerTimestamp(db)
	return *db.CurrentSnapshotBlock().Timestamp
}

func GetTimestampInt64(db vmctxt_interface.VmDatabase) int64 {
	return GetTimestamp(db).Unix()
}

func SetTimerTimestamp(db vmctxt_interface.VmDatabase, timestampInt int64)  {
	db.SetStorage(timestampKey, Uint64ToBytes(uint64(timestampInt)))
}

func GetTimerTimestamp(db vmctxt_interface.VmDatabase) (int64, error) {
	bs := db.GetStorage(&types.AddressDexFund, timestampKey)
	if len(bs) == 8 {
		return int64(BytesToUint64(bs)), nil
	} else {
		return 0, fmt.Errorf("get time stamp failed")
	}
}

func PriceToBytes(price string) []byte {
	parts := strings.Split(price, ".")
	var intPart, decimalPart string
	priceBytes := make([]byte, 10)
	if len(parts) == 2 {
		intPart = parts[0]
		decimalPart = parts[1]
	} else {
		intPart = parts[0]
	}
	if len(intPart) > 0 {
		intValue, _ := strconv.ParseUint(intPart, 10, 64)
		bs := make([]byte, 8)
		binary.BigEndian.PutUint64(bs, intValue)
		copy(priceBytes[:5], bs[3:8])
	}
	decimalLen := len(decimalPart)
	if decimalLen > 0 {
		if decimalLen < priceDecimalMaxLen {
			decimalPartArr := make([]byte, priceDecimalMaxLen)
			copy(decimalPartArr, decimalPart)
			//right pad 0
			for i := decimalLen; i < priceDecimalMaxLen; i++ {
				decimalPartArr[i] = '0'
			}
			decimalPart = string(decimalPartArr)
		}
		decimalValue, _ := strconv.ParseUint(decimalPart, 10, 64)
		bs := make([]byte, 8)
		binary.BigEndian.PutUint64(bs, decimalValue)
		copy(priceBytes[5:], bs[3:8])
	}
	return priceBytes
}

func BytesToPrice(priceBytes []byte) string {
	intBytes := make([]byte, 8)
	copy(intBytes[3:], priceBytes[:5])
	intValue := binary.BigEndian.Uint64(intBytes)
	decimalBytes := make([]byte, 8)
	copy(decimalBytes[3:], priceBytes[5:])
	decimalValue := binary.BigEndian.Uint64(decimalBytes)
	var intStr, decimalStr string
	if intValue == 0 {
		intStr = "0"
	} else {
		intStr = strconv.FormatUint(intValue, 10)
	}
	if decimalValue == 0 {
		return intStr
	} else {
		decimalStr = strconv.FormatUint(decimalValue, 10)
		decimalLen := len(decimalStr)
		decimalPartArr := make([]byte, priceDecimalMaxLen)
		if decimalLen == priceDecimalMaxLen {
			copy(decimalPartArr, decimalStr)
		} else {
			//left pad 0
			for i := 0; i < priceDecimalMaxLen-decimalLen; i++ {
				decimalPartArr[i] = '0'
			}
			copy(decimalPartArr[priceDecimalMaxLen-decimalLen:], decimalStr)
		}
		var rightTruncate = 0
		for i := priceDecimalMaxLen-1; i >= 0; i -- {
			if decimalPartArr[i] == '0' {
				rightTruncate++
			} else {
				break
			}
		}
		return fmt.Sprintf("%s.%s", intStr, string(decimalPartArr[:priceDecimalMaxLen-rightTruncate]))
	}
}

func Uint64ToBytes(value uint64) []byte {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, value)
	return bs
}

func BytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}
