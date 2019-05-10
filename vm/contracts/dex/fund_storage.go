package dex

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"strconv"
)

var (
	ownerKey      = []byte("own:")
	fundKeyPrefix = []byte("fd:") // fund:types.Address
	timestampKey  = []byte("tts") // timerTimestamp

	UserFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	feeSumKeyPrefix     = []byte("fS:")     // feeSum:periodId
	lastFeeSumPeriodKey = []byte("lFSPId:") //

	donateFeeSumKeyPrefix      = []byte("dfS:")   // donateFeeSum:periodId, feeSum for new market fee
	pendingNewMarketFeeSumKey  = []byte("pnmfS:") // pending feeSum for new market
	pendingNewMarketActionsKey = []byte("pmkas:")
	marketIdKey                = []byte("mkId:")
	orderIdSerialNoKey         = []byte("orIdSl:")

	timerContractAddressKey = []byte("tmA:")

	VxFundKeyPrefix          = []byte("vxF:")  // vxFund:types.Address
	vxSumFundsKey            = []byte("vxFS:") // vxFundSum
	lastFeeDividendIdKey     = []byte("lDId:")
	lastMinedVxDividendIdKey = []byte("lMVDId:") //
	marketKeyPrefix          = []byte("mk:")     // market: types.TokenTypeId,types.TokenTypeId

	pledgeForVipPrefix = []byte("pldVip:") // pledgeForVip: types.Address
	pledgeForVxPrefix  = []byte("pldVx:")  // pledgeForVx: types.Address

	tokenInfoPrefix = []byte("tk:") // token:tokenId

	VxTokenBytes               = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	commonTokenPow             = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)
	VxDividendThreshold        = new(big.Int).Mul(commonTokenPow, big.NewInt(10))     // 10
	VxMinedAmtPerPeriod        = new(big.Int).Mul(commonTokenPow, big.NewInt(137000)) // 100,000,000/(365*2) = 136986
	NewMarketFeeAmount         = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	NewMarketFeeDividendAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))
	NewMarketFeeDonateAmount   = new(big.Int).Sub(NewMarketFeeAmount, NewMarketFeeDividendAmount)

	PledgeForVxMinAmount       = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	PledgeForVipAmount         = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	PledgeForVipDuration int64 = 3600 * 24 * 30

	bitcoinToken, _ = types.HexToTokenTypeId("tti_4e88a475c675971dab7ec917")
	ethToken, _     = types.HexToTokenTypeId("tti_2152a3d33c5e2fc90073fad4")
	usdtToken, _    = types.HexToTokenTypeId("tti_77a7a54d540d5c587dd666d6")

	QuoteTokenInfos = map[types.TokenTypeId]*TokenInfo{
		ledger.ViteTokenId: &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "VITE", Index: -1}},
		bitcoinToken:       &TokenInfo{dexproto.TokenInfo{Decimals: 8, Symbol: "BTC", Index: 1}},
		ethToken:           &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "ETH", Index: 1}},
		usdtToken:          &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "USDT", Index: 1}},
	}

	viteMinAmount       = commonTokenPow       // 1 VITE
	bitcoinMinAmount    = big.NewInt(1000000)  //0.01 BTC
	ethMinAmount        = big.NewInt(10000000) //0.1 ETH
	usdtMinAmount       = big.NewInt(100000)   //1 USDT
	QuoteTokenMinAmount = map[types.TokenTypeId]*big.Int{ledger.ViteTokenId: viteMinAmount, bitcoinToken: bitcoinMinAmount, ethToken: ethMinAmount, usdtToken: usdtMinAmount}
)

const (
	priceIntMaxLen     = 12
	priceDecimalMaxLen = 12
)

const (
	PledgeForVx = iota + 1
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
	Amount     *big.Int
}

type ParamDexFundPledgeForVip struct {
	ActionType int8 // 1: pledge 2: cancel pledge
}

type ParamDexFundPledgeCallBack struct {
	Success bool
}

type ParamDexFundPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	PledgeType    uint8
}

type ParamDexFundCancelPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Amount        *big.Int
	PledgeType    uint8
}

type ParamDexFundGetTokenInfoCallback struct {
	Exist       bool
	Decimals    uint8
	TokenSymbol string
	Index       uint16
}

type ParamDexFundGetTokenInfo struct {
	TokenId types.TokenTypeId
}

