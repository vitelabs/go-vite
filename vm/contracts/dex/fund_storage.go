package dex

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var (
	ownerKey                = []byte("own:")
	dexStoppedKey           = []byte("dexStp:")
	fundKeyPrefix           = []byte("fd:") // fund:types.Address
	minTradeAmountKeyPrefix = []byte("mTrAt:")
	mineThresholdKeyPrefix  = []byte("mTh:")

	dexTimestampKey = []byte("tts:") // dexTimestamp

	userFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	dexFeesKeyPrefix       = []byte("fS:")     // dexFees:periodId
	lastDexFeesPeriodIdKey = []byte("lFSPId:") //

	operatorFeesKeyPrefix = []byte("bf:") // operatorFees:periodId, 32 bytes prefix[3] + periodId[8]+ address[21]

	pendingNewMarketActionsKey          = []byte("pmkas:")
	pendingSetQuoteActionsKey           = []byte("psqas:")
	pendingTransferTokenOwnerActionsKey = []byte("ptoas:")
	marketIdKey                         = []byte("mkId:")
	orderIdSerialNoKey                  = []byte("orIdSl:")

	vxUnlocksKeyPrefix    = []byte("vxUl:")
	cancelStakesKeyPrefix = []byte("clSt:")

	timeOracleKey       = []byte("tmA:")
	periodJobTriggerKey = []byte("tgA:")

	vxFundKeyPrefix = []byte("vxF:")  // vxFund:types.Address
	vxSumFundsKey   = []byte("vxFS:") // vxFundSum

	autoLockMinedVxKeyPrefix = []byte("aLMVx:")
	vxLockedFundsKeyPrefix   = []byte("vxlF:")
	vxLockedSumFundsKey      = []byte("vxlFS:") // vxLockedFundSum

	lastJobPeriodIdWithBizTypeKey = []byte("ljpBId:")
	normalMineStartedKey          = []byte("nmst:")
	firstMinedVxPeriodIdKey       = []byte("fMVPId:")
	marketInfoKeyPrefix           = []byte("mk:") // market: tradeToke,quoteToken

	vipStakingKeyPrefix      = []byte("pldVip:")   // vipStaking: types.Address
	superVIPStakingKeyPrefix = []byte("pldSpVip:") // superVIPStaking: types.Address

	delegateStakeInfoPrefix          = []byte("ds:")
	delegateStakeAddressIndexPrefix  = []byte("dA:")
	delegateStakeIndexSerialNoPrefix = []byte("dsISN:")

	miningStakingsKeyPrefix       = []byte("pldsVx:")  // miningStakings: types.Address
	dexMiningStakingsKey          = []byte("pldsVxS:") // dexMiningStakings
	miningStakedAmountKeyPrefix   = []byte("pldVx:")   // miningStakedAmount: types.Address
	miningStakedAmountV2KeyPrefix = []byte("stVx:")    // miningStakedAmount: types.Address

	tokenInfoKeyPrefix = []byte("tk:") // token:tokenId
	vxMinePoolKey      = []byte("vxmPl:")
	vxBurnAmountKey    = []byte("vxBAt:")

	codeByInviterKeyPrefix    = []byte("itr2cd:")
	inviterByCodeKeyPrefix    = []byte("cd2itr:")
	inviterByInviteeKeyPrefix = []byte("ite2itr:")

	maintainerKey                    = []byte("mtA:")
	makerMiningAdminKey              = []byte("mmpA:")
	makerMiningPoolByPeriodKey       = []byte("mmpaP:")
	lastSettledMakerMinedVxPeriodKey = []byte("lsmmvp:")
	lastSettledMakerMinedVxPageKey   = []byte("lsmmvpp:")

	viteOwnerInitiated = []byte("voited:")

	grantedMarketToAgentKeyPrefix = []byte("gtMkAt:")

	commonTokenPow = new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)

	VxTokenId, _             = types.HexToTokenTypeId("tti_564954455820434f494e69b5")
	PreheatMinedAmtPerPeriod = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	VxMinedAmtFirstPeriod    = new(big.Int).Mul(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(13)), nil), big.NewInt(47703236213)) // 477032.36213

	VxDividendThreshold      = new(big.Int).Mul(commonTokenPow, big.NewInt(10))
	NewMarketFeeAmount       = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	NewMarketFeeMineAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))
	NewMarketFeeDonateAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(4000))
	NewMarketFeeBurnAmount   = new(big.Int).Mul(commonTokenPow, big.NewInt(5000))
	NewInviterFeeAmount      = new(big.Int).Mul(commonTokenPow, big.NewInt(1000))

	VxLockThreshold = new(big.Int).Set(commonTokenPow)
	SchedulePeriods = 7 // T+7 schedule

	StakeForMiningMinAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	StakeForVIPAmount       = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	StakeForMiningThreshold = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	StakeForSuperVIPAmount  = new(big.Int).Mul(commonTokenPow, big.NewInt(1000000))

	viteMinAmount    = new(big.Int).Mul(commonTokenPow, big.NewInt(100)) // 100 VITE
	ethMinAmount     = new(big.Int).Div(commonTokenPow, big.NewInt(100)) // 0.01 ETH
	bitcoinMinAmount = big.NewInt(50000)                                 // 0.0005 BTC
	usdMinAmount     = big.NewInt(1000000)                               // 1 USD

	viteMineThreshold    = new(big.Int).Mul(commonTokenPow, big.NewInt(2))    // 2 VITE
	ethMineThreshold     = new(big.Int).Div(commonTokenPow, big.NewInt(5000)) // 0.0002 ETH
	bitcoinMineThreshold = big.NewInt(1000)                                   // 0.00001 BTC
	usdMineThreshold     = big.NewInt(20000)                                  // 0.02USD

	RateSumForFeeMine                = "0.6"                                             // 15% * 4
	RateForStakingMine               = "0.2"                                             // 20%
	RateSumForMakerAndMaintainerMine = "0.2"                                             // 10% + 10%
	vxMineDust                       = new(big.Int).Mul(commonTokenPow, big.NewInt(100)) // 100 VITE

	ViteTokenDecimals int32 = 18

	QuoteTokenTypeInfos = map[int32]*QuoteTokenTypeInfo{
		ViteTokenType: &QuoteTokenTypeInfo{Decimals: 18, DefaultTradeThreshold: viteMinAmount, DefaultMineThreshold: viteMineThreshold},
		EthTokenType:  &QuoteTokenTypeInfo{Decimals: 18, DefaultTradeThreshold: ethMinAmount, DefaultMineThreshold: ethMineThreshold},
		BtcTokenType:  &QuoteTokenTypeInfo{Decimals: 8, DefaultTradeThreshold: bitcoinMinAmount, DefaultMineThreshold: bitcoinMineThreshold},
		UsdTokenType:  &QuoteTokenTypeInfo{Decimals: 6, DefaultTradeThreshold: usdMinAmount, DefaultMineThreshold: usdMineThreshold},
	}
	initOwner, _          = types.HexToAddress("vite_a8a00b3a2f60f5defb221c68f79b65f3620ee874f951a825db")
	initViteTokenOwner, _ = types.HexToAddress("vite_050697d3810c30816b005a03511c734c1159f50907662b046f")
	newOrderMethodId, _   = hex.DecodeString("147927ec")
)

const (
	priceIntMaxLen     = 12
	priceDecimalMaxLen = 12
)

const (
	StakeForMining = iota + 1
	StakeForVIP
	StakeForSuperVIP
	StakeForPrincipalSuperVIP
)

const (
	Stake = iota + 1
	CancelStake
)

//MethodNameDexFundDexAdminConfig
const (
	AdminConfigOwner            = 1
	AdminConfigTimeOracle       = 2
	AdminConfigPeriodJobTrigger = 4
	AdminConfigStopDex          = 8
	AdminConfigMakerMiningAdmin = 16
	AdminConfigMaintainer       = 32
)

//MethodNameDexFundTradeAdminConfig
const (
	TradeAdminConfigMineMarket     = 1
	TradeAdminConfigNewQuoteToken  = 2
	TradeAdminConfigTradeThreshold = 4
	TradeAdminConfigMineThreshold  = 8
	TradeAdminStartNormalMine      = 16
	TradeAdminBurnExtraVx          = 32
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
	OperatorFeeDividendJob
	MineVxForFeeJob
	MineVxForStakingJob
	MineVxForMakerAndMaintainerJob
	FinishVxUnlock
	FinishCancelMiningStake
)

const (
	GetTokenForNewMarket = iota + 1
	GetTokenForSetQuote
	GetTokenForTransferOwner
)

