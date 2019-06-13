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

	brokerFeeSumKeyPrefix = []byte("bf:") // brokerFeeSum:periodId, must

	pendingNewMarketFeeSumKey           = []byte("pnmfS:") // pending feeSum for new market
	pendingNewMarketActionsKey          = []byte("pmkas:")
	pendingSetQuoteActionsKey           = []byte("psqas:")
	pendingTransferTokenOwnerActionsKey = []byte("ptoas:")
	marketIdKey                         = []byte("mkId:")
	orderIdSerialNoKey                  = []byte("orIdSl:")

	timerContractAddressKey = []byte("tmA:")

	VxFundKeyPrefix            = []byte("vxF:")  // vxFund:types.Address
	vxSumFundsKey              = []byte("vxFS:") // vxFundSum
	lastFeeDividendPeriodIdKey = []byte("lFDPId:")
	lastMinedVxPeriodIdKey     = []byte("fMVPId:") //
	firstMinedVxPeriodIdKey    = []byte("fMVPId:")
	marketInfoKeyPrefix        = []byte("mk:") // market: tradeToke,quoteToken

	pledgeForVipPrefix = []byte("pldVip:") // pledgeForVip: types.Address
	pledgeForVxPrefix  = []byte("pldVx:")  // pledgeForVx: types.Address

	tokenInfoPrefix = []byte("tk:") // token:tokenId
	vxBalanceKey    = []byte("vxAmt:")

	codeByInviterPrefix    = []byte("itr2cd:")
	inviterByCodePrefix    = []byte("cd2itr:")
	inviterByInviteePrefix = []byte("ite2itr:")

	VxTokenBytes   = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	VxToken, _     = types.BytesToTokenTypeId(VxTokenBytes)
	commonTokenPow = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)
	VxInitAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(100000000))

	VxDividendThreshold        = new(big.Int).Mul(commonTokenPow, big.NewInt(10)) // 10
	VxMinedAmtFirstPeriod      = new(big.Int).Mul(commonTokenPow, big.NewInt(477210))
	NewMarketFeeAmount         = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	NewMarketFeeDividendAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))
	NewMarketFeeDonateAmount   = new(big.Int).Sub(NewMarketFeeAmount, NewMarketFeeDividendAmount)
	NewInviterFeeAmount        = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))

	PledgeForVxMinAmount       = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	PledgeForVipAmount         = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	PledgeForVipDuration int64 = 3600 * 24 * 30

	viteTokenInfo = dexproto.TokenInfo{TokenId: ledger.ViteTokenId.Bytes(), Decimals: 18, Symbol: "VITE", Index: 0, QuoteTokenType: ViteTokenType}

	viteMinAmount    = new(big.Int).Mul(commonTokenPow, big.NewInt(100)) // 100 VITE
	ethMinAmount     = new(big.Int).Div(commonTokenPow, big.NewInt(100)) // 0.01 ETH
	bitcoinMinAmount = big.NewInt(50000)  // 0.0005 BTC
	usdtMinAmount    = big.NewInt(100000000)   //1 USDT
	QuoteTokenExtras = map[int32]*QuoteTokenExtraInfo{
		ViteTokenType: &QuoteTokenExtraInfo{Decimals: 18, MinAmount: viteMinAmount},
		BtcTokenType:  &QuoteTokenExtraInfo{Decimals: 8, MinAmount: bitcoinMinAmount},
		EthTokenType:  &QuoteTokenExtraInfo{Decimals: 18, MinAmount: ethMinAmount},
		UsdTokenType:  &QuoteTokenExtraInfo{Decimals: 8, MinAmount: usdtMinAmount},
	}
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

const (
	OwnerConfigOwner         = 1
	OwnerConfigTimerAddress  = 2
	OwnerConfigMineMarket    = 4
	OwnerConfigNewQuoteToken = 8
)

const (
	MarketOwnerTransferOwner   = 1
	MarketOwnerConfigTakerRate = 2
	MarketOwnerConfigMakerRate = 4
	MarketOwnerStopMarket      = 8
)

