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
	viteXStoppedKey         = []byte("dexStp:")
	fundKeyPrefix           = []byte("fd:") // fund:types.Address
	minTradeAmountKeyPrefix = []byte("mTrAt:")
	mineThresholdKeyPrefix  = []byte("mTh:")

	timestampKey = []byte("tts:") // timerTimestamp

	userFeeKeyPrefix = []byte("uF:") // userFee:types.Address

	feeSumKeyPrefix     = []byte("fS:")     // feeSum:periodId
	lastFeeSumPeriodKey = []byte("lFSPId:") //

	operatorFeesKeyPrefix = []byte("bf:") // operatorFees:periodId, 32 bytes prefix[3] + periodId[8]+ address[21]

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
	normalMineStartedKey          = []byte("nmst:")
	firstMinedVxPeriodIdKey       = []byte("fMVPId:")
	marketInfoKeyPrefix           = []byte("mk:") // market: tradeToke,quoteToken

	vipStakingKeyPrefix      = []byte("pldVip:")   // vipStaking: types.Address
	superVIPStakingKeyPrefix = []byte("pldSpVip:") // superVIPStaking: types.Address

	miningStakingsKeyPrefix     = []byte("pldsVx:")  // miningStakings: types.Address
	dexMiningStakingsKey        = []byte("pldsVxS:") // dexMiningStakings
	miningStakedAmountKeyPrefix = []byte("pldVx:")   // miningStakedAmount: types.Address

	tokenInfoKeyPrefix = []byte("tk:") // token:tokenId
	vxMinePoolKey      = []byte("vxmPl:")

	codeByInviterKeyPrefix    = []byte("itr2cd:")
	inviterByCodeKeyPrefix    = []byte("cd2itr:")
	inviterByInviteeKeyPrefix = []byte("ite2itr:")

	maintainerKey                    = []byte("mtA:")
	makerMineProxyKey                = []byte("mmpA:")
	makerMineProxyAmountByPeriodKey  = []byte("mmpaP:")
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

	StakeForVxMinAmount    = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	StakeForVIPAmount      = new(big.Int).Mul(commonTokenPow, big.NewInt(10000))
	StakeForVxThreshold    = new(big.Int).Mul(commonTokenPow, big.NewInt(134))
	StakeForSuperVIPAmount = new(big.Int).Mul(commonTokenPow, big.NewInt(1000000))

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
	StakeForVx = iota + 1
	StakeForVIP
	StakeForSuperVIP
)