const (
	GrantAgent = iota + 1
	RevokeAgent
)

const (
	LockVx = iota + 1
	UnlockVx
)

const (
	AutoLockMinedVx = iota + 1
)

const (
	BurnForDexViteFee = iota + 1
)

const (
	StakeSubmitted = iota + 1
	StakeConfirmed
)

type QuoteTokenTypeInfo struct {
	Decimals              int32
	DefaultTradeThreshold *big.Int
	DefaultMineThreshold  *big.Int
}

type ParamWithdraw struct {
	Token  types.TokenTypeId
	Amount *big.Int
}

type ParamPlaceOrder struct {
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
	Side       bool
	OrderType  uint8
	Price      string
	Quantity   *big.Int
}

type ParamPlaceAgentOrder struct {
	Principal types.Address
	ParamPlaceOrder
}

type ParamTriggerPeriodJob struct {
	PeriodId uint64
	BizType  uint8
}

type ParamSerializedData struct {
	Data []byte
}

type ParamOpenNewMarket struct {
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
}

type ParamStakeForMining struct {
	ActionType uint8 // 1: stake 2: cancel stake
	Amount     *big.Int
}

type ParamStakeForVIP struct {
	ActionType uint8 // 1: stake 2: cancel stake
}

type ParamDelegateStakeCallback struct {
	StakeAddress types.Address
	Beneficiary  types.Address
	Amount       *big.Int
	Bid          uint8
	Success      bool
}

type ParamGetTokenInfoCallback struct {
	TokenId     types.TokenTypeId
	Bid         uint8
	Exist       bool
	Decimals    uint8
	TokenSymbol string
	Index       uint16
	Owner       types.Address
}

type ParamDexAdminConfig struct {
	OperationCode    uint8
	Owner            types.Address // 1 owner
	TimeOracle       types.Address // 2 timeOracle
	PeriodJobTrigger types.Address // 4 periodJobTrigger
	StopDex          bool          // 8 stopDex
	MakerMiningAdmin types.Address // 16 maker mining admin
	Maintainer       types.Address // 32 maintainer
}

type ParamTradeAdminConfig struct {
	OperationCode               uint8
	TradeToken                  types.TokenTypeId // 1 mineMarket
	QuoteToken                  types.TokenTypeId // 1 mineMarket
	AllowMining                 bool              // 1 mineMarket
	NewQuoteToken               types.TokenTypeId // 2 new quote token
	QuoteTokenType              uint8             // 2 new quote token
	TokenTypeForTradeThreshold  uint8             // 4 tradeThreshold
	MinTradeThreshold           *big.Int          // 4 tradeThreshold
	TokenTypeForMiningThreshold uint8             // 8 miningThreshold
	MinMiningThreshold          *big.Int          // 8 miningThreshold
}

type ParamMarketAdminConfig struct {
	OperationCode uint8 // 1 owner, 2 takerRate, 4 makerRate, 8 stopMarket
	TradeToken    types.TokenTypeId
	QuoteToken    types.TokenTypeId
	MarketOwner   types.Address
	TakerFeeRate  int32
	MakerFeeRate  int32
	StopMarket    bool
}

type ParamTransferTokenOwnership struct {
	Token    types.TokenTypeId
	NewOwner types.Address
}

type ParamNotifyTime struct {
	Timestamp int64
}

type ParamConfigMarketAgents struct {
	ActionType  uint8 // 1: grant 2: revoke
	Agent       types.Address
	TradeTokens []types.TokenTypeId
	QuoteTokens []types.TokenTypeId
}

type ParamLockVxForDividend struct {
	ActionType uint8 // 1: lockVx 2: unlockVx
	Amount     *big.Int
}

type ParamSwitchConfig struct {
	SwitchType uint8 // 1: autoLockMinedVx
	Enable     bool
}

type ParamCancelStakeById struct {
	Id types.Hash
}

type ParamDelegateStakeCallbackV2 struct {
	Id      types.Hash
	Success bool
}

type Fund struct {
	dexproto.Fund
}

type SerializableDex interface {
	Serialize() ([]byte, error)
	DeSerialize([]byte) error
}

func (df *Fund) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.Fund)
}

func (df *Fund) DeSerialize(fundData []byte) (err error) {
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

type SerialNo struct {
	dexproto.SerialNo
}

func (osn *SerialNo) Serialize() (data []byte, err error) {
	return proto.Marshal(&osn.SerialNo)
}

func (osn *SerialNo) DeSerialize(data []byte) error {
	serialNo := dexproto.SerialNo{}
	if err := proto.Unmarshal(data, &serialNo); err != nil {
		return err
	} else {
		osn.SerialNo = serialNo
		return nil
	}
}

type DexFeesByPeriod struct {
	dexproto.DexFeesByPeriod
}

func (df *DexFeesByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.DexFeesByPeriod)
}

func (df *DexFeesByPeriod) DeSerialize(data []byte) (err error) {
	dexFeesByPeriod := dexproto.DexFeesByPeriod{}
	if err := proto.Unmarshal(data, &dexFeesByPeriod); err != nil {
		return err
	} else {
		df.DexFeesByPeriod = dexFeesByPeriod
		return nil
	}
}

type OperatorFeesByPeriod struct {
	dexproto.OperatorFeesByPeriod
}

func (bfs *OperatorFeesByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&bfs.OperatorFeesByPeriod)
}

func (bfs *OperatorFeesByPeriod) DeSerialize(data []byte) (err error) {
	operatorFeesByPeriod := dexproto.OperatorFeesByPeriod{}
	if err := proto.Unmarshal(data, &operatorFeesByPeriod); err != nil {
		return err
	} else {
		bfs.OperatorFeesByPeriod = operatorFeesByPeriod
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
	dexproto.PendingSetQuoteTokenActions
}

func (psq *PendingSetQuotes) Serialize() (data []byte, err error) {
	return proto.Marshal(&psq.PendingSetQuoteTokenActions)
}

func (psq *PendingSetQuotes) DeSerialize(data []byte) error {
	pendingSetQuotes := dexproto.PendingSetQuoteTokenActions{}
	if err := proto.Unmarshal(data, &pendingSetQuotes); err != nil {
		return err
	} else {
		psq.PendingSetQuoteTokenActions = pendingSetQuotes
		return nil
	}
}

type PendingTransferTokenOwnerActions struct {
	dexproto.PendingTransferTokenOwnerActions
}

func (psq *PendingTransferTokenOwnerActions) Serialize() (data []byte, err error) {
	return proto.Marshal(&psq.PendingTransferTokenOwnerActions)
}

func (psq *PendingTransferTokenOwnerActions) DeSerialize(data []byte) error {
	pendingTransferTokenOwners := dexproto.PendingTransferTokenOwnerActions{}
	if err := proto.Unmarshal(data, &pendingTransferTokenOwners); err != nil {
		return err
	} else {
		psq.PendingTransferTokenOwnerActions = pendingTransferTokenOwners
		return nil
	}
}

type VIPStaking struct {
	dexproto.VIPStaking
}

func (pv *VIPStaking) Serialize() (data []byte, err error) {
	return proto.Marshal(&pv.VIPStaking)
}

func (pv *VIPStaking) DeSerialize(data []byte) error {
	vipStaking := dexproto.VIPStaking{}
	if err := proto.Unmarshal(data, &vipStaking); err != nil {
		return err
	} else {
		pv.VIPStaking = vipStaking
		return nil
	}
}

type MiningStakings struct {
	dexproto.MiningStakings
}

func (mss *MiningStakings) Serialize() (data []byte, err error) {
	return proto.Marshal(&mss.MiningStakings)
}

func (mss *MiningStakings) DeSerialize(data []byte) error {
	miningStakings := dexproto.MiningStakings{}
	if err := proto.Unmarshal(data, &miningStakings); err != nil {
		return err
	} else {
		mss.MiningStakings = miningStakings
		return nil
	}
}

type DelegateStakeInfo struct {
	dexproto.DelegateStakeInfo
}

func (dsi *DelegateStakeInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&dsi.DelegateStakeInfo)
}

func (dsi *DelegateStakeInfo) DeSerialize(data []byte) error {
	delegateStakeInfo := dexproto.DelegateStakeInfo{}
	if err := proto.Unmarshal(data, &delegateStakeInfo); err != nil {
		return err
	} else {
		dsi.DelegateStakeInfo = delegateStakeInfo
		return nil
	}
}

type DelegateStakeAddressIndex struct {
	dexproto.DelegateStakeAddressIndex
}