const (
	ViteTokenType = iota + 1
	EthTokenType
	BtcTokenType
	UsdTokenType
)

const (
	GetTokenForNewMarket     = 1
	GetTokenForSetQuote      = 2
	GetTokenForTransferOwner = 3
)

type QuoteTokenExtraInfo struct {
	Decimals  int32
	MinAmount *big.Int
}

type ParamDexFundWithDraw struct {
	Token  types.TokenTypeId
	Amount *big.Int
}

type ParamDexFundNewOrder struct {
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
	Side       bool
	OrderType  uint8
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

type ParamDexFundPledgeForVx struct {
	ActionType uint8 // 1: pledge 2: cancel pledge
	Amount     *big.Int
}

type ParamDexFundPledgeForVip struct {
	ActionType uint8 // 1: pledge 2: cancel pledge
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
	Success       bool
}

type ParamDexFundGetTokenInfoCallback struct {
	TokenId     types.TokenTypeId
	Bid         uint8
	Exist       bool
	Decimals    uint8
	TokenSymbol string
	Index       uint16
	Owner       types.Address
}

type ParamDexFundOwnerConfig struct {
	OperationCode  uint8 // 1 owner, 2 timerAddress, 4 mineMarket
	Owner          types.Address
	TimerAddress   types.Address
	AllowMine      bool
	TradeToken     types.TokenTypeId // for new market
	QuoteToken     types.TokenTypeId // for new market
	NewQuoteToken  types.TokenTypeId // for set quote token
	QuoteTokenType uint8             // for set quote token
}

type ParamDexFundMarketOwnerConfig struct {
	OperationCode uint8 // 1 owner, 2 takerRate, 4 makerRate, 8 stopMarket
	TradeToken    types.TokenTypeId
	QuoteToken    types.TokenTypeId
	Owner         types.Address
	TakerRate     int32
	MakerRate     int32
	StopMarket    bool
}

type ParamDexFundTransferTokenOwner struct {
	Token types.TokenTypeId
	Owner types.Address
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

type BrokerFeeSumByPeriod struct {
	dexproto.BrokerFeeSumByPeriod
}

func (bfs *BrokerFeeSumByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&bfs.BrokerFeeSumByPeriod)
}

func (bfs *BrokerFeeSumByPeriod) DeSerialize(data []byte) (err error) {
	protoBrokerFeeSumByPeriod := dexproto.BrokerFeeSumByPeriod{}
	if err := proto.Unmarshal(data, &protoBrokerFeeSumByPeriod); err != nil {
		return err
	} else {
		bfs.BrokerFeeSumByPeriod = protoBrokerFeeSumByPeriod
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

type PendingSetQuotes struct {
	dexproto.PendingSetQuotes
}

func (psq *PendingSetQuotes) Serialize() (data []byte, err error) {
	return proto.Marshal(&psq.PendingSetQuotes)
}

func (psq *PendingSetQuotes) DeSerialize(data []byte) error {
	pendingSetQuotes := dexproto.PendingSetQuotes{}
	if err := proto.Unmarshal(data, &pendingSetQuotes); err != nil {
		return err
	} else {
		psq.PendingSetQuotes = pendingSetQuotes
		return nil
	}
}

type PendingTransferTokenOwners struct {
	dexproto.PendingTransferTokenOwners
}

func (psq *PendingTransferTokenOwners) Serialize() (data []byte, err error) {
	return proto.Marshal(&psq.PendingTransferTokenOwners)
}

func (psq *PendingTransferTokenOwners) DeSerialize(data []byte) error {
	pendingTransferTokenOwners := dexproto.PendingTransferTokenOwners{}
	if err := proto.Unmarshal(data, &pendingTransferTokenOwners); err != nil {
		return err
	} else {
		psq.PendingTransferTokenOwners = pendingTransferTokenOwners
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

func GetUserFund(db vm_db.VmDb, address types.Address) (dexFund *UserFund, ok bool) {
	dexFund = &UserFund{}
	ok = deserializeFromDb(db, GetUserFundKey(address), dexFund)
	return
}

func SaveUserFund(db vm_db.VmDb, address types.Address, dexFund *UserFund) {
	serializeToDb(db, GetUserFundKey(address), dexFund)
}

func BatchSaveUserFund(db vm_db.VmDb, address types.Address, funds map[types.TokenTypeId]*big.Int) error {
	userFund, _ := GetUserFund(db, address)
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
	SaveUserFund(db, address, userFund)
	return nil
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetCurrentFeeSum(db vm_db.VmDb, reader util.ConsensusReader) (*FeeSumByPeriod, bool) {
	return getFeeSumByKey(db, GetFeeSumCurrentKey(db, reader))
}

func GetFeeSumByPeriodId(db vm_db.VmDb, periodId uint64) (*FeeSumByPeriod, bool) {
	return getFeeSumByKey(db, GetFeeSumKeyByPeriodId(periodId))
}

func getFeeSumByKey(db vm_db.VmDb, feeKey []byte) (*FeeSumByPeriod, bool) {
	feeSum := &FeeSumByPeriod{}
	ok := deserializeFromDb(db, feeKey, feeSum)
	return feeSum, ok
}

//get all feeSums that not divided yet
func GetNotDividedFeeSumsByPeriodId(db vm_db.VmDb, periodId uint64) (map[uint64]*FeeSumByPeriod) {
	var (
		dexFeeSums    = make(map[uint64]*FeeSumByPeriod)
		dexFeeSum     *FeeSumByPeriod
		ok, everFound bool
	)
	for {
		if dexFeeSum, ok = GetFeeSumByPeriodId(db, periodId); !ok { // found first valid period
			if periodId > 0 && !everFound {
				periodId--
				continue
			} else { // lastValidPeriod is delete
				return dexFeeSums
			}
		} else {
			everFound = true
			if !dexFeeSum.FinishFeeDividend {
				dexFeeSums[periodId] = dexFeeSum
			} else {
				return dexFeeSums
			}
		}
		periodId = dexFeeSum.LastValidPeriod
		if periodId == 0 {
			return dexFeeSums
		}
	}
}

func SaveCurrentFeeSum(db vm_db.VmDb, reader util.ConsensusReader, feeSum *FeeSumByPeriod) {
	feeSumKey := GetFeeSumCurrentKey(db, reader)
	serializeToDb(db, feeSumKey, feeSum)
}

//fee sum used both by fee dividend and mined vx dividend
func MarkFeeSumAsFeeDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.FinishVxMine {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FinishFeeDividend = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func MarkFeeSumAsMinedVxDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.FinishFeeDividend {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FinishVxMine = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func GetFeeSumKeyByPeriodId(periodId uint64) []byte {
	return append(feeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetFeeSumCurrentKey(db vm_db.VmDb, reader util.ConsensusReader) []byte {
	return GetFeeSumKeyByPeriodId(GetCurrentPeriodId(db, reader))
}

func GetFeeSumLastPeriodIdForRoll(db vm_db.VmDb) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, lastFeeSumPeriodKey); len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeSumLastPeriodIdForRoll(db vm_db.VmDb, reader util.ConsensusReader) {
	periodId := GetCurrentPeriodId(db, reader)
	setValueToDb(db, lastFeeSumPeriodKey, Uint64ToBytes(periodId))
}

func SaveCurrentBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, broker []byte, brokerFeeSum *BrokerFeeSumByPeriod) {
	serializeToDb(db, GetCurrentBrokerFeeSumKey(db, reader, broker), brokerFeeSum)
}

func GetCurrentBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, broker []byte) (*BrokerFeeSumByPeriod, bool) {
	currentBrokerFeeSumKey := GetCurrentBrokerFeeSumKey(db, reader, broker)
	return getBrokerFeeSumByKey(db, currentBrokerFeeSumKey)
}

func GetBrokerFeeSumByPeriodId(db vm_db.VmDb, periodId uint64, broker []byte) (*BrokerFeeSumByPeriod, bool) {
	return getBrokerFeeSumByKey(db, GetBrokerFeeSumKeyByPeriodIdAndAddress(periodId, broker))
}

func GetCurrentBrokerFeeSumKey(db vm_db.VmDb, reader util.ConsensusReader, broker []byte) []byte {
	return GetBrokerFeeSumKeyByPeriodIdAndAddress(GetCurrentPeriodId(db, reader), broker)
}

func DeleteBrokerFeeSumByKey(db vm_db.VmDb, key []byte) {
	setValueToDb(db, key, nil)
}

func getBrokerFeeSumByKey(db vm_db.VmDb, feeKey []byte) (*BrokerFeeSumByPeriod, bool) {
	brokerFeeSum := &BrokerFeeSumByPeriod{}
	ok := deserializeFromDb(db, feeKey, brokerFeeSum)
	return brokerFeeSum, ok
}

func GetBrokerFeeSumKeyByPeriodIdAndAddress(periodId uint64, address []byte) []byte {
	return append(append(brokerFeeSumKeyPrefix, Uint64ToBytes(periodId)...), address...)
}

func GetUserFees(db vm_db.VmDb, address []byte) (userFees *UserFees, ok bool) {
	userFees = &UserFees{}
	ok = deserializeFromDb(db, GetUserFeesKey(address), userFees)
	return
}

func SaveUserFees(db vm_db.VmDb, address []byte, userFees *UserFees) {
	serializeToDb(db, GetUserFeesKey(address), userFees)
}

func DeleteUserFees(db vm_db.VmDb, address []byte) {
	setValueToDb(db, GetUserFeesKey(address), nil)
}

func GetUserFeesKey(address []byte) []byte {
	return append(UserFeeKeyPrefix, address...)
}

func GetVxFundsFrom(db vm_db.VmDb, address []byte) (vxFunds *VxFunds, ok bool) {
	vxFunds = &VxFunds{}
	ok = deserializeFromDb(db, GetVxFundsKey(address), vxFunds)
	return
}

func SaveVxFunds(db vm_db.VmDb, address []byte, vxFunds *VxFunds) {
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

func DeleteVxFunds(db vm_db.VmDb, address []byte) {
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

func GetLastFeeDividendPeriodId(db vm_db.VmDb) uint64 {
	if lastFeeDividendPeriodIdBytes := getValueFromDb(db, lastFeeDividendPeriodIdKey); len(lastFeeDividendPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastFeeDividendPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveLastFeeDividendPeriodId(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastFeeDividendPeriodIdKey, Uint64ToBytes(periodId))
}

func GetFirstMinedVxPeriodId(db vm_db.VmDb) uint64 {
	if firstMinedVxPeriodIdBytes := getValueFromDb(db, firstMinedVxPeriodIdKey); len(firstMinedVxPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(firstMinedVxPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFirstMinedVxPeriodId(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, firstMinedVxPeriodIdKey, Uint64ToBytes(periodId))
}

func GetLastMinedVxPeriodId(db vm_db.VmDb) uint64 {
	if lastMinedVxPeriodIdBytes := getValueFromDb(db, lastMinedVxPeriodIdKey); len(lastMinedVxPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastMinedVxPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveLastMinedVxPeriodId(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastMinedVxPeriodIdKey, Uint64ToBytes(periodId))
}

func IsValidVxAmountBytesForDividend(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(VxDividendThreshold) >= 0
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func GetCurrentPeriodId(db vm_db.VmDb, reader util.ConsensusReader) uint64 {
	return reader.GetIndexByTime(GetTimestampInt64(db), 0)
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
		action := &dexproto.NewMarketAction{TradeToken: tradeToken.Bytes(), QuoteTokens: quoteTokens}
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

//handle case on duplicate callback for getTokenInfo
func FilterPendingSetQuotes(db vm_db.VmDb, token types.TokenTypeId) (action *dexproto.SetQuoteAction, err error) {
	pendingSetQuotes := &PendingSetQuotes{}
	deserializeFromDb(db, pendingSetQuoteActionsKey, pendingSetQuotes)
	for index, action := range pendingSetQuotes.PendingActions {
		if bytes.Equal(action.Token, token.Bytes()) {
			actionsLen := len(pendingSetQuotes.PendingActions)
			if actionsLen > 1 {
				pendingSetQuotes.PendingActions[index] = pendingSetQuotes.PendingActions[actionsLen-1]
				pendingSetQuotes.PendingActions = pendingSetQuotes.PendingActions[:actionsLen-1]
				SavePendingSetQuotes(db, pendingSetQuotes)
				return action, nil
			} else {
				setValueToDb(db, pendingSetQuoteActionsKey, nil)
				return action, nil
			}
		}
	}
	return nil, GetTokenInfoCallbackInnerConflictErr
}

func AddToPendingSetQuotes(db vm_db.VmDb, token types.TokenTypeId, quoteType uint8) {
	pendingSetQuotes := &PendingSetQuotes{}
	deserializeFromDb(db, pendingSetQuoteActionsKey, pendingSetQuotes)
	action := &dexproto.SetQuoteAction{}
	action.Token = token.Bytes()
	action.QuoteType = int32(quoteType)
	pendingSetQuotes.PendingActions = append(pendingSetQuotes.PendingActions, action)
	SavePendingSetQuotes(db, pendingSetQuotes)
}

func SavePendingSetQuotes(db vm_db.VmDb, pendingSetQuotes *PendingSetQuotes) {
	serializeToDb(db, pendingSetQuoteActionsKey, pendingSetQuotes)
}

//handle case on duplicate callback for getTokenInfo
func FilterPendingTransferTokenOwners(db vm_db.VmDb, token types.TokenTypeId) (action *dexproto.TransferTokenOwnerAction, err error) {
	pendings := &PendingTransferTokenOwners{}
	deserializeFromDb(db, pendingTransferTokenOwnerActionsKey, pendings)
	for index, action := range pendings.PendingActions {
		if bytes.Equal(action.Token, token.Bytes()) {
			actionsLen := len(pendings.PendingActions)
			if actionsLen > 1 {
				pendings.PendingActions[index] = pendings.PendingActions[actionsLen-1]
				pendings.PendingActions = pendings.PendingActions[:actionsLen-1]
				SavePendingTransferTokenOwners(db, pendings)
				return action, nil
			} else {
				setValueToDb(db, pendingTransferTokenOwnerActionsKey, nil)
				return action, nil
			}
		}
	}
	return nil, GetTokenInfoCallbackInnerConflictErr
}

func AddToPendingTransferTokenOwners(db vm_db.VmDb, token types.TokenTypeId, origin, new types.Address) {
	pendings := &PendingTransferTokenOwners{}
	deserializeFromDb(db, pendingTransferTokenOwnerActionsKey, pendings)
	action := &dexproto.TransferTokenOwnerAction{}
	action.Token = token.Bytes()
	action.Origin = origin.Bytes()
	action.New = new.Bytes()
	pendings.PendingActions = append(pendings.PendingActions, action)
	SavePendingTransferTokenOwners(db, pendings)
}

func SavePendingTransferTokenOwners(db vm_db.VmDb, pendings *PendingTransferTokenOwners) {
	serializeToDb(db, pendingTransferTokenOwnerActionsKey, pendings)
}

func GetTokenInfo(db vm_db.VmDb, token types.TokenTypeId) (tokenInfo *TokenInfo, ok bool) {
	tokenInfo = &TokenInfo{}
	if token == ledger.ViteTokenId {
		tokenInfo.TokenInfo = viteTokenInfo
		return tokenInfo, true
	}
	ok = deserializeFromDb(db, GetTokenInfoKey(token), tokenInfo)
	return
}

func SaveTokenInfo(db vm_db.VmDb, token types.TokenTypeId, tokenInfo *TokenInfo) {
	serializeToDb(db, GetTokenInfoKey(token), tokenInfo)
}

func AddTokenEventLog(db vm_db.VmDb, tokenInfo *TokenInfo) {
	log := &ledger.VmLog{}
	event := TokenEvent{}
	event.TokenInfo = tokenInfo.TokenInfo
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
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

func GetMarketInfoByTokens(db vm_db.VmDb, tradeTokenData, quoteTokenData []byte) (marketInfo *MarketInfo, ok bool) {
	if tradeToken, err := types.BytesToTokenTypeId(tradeTokenData); err != nil {
		panic(InvalidTokenErr)
	} else if quoteToken, err := types.BytesToTokenTypeId(quoteTokenData); err != nil {
		panic(InvalidTokenErr)
	} else {
		return GetMarketInfo(db, tradeToken, quoteToken)
	}
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

func AddMarketEventLog(db vm_db.VmDb, marketInfo *MarketInfo) {
	log := &ledger.VmLog{}
	event := MarketEvent{}
	event.MarketInfo = marketInfo.MarketInfo
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}

func GetMarketInfoKey(tradeToken, quoteToken types.TokenTypeId) []byte {
	re := make([]byte, len(marketInfoKeyPrefix)+2*types.TokenTypeIdSize)
	copy(re[:len(marketInfoKeyPrefix)], marketInfoKeyPrefix)
	copy(re[len(marketInfoKeyPrefix):], tradeToken.Bytes())
	copy(re[len(marketInfoKeyPrefix)+types.TokenTypeIdSize:], quoteToken.Bytes())
	return re
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

func GetCodeByInviter(db vm_db.VmDb, address types.Address) uint32 {
	if bs := getValueFromDb(db, append(codeByInviterPrefix, address.Bytes()...)); len(bs) == 4 {
		return BytesToUint32(bs)
	} else {
		return 0
	}
}

func SaveCodeByInviter(db vm_db.VmDb, address types.Address, inviteCode uint32) {
	setValueToDb(db, append(codeByInviterPrefix, address.Bytes()...), Uint32ToBytes(inviteCode))
}

func GetInviterByCode(db vm_db.VmDb, inviteCode uint32) (inviter *types.Address, err error) {
	if bs := getValueFromDb(db, append(inviterByCodePrefix, Uint32ToBytes(inviteCode)...)); len(bs) == types.AddressSize {
		*inviter, err = types.BytesToAddress(bs)
		return
	} else {
		return nil, InvalidInviteCodeErr
	}
}

func SaveInviterByCode(db vm_db.VmDb, address types.Address, inviteCode uint32) {
	setValueToDb(db, append(inviterByCodePrefix, Uint32ToBytes(inviteCode)...), address.Bytes())
}

func SaveInviterByInvitee(db vm_db.VmDb, invitee, inviter types.Address) {
	setValueToDb(db, append(inviterByInviteePrefix, invitee.Bytes()...), inviter.Bytes())
}

func GetInviterByInvitee(db vm_db.VmDb, address types.Address) (inviter *types.Address, err error) {
	if bs := getValueFromDb(db, append(inviterByInviteePrefix, address.Bytes()...)); len(bs) == types.AddressSize {
		*inviter, err = types.BytesToAddress(bs)
		return
	} else {
		return nil, NotBindInviterErr
	}
}

func NewInviteCode(db vm_db.VmDb, hash types.Hash) uint32 {
	var (
		codeBytes  = []byte{0, 0, 0, 0}
		inviteCode uint32
		ok         bool
		err        error
	)
	for i := 1; i < 250; i++ {
		if codeBytes, ok = randomBytesFromBytes(hash.Bytes(), codeBytes, 0, 32); !ok {
			return 0
		}
		if inviteCode = BytesToUint32(codeBytes); inviteCode == 0 {
			continue
		}
		if _, err = GetInviterByCode(db, inviteCode); err == InvalidInviteCodeErr {
			return inviteCode
		}
	}
	return 0
}

func GetVxBalance(db vm_db.VmDb) *big.Int {
	if data := getValueFromDb(db, vxBalanceKey); len(data) > 0 {
		return new(big.Int).SetBytes(data)
	} else {
		return VxInitAmount
	}
}

func SaveVxBalance(db vm_db.VmDb, amount *big.Int) {
	setValueToDb(db, vxBalanceKey, amount.Bytes())
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