type ParamDexConfigTimerAddress struct {
	Address types.Address
}

type ParamDexFundNotifyTime struct {
	Timestamp int64
}

type UserFund struct {
	dexproto.Fund
}

type FundSerializable interface {
	Serialize() ([]byte, error)
	DeSerialize([]byte) error
}

func (df *UserFund) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.Fund)
}

func (df *UserFund) DeSerialize(fundData []byte) (err error) {
	protoFund := dexproto.Fund{}
	if err := proto.Unmarshal(fundData, &protoFund); err != nil {
		return err
	} else {
		df.Fund = protoFund
		return nil
	}
}

type FeeSumByPeriod struct {
	dexproto.FeeSumByPeriod
}

func (df *FeeSumByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.FeeSumByPeriod)
}

func (df *FeeSumByPeriod) DeSerialize(feeSumData []byte) (err error) {
	protoFeeSum := dexproto.FeeSumByPeriod{}
	if err := proto.Unmarshal(feeSumData, &protoFeeSum); err != nil {
		return err
	} else {
		df.FeeSumByPeriod = protoFeeSum
		return nil
	}
}

type VxFunds struct {
	dexproto.VxFunds
}

func (dvf *VxFunds) Serialize() (data []byte, err error) {
	return proto.Marshal(&dvf.VxFunds)
}

func (dvf *VxFunds) DeSerialize(vxFundsData []byte) error {
	protoVxFunds := dexproto.VxFunds{}
	if err := proto.Unmarshal(vxFundsData, &protoVxFunds); err != nil {
		return err
	} else {
		dvf.VxFunds = protoVxFunds
		return nil
	}
}

type TokenInfo struct {
	dexproto.TokenInfo
}

func (tk *TokenInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&tk.TokenInfo)
}

func (tk *TokenInfo) DeSerialize(data []byte) error {
	tokenInfo := dexproto.TokenInfo{}
	if err := proto.Unmarshal(data, &tokenInfo); err != nil {
		return err
	} else {
		tk.TokenInfo = tokenInfo
		return nil
	}
}

type UserFees struct {
	dexproto.UserFees
}

func (ufs *UserFees) Serialize() (data []byte, err error) {
	return proto.Marshal(&ufs.UserFees)
}

func (ufs *UserFees) DeSerialize(userFeesData []byte) error {
	protoUserFees := dexproto.UserFees{}
	if err := proto.Unmarshal(userFeesData, &protoUserFees); err != nil {
		return err
	} else {
		ufs.UserFees = protoUserFees
		return nil
	}
}

type MarketInfo struct {
	dexproto.MarketInfo
}

func (mi *MarketInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&mi.MarketInfo)
}

func (mi *MarketInfo) DeSerialize(data []byte) error {
	marketInfo := dexproto.MarketInfo{}
	if err := proto.Unmarshal(data, &marketInfo); err != nil {
		return err
	} else {
		mi.MarketInfo = marketInfo
		return nil
	}
}

type PendingNewMarkets struct {
	dexproto.PendingNewMarkets
}

func (pnm *PendingNewMarkets) Serialize() (data []byte, err error) {
	return proto.Marshal(&pnm.PendingNewMarkets)
}

func (pnm *PendingNewMarkets) DeSerialize(data []byte) error {
	pendingNewMarkets := dexproto.PendingNewMarkets{}
	if err := proto.Unmarshal(data, &pendingNewMarkets); err != nil {
		return err
	} else {
		pnm.PendingNewMarkets = pendingNewMarkets
		return nil
	}
}

type OrderIdSerialNo struct {
	dexproto.OrderIdSerialNo
}

func (osn *OrderIdSerialNo) Serialize() (data []byte, err error) {
	return proto.Marshal(&osn.OrderIdSerialNo)
}

func (osn *OrderIdSerialNo) DeSerialize(data []byte) error {
	orderIdSerialNo := dexproto.OrderIdSerialNo{}
	if err := proto.Unmarshal(data, &orderIdSerialNo); err != nil {
		return err
	} else {
		osn.OrderIdSerialNo = orderIdSerialNo
		return nil
	}
}

type PledgeVip struct {
	dexproto.PledgeVip
}

func (pv *PledgeVip) Serialize() (data []byte, err error) {
	return proto.Marshal(&pv.PledgeVip)
}