func (dsi *DelegateStakeAddressIndex) Serialize() (data []byte, err error) {
	return proto.Marshal(&dsi.DelegateStakeAddressIndex)
}

func (dsi *DelegateStakeAddressIndex) DeSerialize(data []byte) error {
	stakeIndex := dexproto.DelegateStakeAddressIndex{}
	if err := proto.Unmarshal(data, &stakeIndex); err != nil {
		return err
	} else {
		dsi.DelegateStakeAddressIndex = stakeIndex
		return nil
	}
}

type VxUnlocks struct {
	dexproto.VxUnlocks
}

func (vu *VxUnlocks) Serialize() (data []byte, err error) {
	return proto.Marshal(&vu.VxUnlocks)
}

func (vu *VxUnlocks) DeSerialize(data []byte) error {
	vxUnlocks := dexproto.VxUnlocks{}
	if err := proto.Unmarshal(data, &vxUnlocks); err != nil {
		return err
	} else {
		vu.VxUnlocks = vxUnlocks
		return nil
	}
}

type CancelStakes struct {
	dexproto.CancelStakes
}

func (cs *CancelStakes) Serialize() (data []byte, err error) {
	return proto.Marshal(&cs.CancelStakes)
}

func (cs *CancelStakes) DeSerialize(data []byte) error {
	cancelStakes := dexproto.CancelStakes{}
	if err := proto.Unmarshal(data, &cancelStakes); err != nil {
		return err
	} else {
		cs.CancelStakes = cancelStakes
		return nil
	}
}

func GetAccountByToken(fund *Fund, token types.TokenTypeId) (account *dexproto.Account, exists bool) {
	for _, a := range fund.Accounts {
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

func GetAccounts(fund *Fund, tokenId *types.TokenTypeId) ([]*Account, error) {
	if fund == nil || len(fund.Accounts) == 0 {
		return nil, errors.New("fund user doesn't exist.")
	}
	var accounts = make([]*Account, 0)
	if tokenId != nil {
		for _, v := range fund.Accounts {
			if bytes.Equal(tokenId.Bytes(), v.Token) {
				var acc = &Account{}
				acc.Deserialize(v)
				accounts = append(accounts, acc)
				break
			}
		}
	} else {
		for _, v := range fund.Accounts {
			var acc = &Account{}
			acc.Deserialize(v)
			accounts = append(accounts, acc)
		}
	}
	return accounts, nil
}

func GetFund(db vm_db.VmDb, address types.Address) (fund *Fund, ok bool) {
	fund = &Fund{}
	ok = deserializeFromDb(db, GetFundKey(address), fund)
	return
}

func SaveFund(db vm_db.VmDb, address types.Address, fund *Fund) {
	serializeToDb(db, GetFundKey(address), fund)
}

func ReduceAccount(db vm_db.VmDb, address types.Address, tokenId []byte, amount *big.Int) (*dexproto.Account, error) {
	return updateFund(db, address, tokenId, amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		available := new(big.Int).SetBytes(acc.Available)
		if available.Cmp(amt) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.Available = available.Sub(available, amt).Bytes()
		}
		return acc, nil
	})
}

func LockVxForDividend(db vm_db.VmDb, address types.Address, amount *big.Int) (*dexproto.Account, error) {
	return updateFund(db, address, VxTokenId.Bytes(), amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		available := new(big.Int).SetBytes(acc.Available)
		if available.Cmp(amt) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.Available = available.Sub(available, amt).Bytes()
			acc.VxLocked = AddBigInt(acc.VxLocked, amt.Bytes())
		}
		return acc, nil
	})
}

func ScheduleVxUnlockForDividend(db vm_db.VmDb, address types.Address, amount *big.Int) (updatedAcc *dexproto.Account, err error) {
	return updateFund(db, address, VxTokenId.Bytes(), amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		vxLocked := new(big.Int).SetBytes(acc.VxLocked)
		if vxLocked.Cmp(amt) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			vxLocked.Sub(vxLocked, amt)
			if vxLocked.Sign() != 0 && vxLocked.Cmp(VxDividendThreshold) < 0 {
				return nil, LockedVxAmountLeavedNotValidErr
			}
			acc.VxLocked = vxLocked.Bytes()
			acc.VxUnlocking = AddBigInt(acc.VxUnlocking, amt.Bytes())
		}
		return acc, nil
	})
}

func FinishVxUnlockForDividend(db vm_db.VmDb, address types.Address, amount *big.Int) (*dexproto.Account, error) {
	return updateFund(db, address, VxTokenId.Bytes(), amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		vxUnlocking := new(big.Int).SetBytes(acc.VxUnlocking)
		if vxUnlocking.Cmp(amt) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.VxUnlocking = vxUnlocking.Sub(vxUnlocking, amt).Bytes()
			acc.Available = AddBigInt(acc.Available, amt.Bytes())
		}
		return acc, nil
	})
}

func ScheduleCancelStake(db vm_db.VmDb, address types.Address, amount *big.Int) (updatedAcc *dexproto.Account, err error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		acc.CancellingStake = AddBigInt(acc.CancellingStake, amount.Bytes())
		return acc, nil
	})
}

func FinishCancelStake(db vm_db.VmDb, address types.Address, amount *big.Int) (updatedAcc *dexproto.Account, err error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), amount, func(acc *dexproto.Account, amt *big.Int) (*dexproto.Account, error) {
		cancellingStakeAmount := new(big.Int).SetBytes(acc.CancellingStake)
		if cancellingStakeAmount.Cmp(amt) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.CancellingStake = cancellingStakeAmount.Sub(cancellingStakeAmount, amt).Bytes()
			acc.Available = AddBigInt(acc.Available, amt.Bytes())
		}
		return acc, nil
	})
}

func updateFund(db vm_db.VmDb, address types.Address, tokenId []byte, amount *big.Int, updateAccFunc func(*dexproto.Account, *big.Int) (*dexproto.Account, error)) (updatedAcc *dexproto.Account, err error) {
	if fund, ok := GetFund(db, address); ok {
		var foundAcc bool
		for _, acc := range fund.Accounts {
			if bytes.Equal(acc.Token, tokenId) {
				foundAcc = true
				if updatedAcc, err = updateAccFunc(acc, amount); err != nil {
					return
				}
				break
			}
		}
		if foundAcc {
			SaveFund(db, address, fund)
		} else {
			err = ExceedFundAvailableErr
		}
	} else {
		err = ExceedFundAvailableErr
	}
	return
}

func BatchUpdateFund(db vm_db.VmDb, address types.Address, accounts map[types.TokenTypeId]*big.Int) error {
	userFund, _ := GetFund(db, address)
	for _, acc := range userFund.Accounts {
		if tk, err := types.BytesToTokenTypeId(acc.Token); err != nil {
			return err
		} else {
			if amt, ok := accounts[tk]; ok {
				acc.Available = AddBigInt(acc.Available, amt.Bytes())
				delete(accounts, tk)
			}
		}
	}
	amtWithTks := MapToAmountWithTokens(accounts)
	for _, amtWtTk := range amtWithTks {
		acc := &dexproto.Account{}
		acc.Token = amtWtTk.Token.Bytes()
		acc.Available = amtWtTk.Amount.Bytes()
		userFund.Accounts = append(userFund.Accounts, acc)
	}
	SaveFund(db, address, userFund)
	return nil
}

func DepositAccount(db vm_db.VmDb, address types.Address, token types.TokenTypeId, amount *big.Int) (updatedAcc *dexproto.Account) {
	userFund, _ := GetFund(db, address)
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
	SaveFund(db, address, userFund)
	return
}

func LockMinedVx(db vm_db.VmDb, address types.Address, amount *big.Int) (updatedAcc *dexproto.Account) {
	userFund, _ := GetFund(db, address)
	var foundVxAcc bool
	for _, acc := range userFund.Accounts {
		if bytes.Equal(acc.Token, VxTokenId.Bytes()) {
			acc.VxLocked = AddBigInt(acc.VxLocked, amount.Bytes())
			updatedAcc = acc
			foundVxAcc = true
			break
		}
	}
	if !foundVxAcc {
		updatedAcc = &dexproto.Account{}
		updatedAcc.Token = VxTokenId.Bytes()
		updatedAcc.VxLocked = amount.Bytes()
		userFund.Accounts = append(userFund.Accounts, updatedAcc)
	}
	SaveFund(db, address, userFund)
	return
}

