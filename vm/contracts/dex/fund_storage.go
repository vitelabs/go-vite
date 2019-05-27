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
		ledger.ViteTokenId: &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "VITE", Index: 0}},
		bitcoinToken:       &TokenInfo{dexproto.TokenInfo{Decimals: 8, Symbol: "BTC", Index: 0}},
		ethToken:           &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "ETH", Index: 0}},
		usdtToken:          &TokenInfo{dexproto.TokenInfo{Decimals: 18, Symbol: "USDT", Index: 0}},
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

type ParamDexFundPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Bid           uint8
}

type ParamDexFundCancelPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Amount        *big.Int
	Bid           uint8
}

type ParamDexFundPledgeCallBack struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Amount        *big.Int
	Bid           uint8
	Success bool
}

type ParamDexFundGetTokenInfoCallback struct {
	TokenId types.TokenTypeId
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

type SerializableDex interface {
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

func GetUserFundFromStorage(db vm_db.VmDb, address types.Address) (dexFund *UserFund, ok bool) {
	dexFund = &UserFund{}
	ok = deserializeFromDb(db, GetUserFundKey(address), dexFund)
	return
}

func SaveUserFundToStorage(db vm_db.VmDb, address types.Address, dexFund *UserFund) {
	serializeToDb(db, GetUserFundKey(address), dexFund)
}

func BatchSaveUserFund(db vm_db.VmDb, address types.Address, funds map[types.TokenTypeId]*big.Int) error {
	userFund, _ := GetUserFundFromStorage(db, address)
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
	SaveUserFundToStorage(db, address, userFund)
	return nil
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetCurrentFeeSumFromStorage(db vm_db.VmDb, reader util.ConsensusReader) (*FeeSumByPeriod, bool) {
	return getFeeSumByKeyFromStorage(db, GetFeeSumCurrentKeyFromStorage(db, reader))
}

func GetFeeSumByPeriodIdFromStorage(db vm_db.VmDb, periodId uint64) (*FeeSumByPeriod, bool) {
	return getFeeSumByKeyFromStorage(db, GetFeeSumKeyByPeriodId(periodId))
}

func getFeeSumByKeyFromStorage(db vm_db.VmDb, feeKey []byte) (*FeeSumByPeriod, bool) {
	feeSum := &FeeSumByPeriod{}
	ok := deserializeFromDb(db, feeKey, feeSum)
	return feeSum, ok
}

//get all feeSums that not divided yet
func GetNotDividedFeeSumsByPeriodIdFromStorage(db vm_db.VmDb, periodId uint64) (map[uint64]*FeeSumByPeriod, map[uint64]*big.Int) {
	var (
		dexFeeSums    = make(map[uint64]*FeeSumByPeriod)
		dexFeeSum     *FeeSumByPeriod
		donateFeeSums = make(map[uint64]*big.Int)
		ok            bool
	)
	for {
		if dexFeeSum, ok = GetFeeSumByPeriodIdFromStorage(db, periodId); !ok {
			if periodId > 0 {
				periodId--
				continue
			} else {
				return nil, nil
			}
		} else {
			if !dexFeeSum.FeeDivided {
				dexFeeSums[periodId] = dexFeeSum
				if donateFeeSum := GetDonateFeeSum(db, periodId); donateFeeSum.Sign() > 0 { // when donateFee exists feeSum must exists
					donateFeeSums[periodId] = donateFeeSum
				}
			} else {
				return dexFeeSums, donateFeeSums
			}
		}
		periodId = dexFeeSum.LastValidPeriod
		if periodId == 0 {
			return dexFeeSums, donateFeeSums
		}
	}
}

func SaveCurrentFeeSumToStorage(db vm_db.VmDb, reader util.ConsensusReader, feeSum *FeeSumByPeriod) {
	feeSumKey := GetFeeSumCurrentKeyFromStorage(db, reader)
	serializeToDb(db, feeSumKey, feeSum)
}

//fee sum used both by fee dividend and mined vx dividend
func MarkFeeSumAsFeeDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.MinedVxDivided {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FeeDivided = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func MarkFeeSumAsMinedVxDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.FeeDivided {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.MinedVxDivided = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func GetFeeSumKeyByPeriodId(periodId uint64) []byte {
	return append(feeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetFeeSumCurrentKeyFromStorage(db vm_db.VmDb, reader util.ConsensusReader) []byte {
	return GetFeeSumKeyByPeriodId(GetCurrentPeriodIdFromStorage(db, reader))
}

func GetFeeSumLastPeriodIdForRoll(db vm_db.VmDb) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, lastFeeSumPeriodKey); len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeSumLastPeriodIdForRoll(db vm_db.VmDb, reader util.ConsensusReader) {
	periodId := GetCurrentPeriodIdFromStorage(db, reader)
	setValueToDb(db, lastFeeSumPeriodKey, Uint64ToBytes(periodId))
}

func GetUserFeesFromStorage(db vm_db.VmDb, address []byte) (userFees *UserFees, ok bool) {
	userFees = &UserFees{}
	ok = deserializeFromDb(db, GetUserFeesKey(address), userFees)
	return
}

func SaveUserFeesToStorage(db vm_db.VmDb, address []byte, userFees *UserFees) {
	serializeToDb(db, GetUserFeesKey(address), userFees)
}

func DeleteUserFeesFromStorage(db vm_db.VmDb, address []byte) {
	setValueToDb(db, GetUserFeesKey(address), nil)
}

func GetUserFeesKey(address []byte) []byte {
	return append(UserFeeKeyPrefix, address...)
}

func GetVxFundsFromStorage(db vm_db.VmDb, address []byte) (vxFunds *VxFunds, ok bool) {
	vxFunds = &VxFunds{}
	ok = deserializeFromDb(db, GetVxFundsKey(address), vxFunds)
	return
}

func SaveVxFundsToStorage(db vm_db.VmDb, address []byte, vxFunds *VxFunds) {
	serializeToDb(db, GetVxFundsKey(address), vxFunds)
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

func DeleteVxFundsFromStorage(db vm_db.VmDb, address []byte) {
	setValueToDb(db, GetVxFundsKey(address), nil)
}

func GetVxFundsKey(address []byte) []byte {
	return append(VxFundKeyPrefix, address...)
}

func GetVxSumFundsFromDb(db vm_db.VmDb) (vxSumFunds *VxFunds, ok bool) {
	vxSumFunds = &VxFunds{}
	ok = deserializeFromDb(db, vxSumFundsKey, vxSumFunds)
	return
}

func SaveVxSumFundsToDb(db vm_db.VmDb, vxSumFunds *VxFunds) {
	serializeToDb(db, vxSumFundsKey, vxSumFunds)
}

func GetLastFeeDividendIdFromStorage(db vm_db.VmDb) uint64 {
	if lastFeeDividendIdBytes := getValueFromDb(db, lastFeeDividendIdKey); len(lastFeeDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastFeeDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastFeeDividendIdToStorage(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastFeeDividendIdKey, Uint64ToBytes(periodId))
}

func GetLastMinedVxDividendIdFromStorage(db vm_db.VmDb) uint64 {
	if lastMinedVxDividendIdBytes := getValueFromDb(db, lastMinedVxDividendIdKey); len(lastMinedVxDividendIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastMinedVxDividendIdBytes)
	} else {
		return 0
	}
}

func SaveLastMinedVxDividendIdToStorage(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastMinedVxDividendIdKey, Uint64ToBytes(periodId))
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
	if amountBytes := getValueFromDb(db, GetDonateFeeSumKey(periodId)); len(amountBytes) > 0 {
		return new(big.Int).SetBytes(amountBytes)
	} else {
		return big.NewInt(0)
	}
}

func AddDonateFeeSum(db vm_db.VmDb, reader util.ConsensusReader) {
	period := GetCurrentPeriodIdFromStorage(db, reader)
	donateFeeSum := GetDonateFeeSum(db, period)
	setValueToDb(db, GetDonateFeeSumKey(period), new(big.Int).Add(donateFeeSum, NewMarketFeeDonateAmount).Bytes())
}

func DeleteDonateFeeSum(db vm_db.VmDb, period uint64) {
	setValueToDb(db, GetDonateFeeSumKey(period), nil)
}

func GetDonateFeeSumKey(periodId uint64) []byte {
	return append(donateFeeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

//handle case on duplicate callback for getTokenInfo
func FilterPendingNewMarkets(db vm_db.VmDb, tradeToken types.TokenTypeId) (quoteTokens [][]byte, err error) {
	pendingNewMarkets := &PendingNewMarkets{}
	deserializeFromDb(db, pendingNewMarketActionsKey, pendingNewMarkets)
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
				SavePendingNewMarkets(db, pendingNewMarkets)
				return quoteTokens, nil
			} else {
				setValueToDb(db, pendingNewMarketActionsKey, nil)
				return quoteTokens, nil
			}
		}
	}
	return nil, GetTokenInfoCallbackInnerConflictErr
}

func AddToPendingNewMarkets(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) {
	pendingNewMarkets := &PendingNewMarkets{}
	deserializeFromDb(db, pendingNewMarketActionsKey, pendingNewMarkets)
	var foundTradeToken bool
	for _, action := range pendingNewMarkets.PendingActions {
		if bytes.Equal(action.TradeToken, tradeToken.Bytes()) {
			for _, qt := range action.QuoteTokens {
				if bytes.Equal(qt, quoteToken.Bytes()) {
					panic(PendingNewMarketInnerConflictErr)
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
	SavePendingNewMarkets(db, pendingNewMarkets)
}

func SavePendingNewMarkets(db vm_db.VmDb, pendingNewMarkets *PendingNewMarkets) {
	serializeToDb(db, pendingNewMarketActionsKey, pendingNewMarkets)
}

func GetPendingNewMarketFeeSum(db vm_db.VmDb) *big.Int {
	if amountBytes := getValueFromDb(db, pendingNewMarketFeeSumKey); len(amountBytes) > 0 {
		return new(big.Int).SetBytes(amountBytes)
	} else {
		return big.NewInt(0)
	}
}

func AddPendingNewMarketFeeSum(db vm_db.VmDb) {
	modifyPendingNewMarketFeeSum(db, true)
}

func SubPendingNewMarketFeeSum(db vm_db.VmDb) {
	modifyPendingNewMarketFeeSum(db, false)
}

func modifyPendingNewMarketFeeSum(db vm_db.VmDb, isAdd bool) {
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
		panic(PendingDonateAmountSubExceedErr)
	}
	if newAmount.Sign() == 0 {
		setValueToDb(db, pendingNewMarketFeeSumKey, nil)
	} else {
		setValueToDb(db, pendingNewMarketFeeSumKey, newAmount.Bytes())
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

func GetTokenInfo(db vm_db.VmDb, token types.TokenTypeId) (tokenInfo *TokenInfo, ok bool) {
	if tokenInfo, ok = QuoteTokenInfos[token]; ok {
		return
	} else {
		tokenInfo = &TokenInfo{}
		ok = deserializeFromDb(db, GetTokenInfoKey(token), tokenInfo)
		return
	}
}

func SaveTokenInfo(db vm_db.VmDb, token types.TokenTypeId, tokenInfo *TokenInfo) {
	serializeToDb(db, GetTokenInfoKey(token), tokenInfo)
}

func GetTokenInfoKey(token types.TokenTypeId) []byte {
	return append(tokenInfoPrefix, token.Bytes()...)
}

func NewAndSaveMarketId(db vm_db.VmDb) (newId int32) {
	if idBytes := getValueFromDb(db, marketIdKey); len(idBytes) == 0 {
		newId = 1
	} else {
		newId = int32(BytesToUint32(idBytes)) + 1
	}
	setValueToDb(db, marketIdKey, Uint32ToBytes(uint32(newId)))
	return
}

func GetMarketInfo(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) (marketInfo *MarketInfo, ok bool) {
	marketInfo = &MarketInfo{}
	ok = deserializeFromDb(db, GetMarketInfoKey(tradeToken, quoteToken), marketInfo)
	return
}

func SaveMarketInfo(db vm_db.VmDb, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId) {
	serializeToDb(db, GetMarketInfoKey(tradeToken, quoteToken), marketInfo)
}

func DeleteMarketInfo(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) {
	setValueToDb(db, GetMarketInfoKey(tradeToken, quoteToken), nil)
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

func NewAndSaveOrderSerialNo(db vm_db.VmDb, timestamp int64) (newSerialNo int32) {
	orderIdSerialNo := &OrderIdSerialNo{}
	if ok := deserializeFromDb(db, orderIdSerialNoKey, orderIdSerialNo); !ok {
		newSerialNo = 0
	} else {
		if timestamp == orderIdSerialNo.Timestamp {
			newSerialNo = orderIdSerialNo.SerialNo + 1
		} else {
			newSerialNo = 0
		}
	}
	orderIdSerialNo.Timestamp = timestamp
	orderIdSerialNo.SerialNo = newSerialNo
	serializeToDb(db, orderIdSerialNoKey, orderIdSerialNo)
	return
}

func IsOwner(db vm_db.VmDb, address types.Address) bool {
	if storeOwner := getValueFromDb(db, ownerKey); len(storeOwner) == types.AddressSize {
		return bytes.Equal(storeOwner, address.Bytes())
	} else {
		return len(storeOwner) == 0
	}
}

func GetOwner(db vm_db.VmDb) (*types.Address, error) {
	if storeOwner := getValueFromDb(db, ownerKey); len(storeOwner) == types.AddressSize {
		if owner, err := types.BytesToAddress(storeOwner); err == nil {
			return &owner, nil
		} else {
			return nil, err
		}
	} else {
		return nil, nil
	}
}

func SetOwner(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, ownerKey, address.Bytes())
}

func ValidTimerAddress(db vm_db.VmDb, address types.Address) bool {
	if timerAddressBytes := getValueFromDb(db, timerContractAddressKey); bytes.Equal(timerAddressBytes, address.Bytes()) {
		return true
	}
	return false
}

func SetTimerAddress(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, timerContractAddressKey, address.Bytes())
}

func GetPledgeForVx(db vm_db.VmDb, address types.Address) *big.Int {
	if bs := getValueFromDb(db, GetPledgeForVxKey(address)); len(bs) > 0 {
		return new(big.Int).SetBytes(bs)
	} else {
		return big.NewInt(0)
	}
}

func SavePledgeForVx(db vm_db.VmDb, address types.Address, amount *big.Int) {
	setValueToDb(db, GetPledgeForVxKey(address), amount.Bytes())
}

func DeletePledgeForVx(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetPledgeForVxKey(address), nil)
}

func GetPledgeForVxKey(address types.Address) []byte {
	return append(pledgeForVxPrefix, address.Bytes()...)
}

func GetPledgeForVip(db vm_db.VmDb, address types.Address) (pledgeVip *PledgeVip, ok bool) {
	pledgeVip = &PledgeVip{}
	ok = deserializeFromDb(db, GetPledgeForVipKey(address), pledgeVip)
	return
}

func SavePledgeForVip(db vm_db.VmDb, address types.Address, pledgeVip *PledgeVip) {
	serializeToDb(db, GetPledgeForVipKey(address), pledgeVip)
}

func DeletePledgeForVip(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetPledgeForVipKey(address), nil)
}

func GetPledgeForVipKey(address types.Address) []byte {
	return append(pledgeForVipPrefix, address.Bytes()...)
}

func GetTimestampInt64(db vm_db.VmDb) int64 {
	timestamp := GetTimerTimestamp(db)
	if timestamp == 0 {
		panic(NotSetTimestampErr)
	} else {
		return timestamp
	}
}

func SetTimerTimestamp(db vm_db.VmDb, timestamp int64) error {
	if timestamp > GetTimerTimestamp(db) {
		setValueToDb(db, timestampKey, Uint64ToBytes(uint64(timestamp)))
		return nil
	} else {
		return InvalidTimestampFromTimerErr
	}
}

func GetTimerTimestamp(db vm_db.VmDb) int64 {
	if bs := getValueFromDb(db, timestampKey); len(bs) == 8 {
		return int64(BytesToUint64(bs))
	} else {
		return 0
	}
}

func deserializeFromDb(db vm_db.VmDb, key []byte, serializable SerializableDex) bool {
	if data := getValueFromDb(db, key); len(data) > 0 {
		if err := serializable.DeSerialize(data); err != nil {
			panic(err)
		}
		return true
	} else {
		return false
	}
}

func serializeToDb(db vm_db.VmDb, key []byte, serializable SerializableDex) {
	if data, err := serializable.Serialize(); err != nil {
		panic(err)
	} else {
		setValueToDb(db, key, data)
	}
}