func (pv *PledgeVip) DeSerialize(data []byte) error {
	pledgeVip := dexproto.PledgeVip{}
	if err := proto.Unmarshal(data, &pledgeVip); err != nil {
		return err
	} else {
		pv.PledgeVip = pledgeVip
		return nil
	}
}

func EmitOrderFailLog(db vm_db.VmDb, orderInfo *dexproto.OrderInfo, errCode int) {
	orderFail := dexproto.OrderFail{}
	orderFail.OrderInfo = orderInfo
	orderFail.ErrCode = strconv.Itoa(errCode)
	event := NewOrderFailEvent{orderFail}

	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}

func EmitErrLog(db vm_db.VmDb, err error) {
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

func GetUserFundFromStorage(db vm_db.VmDb, address types.Address) (*UserFund, error) {
	dexFund := &UserFund{}
	if err := getFundValueFromDb(db, GetUserFundKey(address), dexFund); err == NotFoundValueFromDb {
		return dexFund, nil
	} else {
		return dexFund, err
	}
}

func SaveUserFundToStorage(db vm_db.VmDb, address types.Address, dexFund *UserFund) error {
	return saveFunValueToDb(db, GetUserFundKey(address), dexFund)
}

func BatchSaveUserFund(db vm_db.VmDb, address types.Address, funds map[types.TokenTypeId]*big.Int) error {
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

func GetCurrentFeeSumFromStorage(db vm_db.VmDb, reader util.ConsensusReader) (feeSum *FeeSumByPeriod, err error) {
	return getFeeSumByKeyFromStorage(db, GetFeeSumCurrentKeyFromStorage(db, reader))
}

func GetFeeSumByPeriodIdFromStorage(db vm_db.VmDb, periodId uint64) (feeSum *FeeSumByPeriod, err error) {
	return getFeeSumByKeyFromStorage(db, GetFeeSumKeyByPeriodId(periodId))
}

func getFeeSumByKeyFromStorage(db vm_db.VmDb, feeKey []byte) (*FeeSumByPeriod, error) {
	feeSum := &FeeSumByPeriod{}
	if err := getFundValueFromDb(db, feeKey, feeSum); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return feeSum, err
	}
}

//get all feeSums that not divided yet
func GetNotDividedFeeSumsByPeriodIdFromStorage(db vm_db.VmDb, periodId uint64) (map[uint64]*FeeSumByPeriod, map[uint64]*big.Int, error) {
	var (
		dexFeeSums    = make(map[uint64]*FeeSumByPeriod)
		dexFeeSum     *FeeSumByPeriod
		donateFeeSums = make(map[uint64]*big.Int)
		err           error
	)
	for {
		if dexFeeSum, err = GetFeeSumByPeriodIdFromStorage(db, periodId); err != nil {
			return nil, nil, err
		} else {
			if dexFeeSum == nil {
				if periodId > 0 {
					periodId--
					continue
				} else {
					return nil, nil, nil
				}
			} else {
				if !dexFeeSum.FeeDivided {
					dexFeeSums[periodId] = dexFeeSum
					if donateFeeSum := GetDonateFeeSum(db, periodId); donateFeeSum.Sign() > 0 { // when donateFee exists feeSum must exists
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

func SaveCurrentFeeSumToStorage(db vm_db.VmDb, reader util.ConsensusReader, feeSum *FeeSumByPeriod) error {
	feeSumKey := GetFeeSumCurrentKeyFromStorage(db, reader)
	return saveFunValueToDb(db, feeSumKey, feeSum)
}

//fee sum used both by fee dividend and mined vx dividend
func MarkFeeSumAsFeeDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) error {
	if feeSum.MinedVxDivided {
		return db.SetValue(GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FeeDivided = true
		return saveFunValueToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func MarkFeeSumAsMinedVxDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) error {
	if feeSum.FeeDivided {
		return db.SetValue(GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.MinedVxDivided = true
		return saveFunValueToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func GetFeeSumKeyByPeriodId(periodId uint64) []byte {
	return append(feeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetFeeSumCurrentKeyFromStorage(db vm_db.VmDb, reader util.ConsensusReader) []byte {
	return GetFeeSumKeyByPeriodId(GetCurrentPeriodIdFromStorage(db, reader))
}

func GetFeeSumLastPeriodIdForRoll(db vm_db.VmDb) uint64 {
	if lastPeriodIdBytes, err := db.GetValue(lastFeeSumPeriodKey); err == nil && len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeSumLastPeriodIdForRoll(db vm_db.VmDb, reader util.ConsensusReader) error {
	periodId := GetCurrentPeriodIdFromStorage(db, reader)
	return db.SetValue(lastFeeSumPeriodKey, Uint64ToBytes(periodId))
}

func GetUserFeesFromStorage(db vm_db.VmDb, address []byte) (*UserFees, error) {
	userFees := &UserFees{}
	if err := getFundValueFromDb(db, GetUserFeesKey(address), userFees); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return userFees, err
	}
}

func SaveUserFeesToStorage(db vm_db.VmDb, address []byte, userFees *UserFees) error {
	return saveFunValueToDb(db, GetUserFeesKey(address), userFees)
}

func DeleteUserFeesFromStorage(db vm_db.VmDb, address []byte) error {
	return db.SetValue(GetUserFeesKey(address), nil)
}

func GetUserFeesKey(address []byte) []byte {
	return append(UserFeeKeyPrefix, address...)
}

func GetVxFundsFromStorage(db vm_db.VmDb, address []byte) (*VxFunds, error) {
	var vxFunds = &VxFunds{}
	if err := getFundValueFromDb(db, GetVxFundsKey(address), vxFunds); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return vxFunds, err
	}
}

func SaveVxFundsToStorage(db vm_db.VmDb, address []byte, vxFunds *VxFunds) error {
	return saveFunValueToDb(db, GetVxFundsKey(address), vxFunds)
}

func MatchVxFundsByPeriod(vxFunds *VxFunds, periodId uint64, checkDelete bool) (bool, []byte, bool, bool) {
	var (
		vxAmtBytes        []byte
		matchIndex        int
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
	if matchIndex > 0 { //remove obsolete items, but leave current matched item
		vxFunds.Funds = vxFunds.Funds[matchIndex:]
		needUpdateVxFunds = true
	}
	if len(vxFunds.Funds) > 1 && vxFunds.Funds[1].Period == periodId+1 {
		vxFunds.Funds = vxFunds.Funds[1:]
		needUpdateVxFunds = true
	}
	return true, vxAmtBytes, needUpdateVxFunds, checkDelete && CheckUserVxFundsCanBeDelete(vxFunds)
}

func CheckUserVxFundsCanBeDelete(vxFunds *VxFunds) bool {
	return len(vxFunds.Funds) == 1 && !IsValidVxAmountBytesForDividend(vxFunds.Funds[0].Amount)
}

func DeleteVxFundsFromStorage(db vm_db.VmDb, address []byte) error {
	return db.SetValue(GetVxFundsKey(address), nil)
}

func GetVxFundsKey(address []byte) []byte {
	return append(VxFundKeyPrefix, address...)
}

func GetVxSumFundsFromDb(db vm_db.VmDb) (*VxFunds, error) {
	vxSumFunds := &VxFunds{}
	if err := getFundValueFromDb(db, vxSumFundsKey, vxSumFunds); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return vxSumFunds, err
	}
}

func SaveVxSumFundsToDb(db vm_db.VmDb, vxSumFunds *VxFunds) error {
	if vxSumFundsBytes, err := vxSumFunds.Serialize(); err == nil {
		return db.SetValue(vxSumFundsKey, vxSumFundsBytes)
	} else {
		return err
	}
}

func GetLastFeeDividendIdFromStorage(db vm_db.VmDb) uint64 {
	if lastFeeDividendIdBytes, err := db.GetValue(lastFeeDividendIdKey); err == nil && len(lastFeeDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastFeeDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastFeeDividendIdToStorage(db vm_db.VmDb, periodId uint64) error {
	return db.SetValue(lastFeeDividendIdKey, Uint64ToBytes(periodId))
}

func GetLastMinedVxDividendIdFromStorage(db vm_db.VmDb) uint64 {
	if lastMinedVxDividendIdBytes, err := db.GetValue(lastMinedVxDividendIdKey); err == nil && len(lastMinedVxDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastMinedVxDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastMinedVxDividendIdToStorage(db vm_db.VmDb, periodId uint64) error {
	return db.SetValue(lastMinedVxDividendIdKey, Uint64ToBytes(periodId))
}

func IsValidVxAmountBytesForDividend(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(VxDividendThreshold) >= 0
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func GetCurrentPeriodIdFromStorage(db vm_db.VmDb, reader util.ConsensusReader) uint64 {
	return reader.GetIndexByTime(GetTimestampInt64(db), 0)
}

func GetDonateFeeSum(db vm_db.VmDb, periodId uint64) *big.Int {
	if amountBytes, err := db.GetValue(GetDonateFeeSumKey(periodId)); err == nil && len(amountBytes) > 0 {
		return new(big.Int).SetBytes(amountBytes)
	} else {
		return big.NewInt(0)
	}
}

func AddDonateFeeSum(db vm_db.VmDb, reader util.ConsensusReader) error {
	period := GetCurrentPeriodIdFromStorage(db, reader)
	donateFeeSum := GetDonateFeeSum(db, period)
	return db.SetValue(GetDonateFeeSumKey(period), new(big.Int).Add(donateFeeSum, NewMarketFeeDonateAmount).Bytes())
}

func DeleteDonateFeeSum(db vm_db.VmDb, period uint64) error {
	return db.SetValue(GetDonateFeeSumKey(period), nil)
}

func GetDonateFeeSumKey(periodId uint64) []byte {
	return append(donateFeeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func FilterPendingNewMarkets(db vm_db.VmDb, tradeToken types.TokenTypeId) (quoteTokens [][]byte, err error) {
	pendingNewMarkets := &PendingNewMarkets{}
	if err = getFundValueFromDb(db, pendingNewMarketActionsKey, pendingNewMarkets); err != nil {
		return nil, err
	} else {
		for index, action := range pendingNewMarkets.PendingActions {
			if bytes.Equal(action.TradeToken, tradeToken.Bytes()) {
				quoteTokens = action.QuoteTokens
				if len(quoteTokens) == 0 {
					return nil, GetTokenInfoCallbackInnerConflictErr
				}
				actionsLen := len(pendingNewMarkets.PendingActions)
				if actionsLen > 1 {
					pendingNewMarkets.PendingActions[index] = pendingNewMarkets.PendingActions[actionsLen-1]
					pendingNewMarkets.PendingActions = pendingNewMarkets.PendingActions[:actionsLen-1]
					return quoteTokens, SavePendingNewMarkets(db, pendingNewMarkets)
				} else {
					return quoteTokens, db.SetValue(pendingNewMarketActionsKey, nil)
				}
			}
		}
		return nil, GetTokenInfoCallbackInnerConflictErr
	}
}

func AddToPendingNewMarkets(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) (err error) {
	pendingNewMarkets := &PendingNewMarkets{}
	if err = getFundValueFromDb(db, pendingNewMarketActionsKey, pendingNewMarkets); err == nil || err == NotFoundValueFromDb {
		var foundTradeToken bool
		for _, action := range pendingNewMarkets.PendingActions {
			if bytes.Equal(action.TradeToken, tradeToken.Bytes()) {
				for _, qt := range action.QuoteTokens {
					if bytes.Equal(qt, quoteToken.Bytes()) {
						return PendingNewMarketInnerConflictErr
					}
				}
				foundTradeToken = true
				action.QuoteTokens = append(action.QuoteTokens, quoteToken.Bytes())
				break
			}
		}
		if !foundTradeToken {
			quoteTokens := [][]byte{quoteToken.Bytes()}
			action := &dexproto.PendingNewMarketAction{TradeToken: tradeToken.Bytes(), QuoteTokens: quoteTokens}
			pendingNewMarkets.PendingActions = append(pendingNewMarkets.PendingActions, action)
		}
		return SavePendingNewMarkets(db, pendingNewMarkets)
	}
	return
}

func SavePendingNewMarkets(db vm_db.VmDb, pendingNewMarkets *PendingNewMarkets) error {
	if data, err := pendingNewMarkets.Serialize(); err != nil {
		return err
	} else {
		return db.SetValue(pendingNewMarketActionsKey, data)
	}
}

func GetPendingNewMarketFeeSum(db vm_db.VmDb) *big.Int {
	if amountBytes, err := db.GetValue(pendingNewMarketFeeSumKey); err == nil && len(amountBytes) > 0 {
		return new(big.Int).SetBytes(amountBytes)
	} else {
		return big.NewInt(0)
	}
}

func AddPendingNewMarketFeeSum(db vm_db.VmDb) error {
	return modifyPendingNewMarketFeeSum(db, true)
}

func SubPendingNewMarketFeeSum(db vm_db.VmDb) error {
	return modifyPendingNewMarketFeeSum(db, false)
}

func modifyPendingNewMarketFeeSum(db vm_db.VmDb, isAdd bool) error {
	var (
		originAmount = GetPendingNewMarketFeeSum(db)
		newAmount    *big.Int
	)
	if isAdd {
		newAmount = new(big.Int).Add(originAmount, NewMarketFeeAmount)
	} else {
		newAmount = new(big.Int).Sub(originAmount, NewMarketFeeAmount)
	}
	if newAmount.Sign() < 0 {
		return PendingDonateAmountSubExceedErr
	}
	if newAmount.Sign() == 0 {
		return db.SetValue(pendingNewMarketFeeSumKey, nil)
	} else {
		return db.SetValue(pendingNewMarketFeeSumKey, newAmount.Bytes())
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
		return nil, nil, nil, false
	}
}

func GetTokenInfo(db vm_db.VmDb, token types.TokenTypeId) (*TokenInfo, error) {
	tokenInfo := &TokenInfo{}
	if err := getFundValueFromDb(db, GetTokenInfoKey(token), tokenInfo); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return tokenInfo, err
	}
}

func SaveTokenInfo(db vm_db.VmDb, token types.TokenTypeId, tokenInfo *TokenInfo) error {
	return saveFunValueToDb(db, GetTokenInfoKey(token), tokenInfo)
}

func GetTokenInfoKey(token types.TokenTypeId) []byte {
	return append(tokenInfoPrefix, token.Bytes()...)
}

func NewAndSaveMarketId(db vm_db.VmDb) (int32, error) {
	var newId int32
	if idBytes, err := db.GetValue(marketIdKey); err == nil || err == NotFoundValueFromDb {
		if len(idBytes) == 0 {
			newId = 1
		} else {
			newId = int32(BytesToUint32(idBytes)) + 1
		}
		return newId, db.SetValue(marketIdKey, Uint32ToBytes(uint32(newId)))
	} else {
		return -1, err
	}
}

func GetMarketInfo(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) (*MarketInfo, error) {
	marketInfo := &MarketInfo{}
	if err := getFundValueFromDb(db, GetMarketInfoKey(tradeToken, quoteToken), marketInfo); err == NotFoundValueFromDb {
		return nil, TradeMarketNotExistsError
	} else {
		return marketInfo, err
	}
}

func SaveMarketInfo(db vm_db.VmDb, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId) error {
	return saveFunValueToDb(db, GetMarketInfoKey(tradeToken, quoteToken), marketInfo)
}

func DeleteMarketInfo(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) error {
	return db.SetValue(GetMarketInfoKey(tradeToken, quoteToken), nil)
}

func GetMarketInfoKey(tradeToken, quoteToken types.TokenTypeId) []byte {
	re := make([]byte, len(marketKeyPrefix)+2*types.TokenTypeIdSize)
	copy(re[:len(marketKeyPrefix)], marketKeyPrefix)
	copy(re[len(marketKeyPrefix):], tradeToken.Bytes())
	copy(re[len(marketKeyPrefix)+types.TokenTypeIdSize:], quoteToken.Bytes())
	return re
}

func AddNewMarketEventLog(db vm_db.VmDb, newMarketEvent *NewMarketEvent) {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, newMarketEvent.GetTopicId())
	log.Data = newMarketEvent.toDataBytes()
	db.AddLog(log)
}

func NewAndSaveOrderSerialNo(db vm_db.VmDb, timestamp int64) (int32, error) {
	orderIdSerialNo := &OrderIdSerialNo{}
	var newSerialNo int32
	if err := getFundValueFromDb(db, orderIdSerialNoKey, orderIdSerialNo); err == NotFoundValueFromDb {
		newSerialNo = 0
	} else if err != nil {
		return -1, err
	} else {
		if timestamp == orderIdSerialNo.Timestamp {
			newSerialNo = orderIdSerialNo.SerialNo + 1
		} else {
			newSerialNo = 0
		}
	}
	orderIdSerialNo.Timestamp = timestamp
	orderIdSerialNo.SerialNo = newSerialNo
	return newSerialNo, saveFunValueToDb(db, orderIdSerialNoKey, orderIdSerialNo)
}

func IsOwner(db vm_db.VmDb, address types.Address) bool {
	if storeOwner, err := db.GetValue(ownerKey); err == nil {
		if len(storeOwner) == types.AddressSize {
			if bytes.Equal(storeOwner, address.Bytes()) {
				return true
			}
		} else if len(storeOwner) == 0 {
			return true
		}
	}
	return false
}

func GetOwner(db vm_db.VmDb) (*types.Address, error) {
	if storeOwner, err := db.GetValue(ownerKey); err == nil && len(storeOwner) == types.AddressSize {
		if owner, err := types.BytesToAddress(storeOwner); err == nil {
			return &owner, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func SetOwner(db vm_db.VmDb, address types.Address) error {
	return db.SetValue(ownerKey, address.Bytes())
}

func ValidTimerAddress(db vm_db.VmDb, address types.Address) bool {
	if timerAddressBytes, err := db.GetValue(timerContractAddressKey); err == nil && bytes.Equal(timerAddressBytes, address.Bytes()) {
		return true
	}
	return false
}

func SetTimerAddress(db vm_db.VmDb, address types.Address) error {
	return db.SetValue(timerContractAddressKey, address.Bytes())
}

func GetPledgeForVx(db vm_db.VmDb, address types.Address) *big.Int {
	if bs, err := db.GetValue(GetPledgeForVxKey(address)); err == nil && len(bs) > 0 {
		return new(big.Int).SetBytes(bs)
	} else {
		return big.NewInt(0)
	}
}

func SavePledgeForVx(db vm_db.VmDb, address types.Address, amount *big.Int) error {
	return db.SetValue(GetPledgeForVxKey(address), amount.Bytes())
}

func DeletePledgeForVx(db vm_db.VmDb, address types.Address) error {
	return db.SetValue(GetPledgeForVxKey(address), nil)
}

func GetPledgeForVxKey(address types.Address) []byte {
	return append(pledgeForVxPrefix, address.Bytes()...)
}

func GetPledgeForVip(db vm_db.VmDb, address types.Address) (*PledgeVip, error) {
	pledgeVip := &PledgeVip{}
	if err := getFundValueFromDb(db, GetPledgeForVipKey(address), pledgeVip); err == NotFoundValueFromDb {
		return nil, nil
	} else {
		return pledgeVip, err
	}
}

func SavePledgeForVip(db vm_db.VmDb, address types.Address, pledgeVip *PledgeVip) error {
	return saveFunValueToDb(db, GetPledgeForVipKey(address), pledgeVip)
}

func DeletePledgeForVip(db vm_db.VmDb, address types.Address) error {
	return db.SetValue(GetPledgeForVipKey(address), nil)
}

func GetPledgeForVipKey(address types.Address) []byte {
	return append(pledgeForVipPrefix, address.Bytes()...)
}

func SetTimerTimestamp(db vm_db.VmDb, timestamp int64) error {
	if timestamp > GetTimerTimestamp(db) {
		return db.SetValue(timestampKey, Uint64ToBytes(uint64(timestamp)))
	} else {
		return InvalidTimestampFromTimerErr
	}
}

func GetTimerTimestamp(db vm_db.VmDb) int64 {
	if bs, err := db.GetValue(timestampKey); err == nil && len(bs) == 8 {
		return int64(BytesToUint64(bs))
	} else {
		return 0
	}
}

func GetTimestampInt64(db vm_db.VmDb) int64 {
	timestamp := GetTimerTimestamp(db)
	if timestamp == 0 {
		panic(NotFoundValueFromDb)
	}
	return timestamp
}

func getFundValueFromDb(db vm_db.VmDb, key []byte, fundSerializable FundSerializable) error {
	if data, err := db.GetValue(key); err == nil && len(data) > 0 {
		if err = fundSerializable.DeSerialize(data); err != nil {
			return err
		} else {
			return nil
		}
	} else {
		return NotFoundValueFromDb
	}
}

func saveFunValueToDb(db vm_db.VmDb, key []byte, fundSerializable FundSerializable) error {
	if userFeesBytes, err := fundSerializable.Serialize(); err == nil {
		return db.SetValue(key, userFeesBytes)
	} else {
		return err
	}
}