func GetUserFundsByPage(db abi.StorageDatabase, lastAddress types.Address, count int) (funds []*Fund, err error) {
	var iterator interfaces.StorageIterator
	if iterator, err = db.NewStorageIterator(fundKeyPrefix); err != nil {
		return
	}
	defer iterator.Release()

	if lastAddress != types.ZERO_ADDRESS {
		ok := iterator.Seek(GetFundKey(lastAddress))
		if !ok {
			err = fmt.Errorf("last address not valid for page funds")
			return
		}
	}
	funds = make([]*Fund, 0, count)
	for {
		if !iterator.Next() {
			if err = iterator.Error(); err != nil {
				return
			}
			break
		}
		key := iterator.Key()
		data := iterator.Value()
		if len(data) > 0 {
			fund := &Fund{}
			if err = fund.DeSerialize(data); err != nil {
				return
			} else {
				fund.Address = make([]byte, types.AddressSize)
				copy(fund.Address[:], key[len(fundKeyPrefix):])
				funds = append(funds, fund)
				if len(funds) == count {
					return
				}
			}
		}
	}
	return
}

func GetFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetCurrentDexFees(db vm_db.VmDb, reader util.ConsensusReader) (*DexFeesByPeriod, bool) {
	return getDexFeesByKey(db, GetDexFeesKeyByPeriodId(GetCurrentPeriodId(db, reader)))
}

func GetDexFeesByPeriodId(db vm_db.VmDb, periodId uint64) (*DexFeesByPeriod, bool) {
	return getDexFeesByKey(db, GetDexFeesKeyByPeriodId(periodId))
}

func getDexFeesByKey(db vm_db.VmDb, feeKey []byte) (*DexFeesByPeriod, bool) {
	dexFeesByPeriod := &DexFeesByPeriod{}
	ok := deserializeFromDb(db, feeKey, dexFeesByPeriod)
	return dexFeesByPeriod, ok
}

//get all dexFeeses that not divided yet
func GetNotFinishDividendDexFeesByPeriodMap(db vm_db.VmDb, periodId uint64) map[uint64]*DexFeesByPeriod {
	var (
		dexFeesByPeriods = make(map[uint64]*DexFeesByPeriod)
		dexFeesByPeriod  *DexFeesByPeriod
		ok, everFound    bool
	)
	for {
		if dexFeesByPeriod, ok = GetDexFeesByPeriodId(db, periodId); !ok { // found first valid period
			if periodId > 0 && !everFound {
				periodId--
				continue
			} else { // lastValidPeriod is delete
				return dexFeesByPeriods
			}
		} else {
			everFound = true
			if !dexFeesByPeriod.FinishDividend {
				dexFeesByPeriods[periodId] = dexFeesByPeriod
			} else {
				return dexFeesByPeriods
			}
		}
		periodId = dexFeesByPeriod.LastValidPeriod
		if periodId == 0 {
			return dexFeesByPeriods
		}
	}
}

func SaveDexFeesByPeriodId(db vm_db.VmDb, periodId uint64, dexFeesByPeriod *DexFeesByPeriod) {
	serializeToDb(db, GetDexFeesKeyByPeriodId(periodId), dexFeesByPeriod)
}

//dexFees used both by fee dividend and mined vx dividend
func MarkDexFeesFinishDividend(db vm_db.VmDb, dexFeesByPeriod *DexFeesByPeriod, periodId uint64) {
	if dexFeesByPeriod.FinishMine {
		setValueToDb(db, GetDexFeesKeyByPeriodId(periodId), nil)
	} else {
		dexFeesByPeriod.FinishDividend = true
		serializeToDb(db, GetDexFeesKeyByPeriodId(periodId), dexFeesByPeriod)
	}
}

func RollAndGentNewDexFeesByPeriod(db vm_db.VmDb, periodId uint64) (rolledDexFeesByPeriod *DexFeesByPeriod) {
	formerId := GetDexFeesLastPeriodIdForRoll(db)
	rolledDexFeesByPeriod = &DexFeesByPeriod{}
	if formerId > 0 {
		if formerDexFeesByPeriod, ok := GetDexFeesByPeriodId(db, formerId); !ok { // lastPeriod has been deleted on fee dividend
			panic(NoDexFeesFoundForValidPeriodErr)
		} else {
			rolledDexFeesByPeriod.LastValidPeriod = formerId
			for _, formerFeeForDividend := range formerDexFeesByPeriod.FeesForDividend {
				rolledFee := &dexproto.FeeForDividend{}
				rolledFee.Token = formerFeeForDividend.Token
				if bytes.Equal(rolledFee.Token, ledger.ViteTokenId.Bytes()) && IsEarthFork(db) {
					rolledFee.NotRoll = true
				}
				_, rolledAmount := splitDividendPool(formerFeeForDividend) //when former pool is NotRoll, rolledAmount is nil
				rolledFee.DividendPoolAmount = rolledAmount.Bytes()
				rolledDexFeesByPeriod.FeesForDividend = append(rolledDexFeesByPeriod.FeesForDividend, rolledFee)
			}
		}
	} else {
		// On startup, save one empty dividendPool for vite to diff db storage empty for serialize result
		rolledFee := &dexproto.FeeForDividend{}
		rolledFee.Token = ledger.ViteTokenId.Bytes()
		rolledDexFeesByPeriod.FeesForDividend = append(rolledDexFeesByPeriod.FeesForDividend, rolledFee)
	}
	SaveDexFeesLastPeriodIdForRoll(db, periodId)
	return
}

func MarkDexFeesFinishMine(db vm_db.VmDb, dexFeesByPeriod *DexFeesByPeriod, periodId uint64) {
	if dexFeesByPeriod.FinishDividend {
		setValueToDb(db, GetDexFeesKeyByPeriodId(periodId), nil)
	} else {
		dexFeesByPeriod.FinishMine = true
		serializeToDb(db, GetDexFeesKeyByPeriodId(periodId), dexFeesByPeriod)
	}
	if dexFeesByPeriod.LastValidPeriod > 0 {
		markFormerDexFeesFinishMine(db, dexFeesByPeriod.LastValidPeriod)
	}
}

func markFormerDexFeesFinishMine(db vm_db.VmDb, periodId uint64) {
	if dexFeesByPeriod, ok := GetDexFeesByPeriodId(db, periodId); ok {
		MarkDexFeesFinishMine(db, dexFeesByPeriod, periodId)
	}
}

