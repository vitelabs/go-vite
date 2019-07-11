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
	ownerKey                = []byte("own:")
	viteXStoppedKey         = []byte("dexStp:")
	fundKeyPrefix           = []byte("fd:") // fund:types.Address
	minTradeAmountKeyPrefix = []byte("mTrAt:")
	mineThresholdKeyPrefix  = []byte("mTh:")

	timestampKey = []byte("tts:") // timerTimestamp

	userFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	feeSumKeyPrefix     = []byte("fS:")     // feeSum:periodId
	lastFeeSumPeriodKey = []byte("lFSPId:") //

	brokerFeeSumKeyPrefix = []byte("bf:") // brokerFeeSum:periodId, must

	pendingNewMarketActionsKey          = []byte("pmkas:")
	pendingSetQuoteActionsKey           = []byte("psqas:")
	pendingTransferTokenOwnerActionsKey = []byte("ptoas:")
	marketIdKey                         = []byte("mkId:")
	orderIdSerialNoKey                  = []byte("orIdSl:")

	timerAddressKey   = []byte("tmA:")
	triggerAddressKey = []byte("tgA:")

	VxFundKeyPrefix = []byte("vxF:")  // vxFund:types.Address
	vxSumFundsKey   = []byte("vxFS:") // vxFundSum

	lastJobPeriodIdWithBizTypeKey = []byte("ljpBId:")
	firstMinedVxPeriodIdKey       = []byte("fMVPId:")
	marketInfoKeyPrefix           = []byte("mk:") // market: tradeToke,quoteToken

	pledgeForVipKeyPrefix = []byte("pldVip:") // pledgeForVip: types.Address

	pledgesForVxKeyPrefix = []byte("pldsVx:") // pledgesForVx: types.Address
	pledgesForVxSumKey    = []byte("pldsVxS:")
	pledgeForVxKeyPrefix  = []byte("pldVx:") // pledgeForVx: types.Address

	tokenInfoKeyPrefix = []byte("tk:") // token:tokenId
	vxMinePoolKey      = []byte("vxmPl:")

	codeByInviterKeyPrefix    = []byte("itr2cd:")
	inviterByCodeKeyPrefix    = []byte("cd2itr:")
	inviterByInviteeKeyPrefix = []byte("ite2itr:")

	maintainerKey                   = []byte("mtA:")
	makerMineProxyKey               = []byte("mmpA:")
	makerMineProxyAmountByPeriodKey = []byte("mmpaP:")

	commonTokenPow = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)

	VxTokenId, _          = types.HexToTokenTypeId("tti_340b335ce06aa2a0a6db3c0a")
	VxMinedAmtFirstPeriod = new(big.Int).Mul(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(13)), nil), big.NewInt(47703236213)) // 477032.36213

	VxDividendThreshold      = new(big.Int).Mul(commonTokenPow, big.NewInt(10))
	NewMarketFeeAmount       = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	NewMarketFeeMineAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))
	NewMarketFeeDonateAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(4000))
	NewMarketFeeBurnAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(5000))
	NewInviterFeeAmount      = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))

	PledgeForVxMinAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	PledgeForVipAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	PledgeForVxThreshold = new(big.Int).Mul(commonTokenPow, big.NewInt(134))

	viteMinAmount    = new(big.Int).Mul(commonTokenPow, big.NewInt(100)) // 100 VITE
	ethMinAmount     = new(big.Int).Div(commonTokenPow, big.NewInt(100)) // 0.01 ETH
	bitcoinMinAmount = big.NewInt(50000)                                 // 0.0005 BTC
	usdMinAmount     = big.NewInt(100000000)                             //1 USD

	viteMineThreshold    = new(big.Int).Mul(commonTokenPow, big.NewInt(2))    // 2 VITE
	ethMineThreshold     = new(big.Int).Div(commonTokenPow, big.NewInt(5000)) // 0.0002 ETH
	bitcoinMineThreshold = big.NewInt(1000)                                   // 0.00001 BTC
	usdMineThreshold     = big.NewInt(2000000)                                // 0.1USD

	RateSumForFeeMine                = "0.6"                                             // 15% * 4
	RateForPledgeMine                = "0.2"                                             // 20%
	RateSumForMakerAndMaintainerMine = "0.2"                                             // 10% + 10%
	vxMineDust                       = new(big.Int).Mul(commonTokenPow, big.NewInt(100)) // 100 VITE

	ViteTokenDecimals int32 = 18

	QuoteTokenTypeInfos = map[int32]*QuoteTokenTypeInfo{
		ViteTokenType: &QuoteTokenTypeInfo{Decimals: 18, DefaultTradeThreshold: viteMinAmount, DefaultMineThreshold: viteMineThreshold},
		EthTokenType:  &QuoteTokenTypeInfo{Decimals: 18, DefaultTradeThreshold: ethMinAmount, DefaultMineThreshold: ethMineThreshold},
		BtcTokenType:  &QuoteTokenTypeInfo{Decimals: 8, DefaultTradeThreshold: bitcoinMinAmount, DefaultMineThreshold: bitcoinMineThreshold},
		UsdTokenType:  &QuoteTokenTypeInfo{Decimals: 8, DefaultTradeThreshold: usdMinAmount, DefaultMineThreshold: usdMineThreshold},
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

//MethodNameDexFundOwnerConfig
const (
	OwnerConfigOwner          = 1
	OwnerConfigTimer          = 2
	OwnerConfigTrigger        = 4
	OwnerConfigStopViteX      = 8
	OwnerConfigMakerMineProxy = 16
	OwnerConfigMaintainer     = 32
)

//MethodNameDexFundOwnerConfigTrade
const (
	OwnerConfigMineMarket     = 1
	OwnerConfigNewQuoteToken  = 2
	OwnerConfigTradeThreshold = 4
	OwnerConfigMineThreshold  = 8
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
	MineForMaker = iota + 1
	MineForMaintainer
)

const (
	FeeDividendJob = iota + 1
	BrokerFeeDividendJob
	MineVxForFeeJob
	MineVxForPledgeJob
	MineVxForMakerAndMaintainerJob
)

const (
	GetTokenForNewMarket     = 1
	GetTokenForSetQuote      = 2
	GetTokenForTransferOwner = 3
)

type QuoteTokenTypeInfo struct {
	Decimals              int32
	DefaultTradeThreshold *big.Int
	DefaultMineThreshold  *big.Int
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

type ParamDexPeriodJob struct {
	PeriodId uint64
	BizType  uint8
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
	OperationCode  uint8
	Owner          types.Address // 1 owner
	Timer          types.Address // 2 timerAddress
	Trigger        types.Address // 4 maintainer
	StopViteX      bool          // 8 stopViteX
	MakerMineProxy types.Address // 16 maker mine proxy
	Maintainer     types.Address // 32 maintainer
}

type ParamDexFundOwnerConfigTrade struct {
	OperationCode      uint8
	TradeToken         types.TokenTypeId // 1 mineMarket
	QuoteToken         types.TokenTypeId // 1 mineMarket
	AllowMine          bool              // 1 mineMarket
	NewQuoteToken      types.TokenTypeId // 2 new quote token
	QuoteTokenType     uint8             // 2 new quote token
	TokenType4TradeThr uint8             // 4 maintainer
	TradeThreshold     *big.Int          // 4 maintainer
	TokenType4MineThr  uint8             // 8 maintainer
	MineThreshold      *big.Int          // 8 maintainer
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

type PledgesForVx struct {
	dexproto.PledgesForVx
}

func (psv *PledgesForVx) Serialize() (data []byte, err error) {
	return proto.Marshal(&psv.PledgesForVx)
}

func (psv *PledgesForVx) DeSerialize(data []byte) error {
	pledgesForVx := dexproto.PledgesForVx{}
	if err := proto.Unmarshal(data, &pledgesForVx); err != nil {
		return err
	} else {
		psv.PledgesForVx = pledgesForVx
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

func SubUserFund(db vm_db.VmDb, address types.Address, tokenId []byte, amount *big.Int) (updatedAcc *dexproto.Account, err error) {
	if userFund, ok := GetUserFund(db, address); ok {
		var foundAcc bool
		for _, acc := range userFund.Accounts {
			if bytes.Equal(acc.Token, tokenId) {
				foundAcc = true
				available := new(big.Int).SetBytes(acc.Available)
				if available.Cmp(amount) < 0 {
					err = ExceedFundAvailableErr
				} else {
					acc.Available = available.Sub(available, amount).Bytes()
				}
				updatedAcc = acc
				break
			}
		}
		if foundAcc {
			SaveUserFund(db, address, userFund)
		} else {
			err = ExceedFundAvailableErr
		}
	} else {
		err = ExceedFundAvailableErr
	}
	return
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
	amtWithTks := MapToAmountWithTokens(funds)
	for _, amtWtTk := range amtWithTks {
		acc := &dexproto.Account{}
		acc.Token = amtWtTk.Token.Bytes()
		acc.Available = amtWtTk.Amount.Bytes()
		userFund.Accounts = append(userFund.Accounts, acc)
	}
	SaveUserFund(db, address, userFund)
	return nil
}

func DepositUserAccount(db vm_db.VmDb, address types.Address, token types.TokenTypeId, amount *big.Int) (updatedAcc *dexproto.Account) {
	userFund, _ := GetUserFund(db, address)
	var foundToken bool
	for _, acc := range userFund.Accounts {
		if bytes.Equal(acc.Token, token.Bytes()) {
			acc.Available = AddBigInt(acc.Available, amount.Bytes())
			updatedAcc = acc
			foundToken = true
			break
		}
	}
	if !foundToken {
		updatedAcc = &dexproto.Account{}
		updatedAcc.Token = token.Bytes()
		updatedAcc.Available = amount.Bytes()
		userFund.Accounts = append(userFund.Accounts, updatedAcc)
	}
	SaveUserFund(db, address, userFund)
	return
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
	serializeToDb(db, GetFeeSumCurrentKey(db, reader), feeSum)
}

func SaveFeeSumWithPeriodId(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
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

func RollAndGentNewFeeSumByPeriod(db vm_db.VmDb, periodId uint64) (rolledFeeSumByPeriod *FeeSumByPeriod) {
	formerId := GetFeeSumLastPeriodIdForRoll(db)
	rolledFeeSumByPeriod = &FeeSumByPeriod{}
	if formerId > 0 {
		if formerFeeSumByPeriod, ok := GetFeeSumByPeriodId(db, formerId); !ok { // lastPeriod has been deleted on fee dividend
			panic(NoFeeSumFoundForValidPeriodErr)
		} else {
			rolledFeeSumByPeriod.LastValidPeriod = formerId
			for _, feeForDividend := range formerFeeSumByPeriod.FeesForDividend {
				rolledFee := &dexproto.FeeSumForDividend{}
				rolledFee.Token = feeForDividend.Token
				_, rolledAmount := splitDividendPool(feeForDividend)
				rolledFee.DividendPoolAmount = rolledAmount.Bytes()
				rolledFeeSumByPeriod.FeesForDividend = append(rolledFeeSumByPeriod.FeesForDividend, rolledFee)
			}
		}
	} else {
		// On startup, save one empty dividendPool for vite to diff db storage empty for serialize result
		rolledFee := &dexproto.FeeSumForDividend{}
		rolledFee.Token = ledger.ViteTokenId.Bytes()
		rolledFeeSumByPeriod.FeesForDividend = append(rolledFeeSumByPeriod.FeesForDividend, rolledFee)
	}
	SaveFeeSumLastPeriodIdForRoll(db, periodId)
	return
}

func MarkFeeSumAsMinedVxDivided(db vm_db.VmDb, feeSum *FeeSumByPeriod, periodId uint64) {
	if feeSum.FinishFeeDividend {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FinishVxMine = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
	if feeSum.LastValidPeriod > 0 {
		markFormerFeeSumsAsMined(db, feeSum.LastValidPeriod)
	}
}

func markFormerFeeSumsAsMined(db vm_db.VmDb, periodId uint64) {
	if feeSum, ok := GetFeeSumByPeriodId(db, periodId); ok {
		MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)
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

func SaveFeeSumLastPeriodIdForRoll(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastFeeSumPeriodKey, Uint64ToBytes(periodId))
}

func SaveCurrentBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, broker []byte, brokerFeeSum *BrokerFeeSumByPeriod) {
	serializeToDb(db, GetCurrentBrokerFeeSumKey(db, reader, broker), brokerFeeSum)
}

func GetCurrentBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, broker []byte) (*BrokerFeeSumByPeriod, bool) {
	currentBrokerFeeSumKey := GetCurrentBrokerFeeSumKey(db, reader, broker)
	return getBrokerFeeSumByKey(db, currentBrokerFeeSumKey)
}

func GetBrokerFeeSumByPeriodId(db vm_db.VmDb, broker []byte, periodId uint64) (*BrokerFeeSumByPeriod, bool) {
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

func IsValidFeeForMine(userFee *dexproto.UserFeeAccount, mineThreshold *big.Int) bool {
	return new(big.Int).Add(new(big.Int).SetBytes(userFee.BaseAmount), new(big.Int).SetBytes(userFee.InviteBonusAmount)).Cmp(mineThreshold) >= 0
}

func GetUserFeesKey(address []byte) []byte {
	return append(userFeeKeyPrefix, address...)
}

func GetVxFunds(db vm_db.VmDb, address []byte) (vxFunds *VxFunds, ok bool) {
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

func GetVxSumFunds(db vm_db.VmDb) (vxSumFunds *VxFunds, ok bool) {
	vxSumFunds = &VxFunds{}
	ok = deserializeFromDb(db, vxSumFundsKey, vxSumFunds)
	return
}

func SaveVxSumFunds(db vm_db.VmDb, vxSumFunds *VxFunds) {
	serializeToDb(db, vxSumFundsKey, vxSumFunds)
}

func SaveMakerProxyAmountByPeriodId(db vm_db.VmDb, periodId uint64, amount *big.Int) {
	setValueToDb(db, GetMarkerProxyAmountByPeriodIdKey(periodId), amount.Bytes())
}

func GetMakerProxyAmountByPeriodId(db vm_db.VmDb, periodId uint64) *big.Int {
	if amtBytes := getValueFromDb(db, GetMarkerProxyAmountByPeriodIdKey(periodId)); len(amtBytes) > 0 {
		return new(big.Int).SetBytes(amtBytes)
	} else {
		return new(big.Int)
	}
}

func DeleteMakerProxyAmountByPeriodId(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, GetMarkerProxyAmountByPeriodIdKey(periodId), nil)
}

func GetMarkerProxyAmountByPeriodIdKey(periodId uint64) []byte {
	return append(makerMineProxyAmountByPeriodKey, Uint64ToBytes(periodId)...)
}

func GetLastJobPeriodIdByBizType(db vm_db.VmDb, bizType uint8) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, GetLastJobPeriodIdKey(bizType)); len(lastPeriodIdBytes) == 8 {
		return binary.BigEndian.Uint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveLastJobPeriodIdByBizType(db vm_db.VmDb, periodId uint64, bizType uint8) {
	setValueToDb(db, GetLastJobPeriodIdKey(bizType), Uint64ToBytes(periodId))
}

func GetLastJobPeriodIdKey(bizType uint8) []byte {
	return append(lastJobPeriodIdWithBizTypeKey, byte(bizType))
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

func IsValidVxAmountBytesForDividend(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(VxDividendThreshold) >= 0
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func GetCurrentPeriodId(db vm_db.VmDb, reader util.ConsensusReader) uint64 {
	return GetPeriodIdByTimestamp(reader, GetTimestampInt64(db))
}

func GetPeriodIdByTimestamp(reader util.ConsensusReader, timestamp int64) uint64 {
	return reader.GetIndexByTime(timestamp, 0)
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
	action.QuoteTokenType = int32(quoteType)
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
	ok = deserializeFromDb(db, GetTokenInfoKey(token), tokenInfo)
	return
}

func SaveTokenInfo(db vm_db.VmDb, token types.TokenTypeId, tokenInfo *TokenInfo) {
	serializeToDb(db, GetTokenInfoKey(token), tokenInfo)
}

func GetTokenInfoKey(token types.TokenTypeId) []byte {
	return append(tokenInfoKeyPrefix, token.Bytes()...)
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
			panic(err)
		}
	} else {
		panic(FundOwnerNotConfigErr)
	}
}

func SetOwner(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, ownerKey, address.Bytes())
}

func GetTradeThreshold(db vm_db.VmDb, quoteTokenType int32) *big.Int {
	if val := getValueFromDb(db, GetTradeThresholdKey(uint8(quoteTokenType))); len(val) > 0 {
		return new(big.Int).SetBytes(val)
	} else {
		return QuoteTokenTypeInfos[quoteTokenType].DefaultTradeThreshold
	}
}

func SaveTradeThreshold(db vm_db.VmDb, quoteTokenType uint8, amount *big.Int) {
	setValueToDb(db, GetTradeThresholdKey(quoteTokenType), amount.Bytes())
}

func GetTradeThresholdKey(quoteTokenType uint8) []byte {
	return append(minTradeAmountKeyPrefix, quoteTokenType)
}

func GetMineThreshold(db vm_db.VmDb, quoteTokenType int32) *big.Int {
	if val := getValueFromDb(db, GetMineThresholdKey(uint8(quoteTokenType))); len(val) > 0 {
		return new(big.Int).SetBytes(val)
	} else {
		return QuoteTokenTypeInfos[quoteTokenType].DefaultMineThreshold
	}
}

func SaveMineThreshold(db vm_db.VmDb, quoteTokenType uint8, amount *big.Int) {
	setValueToDb(db, GetMineThresholdKey(quoteTokenType), amount.Bytes())
}

func GetMineThresholdKey(quoteTokenType uint8) []byte {
	return append(mineThresholdKeyPrefix, quoteTokenType)
}

func GetMakerMineProxy(db vm_db.VmDb) *types.Address {
	if mmpBytes := getValueFromDb(db, makerMineProxyKey); len(mmpBytes) == types.AddressSize {
		if makerMineProxy, err := types.BytesToAddress(mmpBytes); err == nil {
			return &makerMineProxy
		} else {
			panic(err)
		}
	} else {
		panic(NotSetMineProxyErr)
	}
}

func SaveMakerMineProxy(db vm_db.VmDb, addr types.Address) {
	setValueToDb(db, makerMineProxyKey, addr.Bytes())
}

func IsMakerMineProxy(db vm_db.VmDb, addr types.Address) bool {
	if mmpBytes := getValueFromDb(db, makerMineProxyKey); len(mmpBytes) == types.AddressSize {
		return bytes.Equal(addr.Bytes(), mmpBytes)
	} else {
		return false
	}
}

func GetMaintainer(db vm_db.VmDb) *types.Address {
	if mtBytes := getValueFromDb(db, maintainerKey); len(mtBytes) == types.AddressSize {
		if maintainer, err := types.BytesToAddress(mtBytes); err == nil {
			return &maintainer
		} else {
			panic(err)
		}
	} else {
		panic(NotSetMaintainerErr)
	}
}

func SaveMaintainer(db vm_db.VmDb, addr types.Address) {
	setValueToDb(db, maintainerKey, addr.Bytes())
}

func IsViteXStopped(db vm_db.VmDb) bool {
	stopped := getValueFromDb(db, viteXStoppedKey)
	return len(stopped) > 0
}

func SaveViteXStopped(db vm_db.VmDb, isStopViteX bool) {
	if isStopViteX {
		setValueToDb(db, viteXStoppedKey, []byte{1})
	} else {
		setValueToDb(db, viteXStoppedKey, nil)
	}
}

func ValidTimerAddress(db vm_db.VmDb, address types.Address) bool {
	if timerAddressBytes := getValueFromDb(db, timerAddressKey); bytes.Equal(timerAddressBytes, address.Bytes()) {
		return true
	}
	return false
}

func GetTimer(db vm_db.VmDb) *types.Address {
	if timerAddressBytes := getValueFromDb(db, timerAddressKey); len(timerAddressKey) == types.AddressSize {
		address := &types.Address{}
		address.SetBytes(timerAddressBytes)
		return address
	} else {
		return nil
	}
}

func SetTimerAddress(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, timerAddressKey, address.Bytes())
}

func ValidTriggerAddress(db vm_db.VmDb, address types.Address) bool {
	if triggerAddressBytes := getValueFromDb(db, triggerAddressKey); bytes.Equal(triggerAddressBytes, address.Bytes()) {
		return true
	}
	return false
}

func GetTrigger(db vm_db.VmDb) *types.Address {
	if triggerAddressBytes := getValueFromDb(db, triggerAddressKey); len(triggerAddressBytes) == types.AddressSize {
		address, _ := types.BytesToAddress(triggerAddressBytes)
		return &address
	} else {
		return nil
	}
}

func SetTriggerAddress(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, triggerAddressKey, address.Bytes())
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
	return append(pledgeForVxKeyPrefix, address.Bytes()...)
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
	return append(pledgeForVipKeyPrefix, address.Bytes()...)
}

func GetPledgesForVx(db vm_db.VmDb, address types.Address) (pledgesForVx *PledgesForVx, ok bool) {
	pledgesForVx = &PledgesForVx{}
	ok = deserializeFromDb(db, GetPledgesForVxKey(address), pledgesForVx)
	return
}

func SavePledgesForVx(db vm_db.VmDb, address types.Address, ps *PledgesForVx) {
	serializeToDb(db, GetPledgesForVxKey(address), ps)
}

func DeletePledgesForVx(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetPledgesForVxKey(address), nil)
}

func GetPledgesForVxKey(address types.Address) []byte {
	return append(pledgesForVxKeyPrefix, address.Bytes()...)
}

func GetPledgesForVxSum(db vm_db.VmDb) (pledgesForVx *PledgesForVx, ok bool) {
	pledgesForVx = &PledgesForVx{}
	ok = deserializeFromDb(db, pledgesForVxSumKey, pledgesForVx)
	return
}

func SavePledgesForVxSum(db vm_db.VmDb, ps *PledgesForVx) {
	serializeToDb(db, pledgesForVxSumKey, ps)
}

func IsValidPledgeAmountForVx(amount *big.Int) bool {
	return amount.Cmp(PledgeForVxThreshold) >= 0
}

func IsValidPledgeAmountBytesForVx(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(PledgeForVxThreshold) >= 0
}

func MatchPledgeForVxByPeriod(pledgesForVx *PledgesForVx, periodId uint64, checkDelete bool) (bool, []byte, bool, bool) {
	var (
		pledgeAmtBytes         []byte
		matchIndex             int
		needUpdatePledgesForVx bool
	)
	for i, pledge := range pledgesForVx.Pledges {
		if periodId >= pledge.Period {
			pledgeAmtBytes = pledge.Amount
			matchIndex = i
		} else {
			break
		}
	}
	if len(pledgeAmtBytes) == 0 {
		return false, nil, false, checkDelete && CheckUserPledgesForVxCanBeDelete(pledgesForVx)
	}
	if matchIndex > 0 { //remove obsolete items, but leave current matched item
		pledgesForVx.Pledges = pledgesForVx.Pledges[matchIndex:]
		needUpdatePledgesForVx = true
	}
	if len(pledgesForVx.Pledges) > 1 && pledgesForVx.Pledges[1].Period == periodId+1 {
		pledgesForVx.Pledges = pledgesForVx.Pledges[1:]
		needUpdatePledgesForVx = true
	}
	return true, pledgeAmtBytes, needUpdatePledgesForVx, checkDelete && CheckUserPledgesForVxCanBeDelete(pledgesForVx)
}

func CheckUserPledgesForVxCanBeDelete(pledgesForVx *PledgesForVx) bool {
	return len(pledgesForVx.Pledges) == 1 && !IsValidPledgeAmountBytesForVx(pledgesForVx.Pledges[0].Amount)
}

func GetTimestampInt64(db vm_db.VmDb) int64 {
	timestamp := GetTimerTimestamp(db)
	if timestamp == 0 {
		panic(NotSetTimestampErr)
	} else {
		return timestamp
	}
}

func SetTimerTimestamp(db vm_db.VmDb, timestamp int64, reader util.ConsensusReader) error {
	oldTime := GetTimerTimestamp(db)
	if timestamp > oldTime {
		oldPeriod := GetPeriodIdByTimestamp(reader, oldTime)
		newPeriod := GetPeriodIdByTimestamp(reader, timestamp)
		if newPeriod != oldPeriod {
			doRollPeriod(db, newPeriod)
		}
		setValueToDb(db, timestampKey, Uint64ToBytes(uint64(timestamp)))
		return nil
	} else {
		return InvalidTimestampFromTimerErr
	}
}

func doRollPeriod(db vm_db.VmDb, newPeriodId uint64) {
	newFeeSum := RollAndGentNewFeeSumByPeriod(db, newPeriodId)
	SaveFeeSumWithPeriodId(db, newFeeSum, newPeriodId)
}

func GetTimerTimestamp(db vm_db.VmDb) int64 {
	if bs := getValueFromDb(db, timestampKey); len(bs) == 8 {
		return int64(BytesToUint64(bs))
	} else {
		return 0
	}
}

func GetCodeByInviter(db vm_db.VmDb, address types.Address) uint32 {
	if bs := getValueFromDb(db, append(codeByInviterKeyPrefix, address.Bytes()...)); len(bs) == 4 {
		return BytesToUint32(bs)
	} else {
		return 0
	}
}

func SaveCodeByInviter(db vm_db.VmDb, address types.Address, inviteCode uint32) {
	setValueToDb(db, append(codeByInviterKeyPrefix, address.Bytes()...), Uint32ToBytes(inviteCode))
}

func GetInviterByCode(db vm_db.VmDb, inviteCode uint32) (inviter *types.Address, err error) {
	if bs := getValueFromDb(db, append(inviterByCodeKeyPrefix, Uint32ToBytes(inviteCode)...)); len(bs) == types.AddressSize {
		*inviter, err = types.BytesToAddress(bs)
		return
	} else {
		return nil, InvalidInviteCodeErr
	}
}

func SaveInviterByCode(db vm_db.VmDb, address types.Address, inviteCode uint32) {
	setValueToDb(db, append(inviterByCodeKeyPrefix, Uint32ToBytes(inviteCode)...), address.Bytes())
}

func SaveInviterByInvitee(db vm_db.VmDb, invitee, inviter types.Address) {
	setValueToDb(db, append(inviterByInviteeKeyPrefix, invitee.Bytes()...), inviter.Bytes())
}

func GetInviterByInvitee(db vm_db.VmDb, address types.Address) (inviter *types.Address, err error) {
	if bs := getValueFromDb(db, append(inviterByInviteeKeyPrefix, address.Bytes()...)); len(bs) == types.AddressSize {
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

func GetVxMinePool(db vm_db.VmDb) *big.Int {
	if data := getValueFromDb(db, vxMinePoolKey); len(data) > 0 {
		return new(big.Int).SetBytes(data)
	} else {
		return new(big.Int)
	}
}

func SaveVxMinePool(db vm_db.VmDb, amount *big.Int) {
	setValueToDb(db, vxMinePoolKey, amount.Bytes())
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