const (
	Stake = iota + 1
	CancelStake
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
	OwnerConfigMineMarket      = 1
	OwnerConfigNewQuoteToken   = 2
	OwnerConfigTradeThreshold  = 4
	OwnerConfigMineThreshold   = 8
	OwnerConfigStartNormalMine = 16
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

type ParamDexFundNewAgentOrder struct {
	Principal types.Address
	ParamDexFundNewOrder
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

type ParamDexFundStakeForMining struct {
	ActionType uint8 // 1: stake 2: cancel stake
	Amount     *big.Int
}

type ParamDexFundStakeForVIP struct {
	ActionType uint8 // 1: stake 2: cancel stake
}

type ParamDexFundStakeCallBack struct {
	StakeAddr   types.Address
	Beneficiary types.Address
	Amount      *big.Int
	Bid         uint8
	Success     bool
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

type ParamDexFundDexAdminConfig struct {
	OperationCode    uint8
	Owner            types.Address // 1 owner
	TimeOracle       types.Address // 2 timeOracle
	PeriodJobTrigger types.Address // 4 periodJobTrigger
	StopDex          bool          // 8 stopDex
	MakerMiningAdmin types.Address // 16 maker mining admin
	Maintainer       types.Address // 32 maintainer
}

type ParamDexFundTradeAdminConfig struct {
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

type ParamDexFundMarketOwnerConfig struct {
	OperationCode uint8 // 1 owner, 2 takerRate, 4 makerRate, 8 stopMarket
	TradeToken    types.TokenTypeId
	QuoteToken    types.TokenTypeId
	MarketOwner   types.Address
	TakerFeeRate  int32
	MakerFeeRate  int32
	StopMarket    bool
}

type ParamDexFundTransferTokenOwnership struct {
	Token    types.TokenTypeId
	NewOwner types.Address
}

type ParamDexFundNotifyTime struct {
	Timestamp int64
}

type ParamDexFundConfigMarketAgents struct {
	ActionType  uint8 // 1: grant 2: revoke
	Agent       types.Address
	TradeTokens []types.TokenTypeId
	QuoteTokens []types.TokenTypeId
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

type DexFeesByPeriod struct {
	dexproto.DexFeesByPeriod
}

func (df *DexFeesByPeriod) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.DexFeesByPeriod)
}

func (df *DexFeesByPeriod) DeSerialize(feeSumData []byte) (err error) {
	protoFeeSum := dexproto.DexFeesByPeriod{}
	if err := proto.Unmarshal(feeSumData, &protoFeeSum); err != nil {
		return err
	} else {
		df.DexFeesByPeriod = protoFeeSum
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

func (psv *MiningStakings) Serialize() (data []byte, err error) {
	return proto.Marshal(&psv.MiningStakings)
}

func (psv *MiningStakings) DeSerialize(data []byte) error {
	miningStakings := dexproto.MiningStakings{}
	if err := proto.Unmarshal(data, &miningStakings); err != nil {
		return err
	} else {
		psv.MiningStakings = miningStakings
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

func SaveFund(db vm_db.VmDb, address types.Address, dexFund *Fund) {
	serializeToDb(db, GetFundKey(address), dexFund)
}

func ReduceAccount(db vm_db.VmDb, address types.Address, tokenId []byte, amount *big.Int) (updatedAcc *dexproto.Account, err error) {
	if fund, ok := GetFund(db, address); ok {
		var foundAcc bool
		for _, acc := range fund.Accounts {
			if bytes.Equal(acc.Token, tokenId) {
				foundAcc = true
				available := new(big.Int).SetBytes(acc.Available)
				if available.Cmp(amount) < 0 {
					err = ExceedFundAvailableErr
					return
				} else {
					acc.Available = available.Sub(available, amount).Bytes()
				}
				updatedAcc = acc
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

func UpdateFund(db vm_db.VmDb, address types.Address, accounts map[types.TokenTypeId]*big.Int) error {
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

func GetCurrentFeeSum(db vm_db.VmDb, reader util.ConsensusReader) (*DexFeesByPeriod, bool) {
	return getFeeSumByKey(db, GetFeeSumKeyByPeriodId(GetCurrentPeriodId(db, reader)))
}

func GetDexFeesByPeriodId(db vm_db.VmDb, periodId uint64) (*DexFeesByPeriod, bool) {
	return getFeeSumByKey(db, GetFeeSumKeyByPeriodId(periodId))
}

func getFeeSumByKey(db vm_db.VmDb, feeKey []byte) (*DexFeesByPeriod, bool) {
	feeSum := &DexFeesByPeriod{}
	ok := deserializeFromDb(db, feeKey, feeSum)
	return feeSum, ok
}

//get all feeSums that not divided yet
func GetNotDividedFeeSumsByPeriodId(db vm_db.VmDb, periodId uint64) map[uint64]*DexFeesByPeriod {
	var (
		dexFeeSums    = make(map[uint64]*DexFeesByPeriod)
		dexFeeSum     *DexFeesByPeriod
		ok, everFound bool
	)
	for {
		if dexFeeSum, ok = GetDexFeesByPeriodId(db, periodId); !ok { // found first valid period
			if periodId > 0 && !everFound {
				periodId--
				continue
			} else { // lastValidPeriod is delete
				return dexFeeSums
			}
		} else {
			everFound = true
			if !dexFeeSum.FinishDividend {
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

func SaveFeeSumWithPeriodId(db vm_db.VmDb, periodId uint64, feeSum *DexFeesByPeriod) {
	serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
}

//fee sum used both by fee dividend and mined vx dividend
func MarkFeeSumAsFeeDivided(db vm_db.VmDb, feeSum *DexFeesByPeriod, periodId uint64) {
	if feeSum.FinishMine {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FinishDividend = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
}

func RollAndGentNewFeeSumByPeriod(db vm_db.VmDb, periodId uint64) (rolledFeeSumByPeriod *DexFeesByPeriod) {
	formerId := GetDexFeesLastPeriodIdForRoll(db)
	rolledFeeSumByPeriod = &DexFeesByPeriod{}
	if formerId > 0 {
		if formerDexFeesByPeriod, ok := GetDexFeesByPeriodId(db, formerId); !ok { // lastPeriod has been deleted on fee dividend
			panic(NoFeeSumFoundForValidPeriodErr)
		} else {
			rolledFeeSumByPeriod.LastValidPeriod = formerId
			for _, feesForDividend := range formerDexFeesByPeriod.FeesForDividend {
				rolledFee := &dexproto.FeesForDividend{}
				rolledFee.Token = feesForDividend.Token
				_, rolledAmount := splitDividendPool(feesForDividend)
				rolledFee.DividendPoolAmount = rolledAmount.Bytes()
				rolledFeeSumByPeriod.FeesForDividend = append(rolledFeeSumByPeriod.FeesForDividend, rolledFee)
			}
		}
	} else {
		// On startup, save one empty dividendPool for vite to diff db storage empty for serialize result
		rolledFee := &dexproto.FeesForDividend{}
		rolledFee.Token = ledger.ViteTokenId.Bytes()
		rolledFeeSumByPeriod.FeesForDividend = append(rolledFeeSumByPeriod.FeesForDividend, rolledFee)
	}
	SaveFeeSumLastPeriodIdForRoll(db, periodId)
	return
}

func MarkFeeSumAsMinedVxDivided(db vm_db.VmDb, feeSum *DexFeesByPeriod, periodId uint64) {
	if feeSum.FinishDividend {
		setValueToDb(db, GetFeeSumKeyByPeriodId(periodId), nil)
	} else {
		feeSum.FinishMine = true
		serializeToDb(db, GetFeeSumKeyByPeriodId(periodId), feeSum)
	}
	if feeSum.LastValidPeriod > 0 {
		markFormerFeeSumsAsMined(db, feeSum.LastValidPeriod)
	}
}

func markFormerFeeSumsAsMined(db vm_db.VmDb, periodId uint64) {
	if feeSum, ok := GetDexFeesByPeriodId(db, periodId); ok {
		MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)
	}
}

func GetFeeSumKeyByPeriodId(periodId uint64) []byte {
	return append(feeSumKeyPrefix, Uint64ToBytes(periodId)...)
}

func GetDexFeesLastPeriodIdForRoll(db vm_db.VmDb) uint64 {
	if lastPeriodIdBytes := getValueFromDb(db, lastFeeSumPeriodKey); len(lastPeriodIdBytes) == 8 {
		return BytesToUint64(lastPeriodIdBytes)
	} else {
		return 0
	}
}

func SaveFeeSumLastPeriodIdForRoll(db vm_db.VmDb, periodId uint64) {
	setValueToDb(db, lastFeeSumPeriodKey, Uint64ToBytes(periodId))
}

func SaveCurrentOperatorFees(db vm_db.VmDb, reader util.ConsensusReader, operator []byte, operatorFeesByPeriod *OperatorFeesByPeriod) {
	serializeToDb(db, GetCurrentOperatorFeesKey(db, reader, operator), operatorFeesByPeriod)
}

func GetCurrentOperatorFees(db vm_db.VmDb, reader util.ConsensusReader, operator []byte) (*OperatorFeesByPeriod, bool) {
	return getOperatorFeesByKey(db, GetCurrentOperatorFeesKey(db, reader, operator))
}

func GetOperatorFeesSumByPeriodId(db vm_db.VmDb, operator []byte, periodId uint64) (*OperatorFeesByPeriod, bool) {
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

func StartNormalMine(db vm_db.VmDb) {
	setValueToDb(db, normalMineStartedKey, []byte{1})
}

func IsNormalMineStarted(db vm_db.VmDb) bool {
	return len(getValueFromDb(db, normalMineStartedKey)) > 0
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
	if timerAddressBytes := getValueFromDb(db, timerAddressKey); len(timerAddressBytes) == types.AddressSize {
		return bytes.Equal(timerAddressBytes, address.Bytes())
	}
	return false
}

func GetTimer(db vm_db.VmDb) *types.Address {
	if timerAddressBytes := getValueFromDb(db, timerAddressKey); len(timerAddressBytes) == types.AddressSize {
		address, _ := types.BytesToAddress(timerAddressBytes)
		return &address
	} else {
		return nil
	}
}

func SetTimerAddress(db vm_db.VmDb, address types.Address) {
	setValueToDb(db, timerAddressKey, address.Bytes())
}

func ValidTriggerAddress(db vm_db.VmDb, address types.Address) bool {
	if triggerAddressBytes := getValueFromDb(db, triggerAddressKey); len(triggerAddressBytes) == types.AddressSize {
		return bytes.Equal(triggerAddressBytes, address.Bytes())
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

func SaveDexMiningStakings(db vm_db.VmDb, ps *MiningStakings) {
	serializeToDb(db, dexMiningStakingsKey, ps)
}

func IsValidMiningStakeAmount(amount *big.Int) bool {
	return amount.Cmp(StakeForVxThreshold) >= 0
}

func IsValidMiningStakeAmountBytes(amount []byte) bool {
	return new(big.Int).SetBytes(amount).Cmp(StakeForVxThreshold) >= 0
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
	SaveFeeSumWithPeriodId(db, newPeriodId, newFeeSum)
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