func GetDexFeesKeyByPeriodId(periodId uint64) []byte {
	return append(dexFeesKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetDexFeesLastPeriodIdForRoll(db vm_db.VmDb) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, lastDexFeesPeriodIdKey); len(lastPeriodIdBytes) == 8 {
		return BytesToUint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveDexFeesLastPeriodIdForRoll(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastDexFeesPeriodIdKey, Uint64ToBytes(periodId))
}

func SaveCurrentOperatorFees(db vm_db.VmDb, reader util.ConsensusReader, operator []byte, operatorFeesByPeriod *OperatorFeesByPeriod) {
	serializeToDb(db, GetCurrentOperatorFeesKey(db, reader, operator), operatorFeesByPeriod)
}

func GetCurrentOperatorFees(db vm_db.VmDb, reader util.ConsensusReader, operator []byte) (*OperatorFeesByPeriod, bool) {
	return getOperatorFeesByKey(db, GetCurrentOperatorFeesKey(db, reader, operator))
}

func GetOperatorFeesByPeriodId(db vm_db.VmDb, operator []byte, periodId uint64) (*OperatorFeesByPeriod, bool) {
	return getOperatorFeesByKey(db, GetOperatorFeesKeyByPeriodIdAndAddress(periodId, operator))
}

func GetCurrentOperatorFeesKey(db vm_db.VmDb, reader util.ConsensusReader, operator []byte) []byte {
	return GetOperatorFeesKeyByPeriodIdAndAddress(GetCurrentPeriodId(db, reader), operator)
}

func DeleteOperatorFeesByKey(db vm_db.VmDb, key []byte) {
	setValueToDb(db, key, nil)
}

func getOperatorFeesByKey(db vm_db.VmDb, feeKey []byte) (*OperatorFeesByPeriod, bool) {
	operatorFeesByPeriod := &OperatorFeesByPeriod{}
	ok := deserializeFromDb(db, feeKey, operatorFeesByPeriod)
	return operatorFeesByPeriod, ok
}

func GetOperatorFeesKeyByPeriodIdAndAddress(periodId uint64, address []byte) []byte {
	return append(append(operatorFeesKeyPrefix, Uint64ToBytes(periodId)...), address...)
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

func TruncateUserFeesToPeriod(userFees *UserFees, periodId uint64) (truncated bool) {
	i := 0
	size := len(userFees.Fees)
	for {
		if i < size && userFees.Fees[i].Period < periodId {
			i++
		} else {
			break
		}
	}
	if i > 0 {
		userFees.Fees = userFees.Fees[i:]
		truncated = true
	}
	return
}

func IsValidFeeForMine(userFee *dexproto.FeeAccount, mineThreshold *big.Int) bool {
	return new(big.Int).Add(new(big.Int).SetBytes(userFee.BaseAmount), new(big.Int).SetBytes(userFee.InviteBonusAmount)).Cmp(mineThreshold) >= 0
}

func GetUserFeesKey(address []byte) []byte {
	return append(userFeeKeyPrefix, address...)
}

func GetVxFundsWithForkCheck(db vm_db.VmDb, address []byte) (vxFunds *VxFunds, ok bool) {
	if IsEarthFork(db) {
		return GetVxLockedFunds(db, address)
	} else {
		return GetVxFunds(db, address)
	}
}

func SaveVxFundsWithForkCheck(db vm_db.VmDb, address []byte, vxFunds *VxFunds) {
	if IsEarthFork(db) {
		SaveVxLockedFunds(db, address, vxFunds)
	} else {
		SaveVxFunds(db, address, vxFunds)
	}
}

func DeleteVxFundsWithForkCheck(db vm_db.VmDb, address []byte) {
	if IsEarthFork(db) {
		DeleteVxLockedFunds(db, address)
	} else {
		DeleteVxFunds(db, address)
	}
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
	return append(vxFundKeyPrefix, address...)
}

func GetVxLockedFunds(db vm_db.VmDb, address []byte) (vxFunds *VxFunds, ok bool) {
	vxFunds = &VxFunds{}
	ok = deserializeFromDb(db, GetVxLockedFundsKey(address), vxFunds)
	return
}

func SaveVxLockedFunds(db vm_db.VmDb, address []byte, vxFunds *VxFunds) {
	serializeToDb(db, GetVxLockedFundsKey(address), vxFunds)
}

func DeleteVxLockedFunds(db vm_db.VmDb, address []byte) {
	setValueToDb(db, GetVxLockedFundsKey(address), nil)
}

func GetVxLockedFundsKey(address []byte) []byte {
	return append(vxLockedFundsKeyPrefix, address...)
}

func SetAutoLockMinedVx(db vm_db.VmDb, address []byte, enable bool) {
	if enable {
		setValueToDb(db, GetVxAutoLockMinedVxKey(address), []byte{1})
	} else {
		setValueToDb(db, GetVxAutoLockMinedVxKey(address), nil)
	}
}

func IsAutoLockMinedVx(db vm_db.VmDb, address []byte) bool {
	return len(getValueFromDb(db, GetVxAutoLockMinedVxKey(address))) > 0
}

func GetVxAutoLockMinedVxKey(address []byte) []byte {
	return append(autoLockMinedVxKeyPrefix, address...)
}

func GetVxSumFundsWithForkCheck(db vm_db.VmDb) (vxSumFunds *VxFunds, ok bool) {
	if IsEarthFork(db) {
		return GetVxLockedSumFunds(db)
	} else {
		return GetVxSumFunds(db)
	}
}

func SaveVxSumFundsWithForkCheck(db vm_db.VmDb, vxSumFunds *VxFunds) {
	if IsEarthFork(db) {
		SaveVxLockedSumFunds(db, vxSumFunds)
	} else {
		SaveVxSumFunds(db, vxSumFunds)
	}
}

func GetVxSumFunds(db vm_db.VmDb) (vxSumFunds *VxFunds, ok bool) {
	vxSumFunds = &VxFunds{}
	ok = deserializeFromDb(db, vxSumFundsKey, vxSumFunds)
	return
}

func SaveVxSumFunds(db vm_db.VmDb, vxSumFunds *VxFunds) {
	serializeToDb(db, vxSumFundsKey, vxSumFunds)
}

func GetVxLockedSumFunds(db vm_db.VmDb) (vxLockedSumFunds *VxFunds, ok bool) {
	vxLockedSumFunds = &VxFunds{}
	ok = deserializeFromDb(db, vxLockedSumFundsKey, vxLockedSumFunds)
	return
}

func SaveVxLockedSumFunds(db vm_db.VmDb, vxLockedSumFunds *VxFunds) {
	serializeToDb(db, vxLockedSumFundsKey, vxLockedSumFunds)
}

func SaveMakerMiningPoolByPeriodId(db vm_db.VmDb, periodId uint64, amount *big.Int) {
	setValueToDb(db, GetMarkerMiningPoolByPeriodIdKey(periodId), amount.Bytes())
}

func GetMakerMiningPoolByPeriodId(db vm_db.VmDb, periodId uint64) *big.Int {
	if amtBytes := getValueFromDb(db, GetMarkerMiningPoolByPeriodIdKey(periodId)); len(amtBytes) > 0 {
		return new(big.Int).SetBytes(amtBytes)
	} else {
		return new(big.Int)
	}
}

func DeleteMakerMiningPoolByPeriodId(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, GetMarkerMiningPoolByPeriodIdKey(periodId), nil)
}

func GetMarkerMiningPoolByPeriodIdKey(periodId uint64) []byte {
	return append(makerMiningPoolByPeriodKey, Uint64ToBytes(periodId)...)
}

func GetLastJobPeriodIdByBizType(db vm_db.VmDb, bizType uint8) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, GetLastJobPeriodIdKey(bizType)); len(lastPeriodIdBytes) == 8 {
		return BytesToUint64(lastPeriodIdBytes)
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
		return BytesToUint64(firstMinedVxPeriodIdBytes)
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
	if pendingNewMarkets, ok := GetPendingNewMarkets(db); !ok {
		return nil, GetTokenInfoCallbackInnerConflictErr
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
					SavePendingNewMarkets(db, pendingNewMarkets)
					return quoteTokens, nil
				} else {
					DeletePendingNewMarkets(db)
					return quoteTokens, nil
				}
			}
		}
		return nil, GetTokenInfoCallbackInnerConflictErr
	}
}

func AddToPendingNewMarkets(db vm_db.VmDb, tradeToken, quoteToken types.TokenTypeId) error {
	pendingNewMarkets, _ := GetPendingNewMarkets(db)
	var foundTradeToken bool
	for _, action := range pendingNewMarkets.PendingActions {
		if bytes.Equal(action.TradeToken, tradeToken.Bytes()) {
			for _, qt := range action.QuoteTokens {
				if bytes.Equal(qt, quoteToken.Bytes()) {
					if IsStemFork(db) {
						return PendingNewMarketInnerConflictErr
					} else {
						panic(PendingNewMarketInnerConflictErr)
					}
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
	return nil
}

func GetPendingNewMarkets(db vm_db.VmDb) (pendingNewMarkets *PendingNewMarkets, ok bool) {
	pendingNewMarkets = &PendingNewMarkets{}
	ok = deserializeFromDb(db, pendingNewMarketActionsKey, pendingNewMarkets)
	return
}

func SavePendingNewMarkets(db vm_db.VmDb, pendingNewMarkets *PendingNewMarkets) {
	serializeToDb(db, pendingNewMarketActionsKey, pendingNewMarkets)
}

func DeletePendingNewMarkets(db vm_db.VmDb) {
	setValueToDb(db, pendingNewMarketActionsKey, nil)
}

//handle case on duplicate callback for getTokenInfo
func FilterPendingSetQuotes(db vm_db.VmDb, token types.TokenTypeId) (action *dexproto.SetQuoteTokenAction, err error) {
	if pendingSetQuotes, ok := GetPendingSetQuotes(db); !ok {
		return nil, GetTokenInfoCallbackInnerConflictErr
	} else {
		for index, action := range pendingSetQuotes.PendingActions {
			if bytes.Equal(action.Token, token.Bytes()) {
				actionsLen := len(pendingSetQuotes.PendingActions)
				if actionsLen > 1 {
					pendingSetQuotes.PendingActions[index] = pendingSetQuotes.PendingActions[actionsLen-1]
					pendingSetQuotes.PendingActions = pendingSetQuotes.PendingActions[:actionsLen-1]
					SavePendingSetQuotes(db, pendingSetQuotes)
					return action, nil
				} else {
					DeletePendingSetQuotes(db)
					return action, nil
				}
			}
		}
	}
	return nil, GetTokenInfoCallbackInnerConflictErr
}

func AddToPendingSetQuotes(db vm_db.VmDb, token types.TokenTypeId, quoteType uint8) {
	pendingSetQuotes, _ := GetPendingSetQuotes(db)
	action := &dexproto.SetQuoteTokenAction{}
	action.Token = token.Bytes()
	action.QuoteTokenType = int32(quoteType)
	pendingSetQuotes.PendingActions = append(pendingSetQuotes.PendingActions, action)
	SavePendingSetQuotes(db, pendingSetQuotes)
}

func SavePendingSetQuotes(db vm_db.VmDb, pendingSetQuotes *PendingSetQuotes) {
	serializeToDb(db, pendingSetQuoteActionsKey, pendingSetQuotes)
}

func GetPendingSetQuotes(db vm_db.VmDb) (pendingSetQuotes *PendingSetQuotes, ok bool) {
	pendingSetQuotes = &PendingSetQuotes{}
	ok = deserializeFromDb(db, pendingSetQuoteActionsKey, pendingSetQuotes)
	return
}

func DeletePendingSetQuotes(db vm_db.VmDb) {
	setValueToDb(db, pendingSetQuoteActionsKey, nil)
}

//handle case on duplicate callback for getTokenInfo
func FilterPendingTransferTokenOwners(db vm_db.VmDb, token types.TokenTypeId) (action *dexproto.TransferTokenOwnerAction, err error) {
	if pendings, ok := GetPendingTransferTokenOwners(db); !ok {
		return nil, GetTokenInfoCallbackInnerConflictErr
	} else {
		for index, action := range pendings.PendingActions {
			if bytes.Equal(action.Token, token.Bytes()) {
				actionsLen := len(pendings.PendingActions)
				if actionsLen > 1 {
					pendings.PendingActions[index] = pendings.PendingActions[actionsLen-1]
					pendings.PendingActions = pendings.PendingActions[:actionsLen-1]
					SavePendingTransferTokenOwners(db, pendings)
					return action, nil
				} else {
					DeletePendingTransferTokenOwners(db)
					return action, nil
				}
			}
		}
		return nil, GetTokenInfoCallbackInnerConflictErr
	}
}

func AddToPendingTransferTokenOwners(db vm_db.VmDb, token types.TokenTypeId, origin, new types.Address) {
	pendings, _ := GetPendingTransferTokenOwners(db)
	action := &dexproto.TransferTokenOwnerAction{}
	action.Token = token.Bytes()
	action.Origin = origin.Bytes()
	action.New = new.Bytes()
	pendings.PendingActions = append(pendings.PendingActions, action)
	SavePendingTransferTokenOwners(db, pendings)
}

func SavePendingTransferTokenOwners(db vm_db.VmDb, pendings *PendingTransferTokenOwnerActions) {
	serializeToDb(db, pendingTransferTokenOwnerActionsKey, pendings)
}

func GetPendingTransferTokenOwners(db vm_db.VmDb) (pendings *PendingTransferTokenOwnerActions, ok bool) {
	pendings = &PendingTransferTokenOwnerActions{}
	ok = deserializeFromDb(db, pendingTransferTokenOwnerActionsKey, pendings)
	return
}

func DeletePendingTransferTokenOwners(db vm_db.VmDb) {
	setValueToDb(db, pendingTransferTokenOwnerActionsKey, nil)
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
	serialNo := &SerialNo{}
	if ok := deserializeFromDb(db, orderIdSerialNoKey, serialNo); !ok {
		newSerialNo = 0
	} else {
		if timestamp == serialNo.Timestamp {
			newSerialNo = serialNo.No + 1
		} else {
			newSerialNo = 0
		}
	}
	serialNo.Timestamp = timestamp
	serialNo.No = newSerialNo
	serializeToDb(db, orderIdSerialNoKey, serialNo)
	return
}

func IsOwner(db vm_db.VmDb, address types.Address) bool {
	if storeOwner := getValueFromDb(db, ownerKey); len(storeOwner) == types.AddressSize {
		return bytes.Equal(storeOwner, address.Bytes())
	} else {
		return address == initOwner
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

func GetMakerMiningAdmin(db vm_db.VmDb) *types.Address {
	if mmaBytes := getValueFromDb(db, makerMiningAdminKey); len(mmaBytes) == types.AddressSize {
		if makerMiningAdmin, err := types.BytesToAddress(mmaBytes); err == nil {
			return &makerMiningAdmin
		} else {
			panic(err)
		}
	} else {
		panic(NotSetMakerMiningAdmin)
	}
}

func SaveMakerMiningAdmin(db vm_db.VmDb, addr types.Address) {
	setValueToDb(db, makerMiningAdminKey, addr.Bytes())
}

func IsMakerMiningAdmin(db vm_db.VmDb, addr types.Address) bool {
	if mmpBytes := getValueFromDb(db, makerMiningAdminKey); len(mmpBytes) == types.AddressSize {
		return bytes.Equal(addr.Bytes(), mmpBytes)
	} else {
		return false
	}
}

func GetLastSettledMakerMinedVxPeriod(db vm_db.VmDb) uint64 {
	if pIdBytes := getValueFromDb(db, lastSettledMakerMinedVxPeriodKey); len(pIdBytes) == 8 {
		return BytesToUint64(pIdBytes)
	} else {
		return 0
	}
}
func SaveLastSettledMakerMinedVxPeriod(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastSettledMakerMinedVxPeriodKey, Uint64ToBytes(periodId))
}

func GetLastSettledMakerMinedVxPage(db vm_db.VmDb) int32 {
	if pageBytes := getValueFromDb(db, lastSettledMakerMinedVxPageKey); len(pageBytes) == 4 {
		return int32(BytesToUint32(pageBytes))
	} else {
		return 0
	}
}
func SaveLastSettledMakerMinedVxPage(db vm_db.VmDb, pageId int32) {
	setValueToDb(db, lastSettledMakerMinedVxPageKey, Uint32ToBytes(uint32(pageId)))
}
func DeleteLastSettledMakerMinedVxPage(db vm_db.VmDb) {
	setValueToDb(db, lastSettledMakerMinedVxPageKey, nil)
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

func IsDexStopped(db vm_db.VmDb) bool {
	stopped := getValueFromDb(db, dexStoppedKey)
	return len(stopped) > 0
}

func SaveDexStopped(db vm_db.VmDb, isStopDex bool) {
	if isStopDex {
		setValueToDb(db, dexStoppedKey, []byte{1})
	} else {
		setValueToDb(db, dexStoppedKey, nil)
	}
}

func ValidTimeOracle(db vm_db.VmDb, address types.Address) bool {
	if timeOracleBytes := getValueFromDb(db, timeOracleKey); len(timeOracleBytes) == types.AddressSize {
		return bytes.Equal(timeOracleBytes, address.Bytes())
	}
	return false
}

func GetTimeOracle(db vm_db.VmDb) *types.Address {
	if timeOracleBytes := getValueFromDb(db, timeOracleKey); len(timeOracleBytes) == types.AddressSize {
		address, _ := types.BytesToAddress(timeOracleBytes)
		return &address
	} else {
		return nil
	}
}

func SetTimeOracle(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, timeOracleKey, address.Bytes())
}

func ValidTriggerAddress(db vm_db.VmDb, address types.Address) bool {
	if triggerAddressBytes := getValueFromDb(db, periodJobTriggerKey); len(triggerAddressBytes) == types.AddressSize {
		return bytes.Equal(triggerAddressBytes, address.Bytes())
	}
	return false
}

func GetPeriodJobTrigger(db vm_db.VmDb) *types.Address {
	if triggerAddressBytes := getValueFromDb(db, periodJobTriggerKey); len(triggerAddressBytes) == types.AddressSize {
		address, _ := types.BytesToAddress(triggerAddressBytes)
		return &address
	} else {
		return nil
	}
}

func SetPeriodJobTrigger(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, periodJobTriggerKey, address.Bytes())
}

func GetMiningStakedAmount(db vm_db.VmDb, address types.Address) *big.Int {
	if bs := getValueFromDb(db, GetMiningStakedAmountKey(address)); len(bs) > 0 {
		return new(big.Int).SetBytes(bs)
	} else {
		return big.NewInt(0)
	}
}

func SaveMiningStakedAmount(db vm_db.VmDb, address types.Address, amount *big.Int) {
	setValueToDb(db, GetMiningStakedAmountKey(address), amount.Bytes())
}

func DeleteMiningStakedAmount(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetMiningStakedAmountKey(address), nil)
}

func GetMiningStakedAmountKey(address types.Address) []byte {
	return append(miningStakedAmountKeyPrefix, address.Bytes()...)
}

func GetMiningStakedV2Amount(db vm_db.VmDb, address types.Address) *big.Int {
	if bs := getValueFromDb(db, GetMiningStakedV2AmountKey(address)); len(bs) > 0 {
		return new(big.Int).SetBytes(bs)
	} else {
		return big.NewInt(0)
	}
}

func SaveMiningStakedV2Amount(db vm_db.VmDb, address types.Address, amount *big.Int) {
	setValueToDb(db, GetMiningStakedV2AmountKey(address), amount.Bytes())
}

func DeleteMiningStakedV2Amount(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetMiningStakedV2AmountKey(address), nil)
}

func GetMiningStakedV2AmountKey(address types.Address) []byte {
	return append(miningStakedAmountV2KeyPrefix, address.Bytes()...)
}

func GetVIPStaking(db vm_db.VmDb, address types.Address) (vipStaking *VIPStaking, ok bool) {
	vipStaking = &VIPStaking{}
	ok = deserializeFromDb(db, GetVIPStakingKey(address), vipStaking)
	return
}

func SaveVIPStaking(db vm_db.VmDb, address types.Address, vipStaking *VIPStaking) {
	serializeToDb(db, GetVIPStakingKey(address), vipStaking)
}

func DeleteVIPStaking(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetVIPStakingKey(address), nil)
}

func GetVIPStakingKey(address types.Address) []byte {
	return append(vipStakingKeyPrefix, address.Bytes()...)
}

func GetSuperVIPStaking(db vm_db.VmDb, address types.Address) (superVIPStaking *VIPStaking, ok bool) {
	superVIPStaking = &VIPStaking{}
	ok = deserializeFromDb(db, GetSuperVIPStakingKey(address), superVIPStaking)
	return
}

func SaveSuperVIPStaking(db vm_db.VmDb, address types.Address, superVIPStaking *VIPStaking) {
	serializeToDb(db, GetSuperVIPStakingKey(address), superVIPStaking)
}

func DeleteSuperVIPStaking(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetSuperVIPStakingKey(address), nil)
}

func GetSuperVIPStakingKey(address types.Address) []byte {
	return append(superVIPStakingKeyPrefix, address.Bytes()...)
}

func ReduceVipStakingHash(stakings *VIPStaking, hash types.Hash) bool {
	var found bool
	for i, hs := range stakings.StakingHashes {
		if bytes.Equal(hs, hash.Bytes()) {
			size := len(stakings.StakingHashes)
			if i != size-1 {
				stakings.StakingHashes[i] = stakings.StakingHashes[size-1]
			}
			stakings.StakingHashes = stakings.StakingHashes[:size-1]
			found = true
			break
		}
	}
	return found
}

func GetMiningStakings(db vm_db.VmDb, address types.Address) (miningStakings *MiningStakings, ok bool) {
	miningStakings = &MiningStakings{}
	ok = deserializeFromDb(db, GetMiningStakingsKey(address), miningStakings)
	return
}

func SaveMiningStakings(db vm_db.VmDb, address types.Address, ps *MiningStakings) {
	serializeToDb(db, GetMiningStakingsKey(address), ps)
}

func DeleteMiningStakings(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, GetMiningStakingsKey(address), nil)
}

func GetMiningStakingsKey(address types.Address) []byte {
	return append(miningStakingsKeyPrefix, address.Bytes()...)
}

func GetDexMiningStakings(db vm_db.VmDb) (dexMiningStakings *MiningStakings, ok bool) {
	dexMiningStakings = &MiningStakings{}
	ok = deserializeFromDb(db, dexMiningStakingsKey, dexMiningStakings)
	return
}

func SaveDexMiningStakings(db vm_db.VmDb, ms *MiningStakings) {
	serializeToDb(db, dexMiningStakingsKey, ms)
}

func IsValidMiningStakeAmount(amount *big.Int) bool {
	return amount.Cmp(StakeForMiningThreshold) >= 0
}

func IsValidMiningStakeAmountBytes(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(StakeForMiningThreshold) >= 0
}

func MatchMiningStakingByPeriod(miningStakings *MiningStakings, periodId uint64, checkDelete bool) (bool, []byte, bool, bool) {
	var (
		stakedAmtBytes           []byte
		matchIndex               int
		needUpdateMiningStakings bool
	)
	for i, staking := range miningStakings.Stakings {
		if periodId >= staking.Period {
			stakedAmtBytes = staking.Amount
			matchIndex = i
		} else {
			break
		}
	}
	if len(stakedAmtBytes) == 0 {
		return false, nil, false, checkDelete && CheckMiningStakingsCanBeDelete(miningStakings)
	}
	if matchIndex > 0 { //remove obsolete items, but leave current matched item
		miningStakings.Stakings = miningStakings.Stakings[matchIndex:]
		needUpdateMiningStakings = true
	}
	if len(miningStakings.Stakings) > 1 && miningStakings.Stakings[1].Period == periodId+1 {
		miningStakings.Stakings = miningStakings.Stakings[1:]
		needUpdateMiningStakings = true
	}
	return true, stakedAmtBytes, needUpdateMiningStakings, checkDelete && CheckMiningStakingsCanBeDelete(miningStakings)
}

func CheckMiningStakingsCanBeDelete(miningStakings *MiningStakings) bool {
	return len(miningStakings.Stakings) == 1 && !IsValidMiningStakeAmountBytes(miningStakings.Stakings[0].Amount)
}

func GetDelegateStakeInfo(db vm_db.VmDb, hash []byte) (info *DelegateStakeInfo, ok bool) {
	info = &DelegateStakeInfo{}
	ok = deserializeFromDb(db, GetDelegateStakeInfoKey(hash), info)
	return
}

func SaveDelegateStakeInfo(db vm_db.VmDb, hash types.Hash, stakeType uint8, address, principal types.Address, amount *big.Int) {
	info := &DelegateStakeInfo{}
	info.StakeType = int32(stakeType)
	info.Address = address.Bytes()
	if principal != types.ZERO_ADDRESS {
		info.Principal = principal.Bytes()
	}
	info.Amount = amount.Bytes()
	info.Status = StakeSubmitted
	serializeToDb(db, GetDelegateStakeInfoKey(hash.Bytes()), info)
}

func ConfirmDelegateStakeInfo(db vm_db.VmDb, hash types.Hash, info *DelegateStakeInfo, serialNo uint64) {
	info.Status = int32(StakeConfirmed)
	info.SerialNo = serialNo
	serializeToDb(db, GetDelegateStakeInfoKey(hash.Bytes()), info)
}

func DeleteDelegateStakeInfo(db vm_db.VmDb, hash []byte) {
	setValueToDb(db, GetDelegateStakeInfoKey(hash), nil)
}

func GetDelegateStakeInfoKey(hash []byte) []byte {
	return append(delegateStakeInfoPrefix, hash[len(delegateStakeInfoPrefix):]...)
}

func SaveDelegateStakeAddressIndex(db vm_db.VmDb, id types.Hash, stakeType int32, address []byte) uint64 {
	serialNo := NewDelegateStakeIndexSerialNo(db, address)
	index := &DelegateStakeAddressIndex{}
	index.StakeType = int32(stakeType)
	index.Id = id.Bytes()
	serializeToDb(db, GetDelegateStakeAddressIndexKey(address, serialNo), index)
	return serialNo
}

func DeleteDelegateStakeAddressIndex(db vm_db.VmDb, address []byte, serialNo uint64) {
	setValueToDb(db, GetDelegateStakeAddressIndexKey(address, serialNo), nil)
}

func GetDelegateStakeAddressIndexKey(address []byte, serialNo uint64) []byte {
	return append(append(delegateStakeAddressIndexPrefix, address...), Uint64ToBytes(serialNo)...) //4+20+8
}

func NewDelegateStakeIndexSerialNo(db vm_db.VmDb, address []byte) (serialNo uint64) {
	if data := getValueFromDb(db, GetDelegateStakeIndexSerialNoKey(address)); len(data) == 8 {
		serialNo = BytesToUint64(data)
		serialNo++
	} else {
		serialNo = 1
	}
	setValueToDb(db, GetDelegateStakeIndexSerialNoKey(address), Uint64ToBytes(serialNo))
	return
}

func GetDelegateStakeIndexSerialNoKey(address []byte) []byte {
	return append(delegateStakeIndexSerialNoPrefix, address...)
}

func GetTimestampInt64(db vm_db.VmDb) int64 {
	timestamp := GetDexTimestamp(db)
	if timestamp == 0 {
		panic(NotSetTimestampErr)
	} else {
		return timestamp
	}
}

func SetDexTimestamp(db vm_db.VmDb, timestamp int64, reader util.ConsensusReader) error {
	oldTime := GetDexTimestamp(db)
	if timestamp > oldTime {
		oldPeriod := GetPeriodIdByTimestamp(reader, oldTime)
		newPeriod := GetPeriodIdByTimestamp(reader, timestamp)
		if newPeriod != oldPeriod {
			doRollPeriod(db, newPeriod)
		}
		setValueToDb(db, dexTimestampKey, Uint64ToBytes(uint64(timestamp)))
		return nil
	} else {
		return InvalidTimestampFromTimeOracleErr
	}
}

func doRollPeriod(db vm_db.VmDb, newPeriodId uint64) {
	newDexFeesByPeriod := RollAndGentNewDexFeesByPeriod(db, newPeriodId)
	SaveDexFeesByPeriodId(db, newPeriodId, newDexFeesByPeriod)
}

func GetDexTimestamp(db vm_db.VmDb) int64 {
	if bs := getValueFromDb(db, dexTimestampKey); len(bs) == 8 {
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

func GetInviterByCode(db vm_db.VmDb, inviteCode uint32) (*types.Address, error) {
	if bs := getValueFromDb(db, append(inviterByCodeKeyPrefix, Uint32ToBytes(inviteCode)...)); len(bs) == types.AddressSize {
		if inviter, err := types.BytesToAddress(bs); err != nil {
			return nil, err
		} else {
			return &inviter, nil
		}
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

func GetInviterByInvitee(db vm_db.VmDb, invitee types.Address) (*types.Address, error) {
	if bs := getValueFromDb(db, append(inviterByInviteeKeyPrefix, invitee.Bytes()...)); len(bs) == types.AddressSize {
		if inviter, err := types.BytesToAddress(bs); err != nil {
			return nil, err
		} else {
			return &inviter, nil
		}
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

func StartNormalMine(db vm_db.VmDb) {
	setValueToDb(db, normalMineStartedKey, []byte{1})
}

func IsNormalMiningStarted(db vm_db.VmDb) bool {
	return len(getValueFromDb(db, normalMineStartedKey)) > 0
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

func GetVxBurnAmount(db vm_db.VmDb) *big.Int {
	if data := getValueFromDb(db, vxBurnAmountKey); len(data) > 0 {
		return new(big.Int).SetBytes(data)
	} else {
		return new(big.Int)
	}
}

func SaveVxBurnAmount(db vm_db.VmDb, amount *big.Int) {
	setValueToDb(db, vxBurnAmountKey, amount.Bytes())
}

func GrantMarketToAgent(db vm_db.VmDb, principal, agent types.Address, marketId int32) {
	var (
		key   = GetGrantedMarketToAgentKey(principal, marketId)
		data  []byte
		found bool
	)
	if data = getValueFromDb(db, key); len(data) >= types.AddressSize {
		dataLen := len(data)
		if dataLen%types.AddressSize != 0 {
			panic(InternalErr)
		}
		for i := 0; (i+1)*types.AddressSize <= dataLen; i++ {
			if bytes.Equal(agent.Bytes(), data[i*types.AddressSize:(i+1)*types.AddressSize]) {
				found = true
				break
			}
		}
	}
	if !found {
		data = append(data, agent.Bytes()...)
		setValueToDb(db, key, data)
	}
}

func RevokeMarketFromAgent(db vm_db.VmDb, principal, agent types.Address, marketId int32) {
	var (
		key  = GetGrantedMarketToAgentKey(principal, marketId)
		data []byte
	)
	if data = getValueFromDb(db, key); len(data) >= types.AddressSize {
		dataLen := len(data)
		if dataLen%types.AddressSize != 0 {
			panic(InternalErr)
		}
		for i := 0; (i+1)*types.AddressSize <= dataLen; i++ {
			if bytes.Equal(agent.Bytes(), data[i*types.AddressSize:(i+1)*types.AddressSize]) {
				if (i+1)*types.AddressSize < dataLen {
					copy(data[i*types.AddressSize:], data[dataLen-types.AddressSize:])
				}
				data = data[:dataLen-types.AddressSize]
				break
			}
		}
		setValueToDb(db, key, data)
	}
}

func IsMarketGrantedToAgent(db vm_db.VmDb, principal, agent types.Address, marketId int32) bool {
	if data := getValueFromDb(db, GetGrantedMarketToAgentKey(principal, marketId)); len(data) >= types.AddressSize {
		if len(data)%types.AddressSize != 0 {
			panic(InternalErr)
		}
		for i := 0; (i+1)*types.AddressSize <= len(data); i++ {
			if bytes.Equal(agent.Bytes(), data[i*types.AddressSize:(i+1)*types.AddressSize]) {
				return true
			}
		}
	}
	return false
}

func GetGrantedMarketToAgentKey(principal types.Address, marketId int32) []byte {
	re := make([]byte, len(grantedMarketToAgentKeyPrefix)+types.AddressSize+3)
	copy(re[:], grantedMarketToAgentKeyPrefix)
	copy(re[len(grantedMarketToAgentKeyPrefix):], principal.Bytes())
	copy(re[len(grantedMarketToAgentKeyPrefix)+types.AddressSize:], Uint32ToBytes(uint32(marketId))[1:])
	return re
}

func GetVxUnlocks(db vm_db.VmDb, address types.Address) (unlocks *VxUnlocks, ok bool) {
	unlocks = &VxUnlocks{}
	ok = deserializeFromDb(db, GetVxUnlocksKey(address), unlocks)
	return
}

func AddVxUnlock(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount *big.Int) {
	unlocks := &VxUnlocks{}
	currentPeriodId := GetCurrentPeriodId(db, reader)
	var updated bool
	if ok := deserializeFromDb(db, GetVxUnlocksKey(address), unlocks); ok {
		size := len(unlocks.Unlocks)
		if unlocks.Unlocks[size-1].PeriodId == currentPeriodId {
			unlocks.Unlocks[size-1].Amount = AddBigInt(unlocks.Unlocks[size-1].Amount, amount.Bytes())
			updated = true
		}
	}
	if !updated {
		newUnlock := &dexproto.VxUnlock{}
		newUnlock.Amount = amount.Bytes()
		newUnlock.PeriodId = currentPeriodId
		unlocks.Unlocks = append(unlocks.Unlocks, newUnlock)
	}
	serializeToDb(db, GetVxUnlocksKey(address), unlocks)
}

func UpdateVxUnlocks(db vm_db.VmDb, address types.Address, unlocks *VxUnlocks) {
	if len(unlocks.Unlocks) == 0 {
		setValueToDb(db, GetVxUnlocksKey(address), nil)
	} else {
		serializeToDb(db, GetVxUnlocksKey(address), unlocks)
	}
}

func GetVxUnlocksKey(address types.Address) []byte {
	return append(vxUnlocksKeyPrefix, address.Bytes()...)
}

func GetCancelStakes(db vm_db.VmDb, address types.Address) (cancelStakes *CancelStakes, ok bool) {
	cancelStakes = &CancelStakes{}
	ok = deserializeFromDb(db, GetCancelStakesKey(address), cancelStakes)
	return
}

func AddCancelStake(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount *big.Int) {
	cancelStakes := &CancelStakes{}
	currentPeriodId := GetCurrentPeriodId(db, reader)
	var updated bool
	if ok := deserializeFromDb(db, GetCancelStakesKey(address), cancelStakes); ok {
		size := len(cancelStakes.Cancels)
		if cancelStakes.Cancels[size-1].PeriodId == currentPeriodId {
			cancelStakes.Cancels[size-1].Amount = AddBigInt(cancelStakes.Cancels[size-1].Amount, amount.Bytes())
			updated = true
		}
	}
	if !updated {
		newCancel := &dexproto.CancelStake{}
		newCancel.Amount = amount.Bytes()
		newCancel.PeriodId = currentPeriodId
		cancelStakes.Cancels = append(cancelStakes.Cancels, newCancel)
	}
	serializeToDb(db, GetCancelStakesKey(address), cancelStakes)
}

func UpdateCancelStakes(db vm_db.VmDb, address types.Address, cancelStakes *CancelStakes) {
	if len(cancelStakes.Cancels) == 0 {
		setValueToDb(db, GetCancelStakesKey(address), nil)
	} else {
		serializeToDb(db, GetCancelStakesKey(address), cancelStakes)
	}
}

func GetCancelStakesKey(address types.Address) []byte {
	return append(cancelStakesKeyPrefix, address.Bytes()...)
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
