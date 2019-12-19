package api

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	apidex "github.com/vitelabs/go-vite/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"strconv"
)

type DexApi struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDexApi(vite *vite.Vite) *DexApi {
	return &DexApi{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dex_api"),
	}
}

func (f DexApi) String() string {
	return "DexApi"
}

type AccountBalanceInfo struct {
	TokenInfo       *RpcTokenInfo `json:"tokenInfo,omitempty"`
	Available       string        `json:"available"`
	Locked          string        `json:"locked"`
	VxLocked        string        `json:"vxLocked,omitempty"`
	VxUnlocking     string        `json:"vxUnlocking,omitempty"`
	CancellingStake string        `json:"cancellingStake,omitempty"`
}

func (f DexApi) GetAccountBalanceInfo(addr types.Address, tokenId *types.TokenTypeId) (map[types.TokenTypeId]*AccountBalanceInfo, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	fund, _ := dex.GetFund(db, addr)
	accounts, err := dex.GetAccounts(fund, tokenId)
	if err != nil {
		return nil, err
	}

	balanceInfo := make(map[types.TokenTypeId]*AccountBalanceInfo, 0)
	for _, v := range accounts {
		tokenInfo, err := f.chain.GetTokenInfoById(v.Token)
		if err != nil {
			return nil, err
		}
		info := &AccountBalanceInfo{TokenInfo: RawTokenInfoToRpc(tokenInfo, v.Token)}
		if v.Available != nil {
			info.Available = v.Available.String()
		} else {
			info.Available = "0"
		}

		if v.Locked != nil {
			info.Locked = v.Locked.String()
		} else {
			info.Locked = "0"
		}

		if v.Token == dex.VxTokenId {
			if v.VxLocked != nil {
				info.VxLocked = v.VxLocked.String()
			}
			if v.VxUnlocking != nil {
				info.VxUnlocking = v.VxUnlocking.String()
			}
		}

		if v.Token == ledger.ViteTokenId && v.CancellingStake != nil {
			info.CancellingStake = v.CancellingStake.String()
		}
		balanceInfo[v.Token] = info
	}
	return balanceInfo, nil
}

func (f DexApi) GetTokenInfo(token types.TokenTypeId) (*apidex.RpcDexTokenInfo, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if tokenInfo, ok := dex.GetTokenInfo(db, token); ok {
		return apidex.TokenInfoToRpc(tokenInfo, token), nil
	} else {
		return nil, dex.InvalidTokenErr
	}
}

func (f DexApi) GetMarketInfo(tradeToken, quoteToken types.TokenTypeId) (*apidex.NewRpcMarketInfo, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, quoteToken); ok {
		return apidex.MarketInfoToNewRpc(marketInfo), nil
	} else {
		return nil, dex.TradeMarketNotExistsErr
	}
}

func (f DexApi) GetDividendPoolsInfo() (map[types.TokenTypeId]*apidex.DividendPoolInfo, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	var pools map[types.TokenTypeId]*apidex.DividendPoolInfo
	if dexFeesByPeriod, ok := dex.GetCurrentDexFees(db, apidex.GetConsensusReader(f.vite)); !ok {
		return nil, nil
	} else {
		pools = make(map[types.TokenTypeId]*apidex.DividendPoolInfo)
		for _, pool := range dexFeesByPeriod.FeesForDividend {
			tk, _ := types.BytesToTokenTypeId(pool.Token)
			if tokenInfo, ok := dex.GetTokenInfo(db, tk); !ok {
				return nil, dex.InvalidTokenErr
			} else {
				amt := new(big.Int).SetBytes(pool.DividendPoolAmount)
				pool := &apidex.DividendPoolInfo{amt.String(), tokenInfo.QuoteTokenType, apidex.TokenInfoToRpc(tokenInfo, tk)}
				pools[tk] = pool
			}
		}
	}
	return pools, nil
}

func (f DexApi) HasStakedForVIP(address types.Address) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	_, ok := dex.GetVIPStaking(db, address)
	return ok, nil
}

func (f DexApi) GetStakedForVIP(address types.Address) (*apidex.VIPStakingRpc, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if vipStaking, ok := dex.GetVIPStaking(db, address); ok {
		return VIPStakingToRpc(f.chain, address, vipStaking, dex.StakeForVIP, dex.StakeForVIPAmount)
	} else {
		return nil, nil
	}
}

func (f DexApi) HasStakedForSVIP(address types.Address) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	_, ok := dex.GetSuperVIPStaking(db, address)
	return ok, nil
}

func (f DexApi) IsDexStopped() (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	return dex.IsDexStopped(db), nil
}

func (f DexApi) GetInviteCode(address types.Address) (uint32, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	return dex.GetCodeByInviter(db, address), nil
}

func (f DexApi) GetInviteCodeBinding(address types.Address) (uint32, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	if inviter, err := dex.GetInviterByInvitee(db, address); err != nil {
		if err == dex.NotBindInviterErr {
			return 0, nil
		} else {
			return 0, err
		}
	} else {
		return dex.GetCodeByInviter(db, *inviter), nil
	}
}

func (f DexApi) IsInviteCodeValid(code uint32) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	if addr, err := dex.GetInviterByCode(db, code); err != nil && err != dex.InvalidInviteCodeErr {
		return false, err
	} else {
		return addr != nil, nil
	}
}

func (f DexApi) IsMarketDelegatedTo(principal, agent types.Address, tradeToken, quoteToken types.TokenTypeId) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, quoteToken); !ok {
		return false, dex.TradeMarketNotExistsErr
	} else {
		return dex.IsMarketGrantedToAgent(db, principal, agent, marketInfo.MarketId), nil
	}
}

func (f DexApi) GetCurrentMiningInfo() (mineInfo *apidex.NewRpcVxMineInfo, err error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	periodId := dex.GetCurrentPeriodId(db, apidex.GetConsensusReader(f.vite))
	toMine := dex.GetVxToMineByPeriodId(db, periodId)
	available := dex.GetVxMinePool(db)
	if toMine.Cmp(available) > 0 {
		toMine = available
	}
	mineInfo = new(apidex.NewRpcVxMineInfo)
	if toMine.Sign() == 0 {
		err = fmt.Errorf("no vx available on mine")
		return
	}
	total := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100000000))
	total.Sub(total, dex.GetVxBurnAmount(db))
	mineInfo.HistoryMinedSum = new(big.Int).Sub(total, available).String()
	mineInfo.Total = toMine.String()
	var (
		amountForItems map[int32]*big.Int
		amount         *big.Int
		success        bool
	)
	if amountForItems, available, success = dex.GetVxAmountsForEqualItems(db, periodId, available, dex.RateSumForFeeMine, dex.ViteTokenType, dex.UsdTokenType); success {
		mineInfo.FeeMineDetail = make(map[int32]string)
		feeMineSum := new(big.Int)
		for tokenType, amount := range amountForItems {
			mineInfo.FeeMineDetail[tokenType] = amount.String()
			feeMineSum.Add(feeMineSum, amount)
		}
		mineInfo.FeeMineTotal = feeMineSum.String()
	} else {
		return
	}
	if amount, available, success = dex.GetVxAmountToMine(db, periodId, available, dex.RateForStakingMine); success {
		mineInfo.StakingMine = amount.String()
	} else {
		return
	}
	if amountForItems, available, success = dex.GetVxAmountsForEqualItems(db, periodId, available, dex.RateSumForMakerAndMaintainerMine, dex.MineForMaker, dex.MineForMaintainer); success {
		mineInfo.MakerMine = amountForItems[dex.MineForMaker].String()
	}
	return
}

func (f DexApi) GetCurrentFeesValidForMining() (fees map[int32]string, err error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if dexFeesByPeriod, ok := dex.GetCurrentDexFees(db, apidex.GetConsensusReader(f.vite)); !ok {
		return
	} else {
		fees = make(map[int32]string, len(dexFeesByPeriod.FeesForMine))
		for _, feeForMine := range dexFeesByPeriod.FeesForMine {
			fees[feeForMine.QuoteTokenType] = new(big.Int).Add(new(big.Int).SetBytes(feeForMine.BaseAmount), new(big.Int).SetBytes(feeForMine.InviteBonusAmount)).String()
		}
		return
	}
}

func (f DexApi) GetCurrentStakingValidForMining() (string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return "0", err
	}
	if miningStakings, ok := dex.GetDexMiningStakings(db); !ok {
		return "0", nil
	} else {
		stakingsLen := len(miningStakings.Stakings)
		if stakingsLen == 0 {
			return "0", nil
		} else {
			return new(big.Int).SetBytes(miningStakings.Stakings[stakingsLen-1].Amount).String(), nil
		}
	}
}

func (f DexApi) GetOrderById(orderIdStr string) (*apidex.RpcOrder, error) {
	orderId, err := hex.DecodeString(orderIdStr)
	if err != nil {
		return nil, err
	}
	if db, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		return apidex.InnerGetOrderById(db, orderId)
	}
}

func (f DexApi) GetOrderByTransactionHash(sendHash types.Hash) (*apidex.RpcOrder, error) {
	if db, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if orderId, ok := dex.GetOrderIdByHash(db, sendHash.Bytes()); !ok {
			return nil, dex.OrderNotExistsErr
		} else {
			return apidex.InnerGetOrderById(db, orderId)
		}
	}
}

func (f DexApi) GetOrdersForMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (ordersRes *apidex.OrdersRes, err error) {
	if fundDb, err := getVmDb(f.chain, types.AddressDexFund); err != nil {
		return nil, err
	} else {
		if marketInfo, ok := dex.GetMarketInfo(fundDb, tradeToken, quoteToken); !ok {
			return nil, dex.TradeMarketNotExistsErr
		} else {
			if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
				return nil, err
			} else {
				matcher := dex.NewMatcherWithMarketInfo(tradeDb, marketInfo)
				if ods, size, err := matcher.GetOrdersFromMarket(side, begin, end); err == nil {
					ordersRes = &apidex.OrdersRes{apidex.OrdersToRpc(ods), size}
					return ordersRes, err
				} else {
					return &apidex.OrdersRes{apidex.OrdersToRpc(ods), size}, err
				}
			}
		}
	}
}

func (f DexApi) GetVIPStakeInfoList(address types.Address, pageIndex int, pageSize int) (*apidex.StakeInfoList, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	var (
		stakeInfos    = make([]*dex.DelegateStakeInfo, 0)
		newStakeInfos []*dex.DelegateStakeInfo
		count         int
		totalAmount   = new(big.Int)
		newAmount     *big.Int
	)
	if vipStaking, ok := dex.GetVIPStaking(db, address); ok && len(vipStaking.StakingHashes) < int(vipStaking.StakedTimes) {
		vipStakeInfo := &dex.DelegateStakeInfo{}
		vipStakeInfo.StakeType = dex.StakeForVIP
		vipStakeInfo.Address = address.Bytes()
		vipStakeInfo.Amount = dex.StakeForVIPAmount.Bytes()
		stakeInfos = append(stakeInfos, vipStakeInfo)
		totalAmount.Add(totalAmount, dex.StakeForVIPAmount)
	}
	if superVipStaking, ok := dex.GetSuperVIPStaking(db, address); ok && len(superVipStaking.StakingHashes) < int(superVipStaking.StakedTimes) {
		superVipStakeInfo := &dex.DelegateStakeInfo{}
		superVipStakeInfo.StakeType = dex.StakeForSuperVIP
		superVipStakeInfo.Address = address.Bytes()
		superVipStakeInfo.Amount = dex.StakeForSuperVIPAmount.Bytes()
		stakeInfos = append(stakeInfos, superVipStakeInfo)
		totalAmount.Add(totalAmount, dex.StakeForSuperVIPAmount)
	}
	if newStakeInfos, newAmount, err = dex.GetStakeInfoList(db, address, func(index *dex.DelegateStakeAddressIndex) bool {
		bids := []int32{dex.StakeForVIP, dex.StakeForSuperVIP, dex.StakeForPrincipalSuperVIP}
		for _, bid := range bids {
			if index.StakeType == bid {
				return true
			}
		}
		return false
	}); err != nil {
		return nil, err
	}
	stakeInfos = append(stakeInfos, newStakeInfos...)
	count = len(stakeInfos)
	totalAmount.Add(totalAmount, newAmount)
	if count > pageIndex*pageSize {
		var endIndex = (pageIndex + 1) * pageSize
		if count < endIndex {
			endIndex = count
		}
		stakeInfos = stakeInfos[pageIndex*pageSize : endIndex]
	} else {
		stakeInfos = nil
	}
	return StakeListToDexRpc(stakeInfos, totalAmount, count, f.chain)
}

func (f DexApi) GetMiningStakeInfoList(address types.Address, pageIndex int, pageSize int) (*apidex.StakeInfoList, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	var (
		stakeInfos    = make([]*dex.DelegateStakeInfo, 0)
		newStakeInfos []*dex.DelegateStakeInfo
		count         int
		totalAmount   = new(big.Int)
		newAmount     *big.Int
	)
	if amount := dex.GetMiningStakedAmount(db, address); amount.Sign() > 0 {
		miningStakeInfo := &dex.DelegateStakeInfo{}
		miningStakeInfo.StakeType = dex.StakeForMining
		miningStakeInfo.Address = address.Bytes()
		miningStakeInfo.Amount = amount.Bytes()
		stakeInfos = append(stakeInfos, miningStakeInfo)
		totalAmount.Add(totalAmount, new(big.Int).SetBytes(miningStakeInfo.Amount))
	}
	if newStakeInfos, newAmount, err = dex.GetStakeInfoList(db, address, func(index *dex.DelegateStakeAddressIndex) bool {
		return index.StakeType == dex.StakeForMining
	}); err != nil {
		return nil, err
	}
	stakeInfos = append(stakeInfos, newStakeInfos...)
	count = len(stakeInfos)
	totalAmount.Add(totalAmount, newAmount)
	if count > pageIndex*pageSize {
		var endIndex = (pageIndex + 1) * pageSize
		if count < endIndex {
			endIndex = count
		}
		stakeInfos = stakeInfos[pageIndex*pageSize : endIndex]
	} else {
		stakeInfos = nil
	}
	return StakeListToDexRpc(stakeInfos, totalAmount, count, f.chain)
}

func (f DexApi) IsAutoLockMinedVx(address types.Address) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	return dex.IsAutoLockMinedVx(db, address.Bytes()), nil
}

func (f DexApi) GetVxUnlockList(address types.Address, pageIndex int, pageSize int) (*apidex.VxUnlockList, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	} else {
		if unlocks, ok := dex.GetVxUnlocks(db, address); !ok {
			return nil, nil
		} else {
			return apidex.UnlockListToRpc(unlocks, pageIndex, pageSize, f.chain), nil
		}
	}
}

func (f DexApi) GetCancelStakeList(address types.Address, pageIndex int, pageSize int) (*apidex.CancelStakeList, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	} else {
		if cancelStakes, ok := dex.GetCancelStakes(db, address); !ok {
			return nil, nil
		} else {
			return apidex.CancelStakeListToRpc(cancelStakes, pageIndex, pageSize, f.chain), nil
		}
	}
}

type DexPrivateApi struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDexPrivateApi(vite *vite.Vite) *DexPrivateApi {
	return &DexPrivateApi{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dex_private_api"),
	}
}

func (f DexPrivateApi) String() string {
	return "DexPrivateApi"
}

func (f DexPrivateApi) GetOwner() (*types.Address, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	return dex.GetOwner(db)
}

func (f DexPrivateApi) GetTime() (int64, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return -1, err
	}
	return dex.GetTimestampInt64(db), nil
}

func (f DexPrivateApi) GetPeriodId() (uint64, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	return dex.GetCurrentPeriodId(db, apidex.GetConsensusReader(f.vite)), nil
}

func (f DexPrivateApi) GetCurrentDexFees() (*apidex.RpcDexFeesByPeriod, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if dexFeesByPeriod, ok := dex.GetCurrentDexFees(db, apidex.GetConsensusReader(f.vite)); !ok {
		return nil, nil
	} else {
		return apidex.DexFeesByPeriodToRpc(dexFeesByPeriod), nil
	}
}

func (f DexPrivateApi) GetDexFeesByPeriod(periodId uint64) (*apidex.RpcDexFeesByPeriod, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if dexFeesByPeriod, ok := dex.GetDexFeesByPeriodId(db, periodId); !ok {
		return nil, nil
	} else {
		return apidex.DexFeesByPeriodToRpc(dexFeesByPeriod), nil
	}
}

func (f DexPrivateApi) GetCurrentOperatorFees(operator types.Address) (*apidex.RpcOperatorFeesByPeriod, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if operatorFeesByPeriod, ok := dex.GetCurrentOperatorFees(db, apidex.GetConsensusReader(f.vite), operator.Bytes()); !ok {
		return nil, nil
	} else {
		return apidex.OperatorFeesByPeriodToRpc(operatorFeesByPeriod), nil
	}
}

func (f DexPrivateApi) GetOperatorFeesByPeriod(periodId uint64, operator types.Address) (*apidex.RpcOperatorFeesByPeriod, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if operatorFeesByPeriod, ok := dex.GetOperatorFeesByPeriodId(db, operator.Bytes(), periodId); !ok {
		return nil, nil
	} else {
		return apidex.OperatorFeesByPeriodToRpc(operatorFeesByPeriod), nil
	}
}

func (f DexPrivateApi) GetAllFeesOfAddress(address types.Address) (*apidex.RpcUserFees, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if userFees, ok := dex.GetUserFees(db, address.Bytes()); !ok {
		return nil, nil
	} else {
		return apidex.UserFeesToRpc(userFees), nil
	}
}

func (f DexPrivateApi) GetAllTotalVxBalance() (*apidex.RpcVxFunds, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if vxSumFunds, ok := dex.GetVxSumFundsWithForkCheck(db); !ok {
		return nil, nil
	} else {
		return apidex.VxFundsToRpc(vxSumFunds), nil
	}
}

func (f DexPrivateApi) GetAllVxBalanceByAddress(address types.Address) (*apidex.RpcVxFunds, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if vxFunds, ok := dex.GetVxFundsWithForkCheck(db, address.Bytes()); !ok {
		return nil, nil
	} else {
		return apidex.VxFundsToRpc(vxFunds), nil
	}
}

func (f DexPrivateApi) GetVxPoolBalance() (string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return "", err
	}
	balance := dex.GetVxMinePool(db)
	return balance.String(), nil
}

func (f DexPrivateApi) GetVxBurnBalance() (string, error) {
	if db, err := getVmDb(f.chain, types.AddressDexFund); err != nil {
		return "-1", err
	} else {
		return dex.GetVxBurnAmount(db).String(), nil
	}
}

func (f DexPrivateApi) GetVIPStakingInfoByAddress(address types.Address) (*dex.VIPStaking, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if info, ok := dex.GetVIPStaking(db, address); ok {
		return info, nil
	} else {
		return nil, nil
	}
}

func (f DexPrivateApi) GetCurrentMiningStakingAmountByAddress(address types.Address) (map[string]string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	res := make(map[string]string, 0)
	res["v1"] = dex.GetMiningStakedAmount(db, address).String()
	res["v2"] = dex.GetMiningStakedV2Amount(db, address).String()
	return res, nil
}

func (f DexPrivateApi) GetAllMiningStakingInfoByAddress(address types.Address) (*apidex.RpcMiningStakings, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if miningStakings, ok := dex.GetMiningStakings(db, address); ok {
		return apidex.MiningStakingsToRpc(miningStakings), nil
	} else {
		return nil, nil
	}
}

func (f DexPrivateApi) GetAllMiningStakingInfo() (*apidex.RpcMiningStakings, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if miningStakings, ok := dex.GetDexMiningStakings(db); ok {
		return apidex.MiningStakingsToRpc(miningStakings), nil
	} else {
		return nil, nil
	}
}

func (f DexPrivateApi) GetDexConfig() (map[string]string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	configs := make(map[string]string)
	owner, _ := dex.GetOwner(db)
	configs["owner"] = owner.String()
	if timer := dex.GetTimeOracle(db); timer != nil {
		configs["timer"] = timer.String()
	}
	if trigger := dex.GetPeriodJobTrigger(db); trigger != nil {
		configs["trigger"] = trigger.String()
	}
	if mineProxy := dex.GetMakerMiningAdmin(db); mineProxy != nil {
		configs["mineProxy"] = mineProxy.String()
	}
	if maintainer := dex.GetMaintainer(db); maintainer != nil {
		configs["maintainer"] = maintainer.String()
	}
	return configs, nil
}

func (f DexPrivateApi) GetMinThresholdForTradeAndMining() (map[int]*apidex.RpcThresholdForTradeAndMine, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	thresholds := make(map[int]*apidex.RpcThresholdForTradeAndMine, 4)
	for tokenType := dex.ViteTokenType; tokenType <= dex.UsdTokenType; tokenType++ {
		tradeThreshold := dex.GetTradeThreshold(db, int32(tokenType))
		mineThreshold := dex.GetMineThreshold(db, int32(tokenType))
		thresholds[tokenType] = &apidex.RpcThresholdForTradeAndMine{TradeThreshold: tradeThreshold.String(), MineThreshold: mineThreshold.String()}
	}
	return thresholds, nil
}

func (f DexPrivateApi) GetMakerMiningPool(periodId uint64) (string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return "", err
	}
	balance := dex.GetMakerMiningPoolByPeriodId(db, periodId)
	return balance.String(), nil
}

func (f DexPrivateApi) GetLastPeriodIdByJobType(bizType uint8) (uint64, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	if lastPeriodId := dex.GetLastJobPeriodIdByBizType(db, bizType); err != nil {
		return 0, err
	} else {
		return lastPeriodId, nil
	}
}

func (f DexPrivateApi) VerifyDexBalance() (*dex.FundVerifyRes, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	return dex.VerifyDexFundBalance(db, apidex.GetConsensusReader(f.vite)), nil
}

func (f DexPrivateApi) IsNormalMiningStarted() (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	return dex.IsNormalMiningStarted(db), nil
}

func (f DexPrivateApi) GetFirstMiningPeriodId() (uint64, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	if firstPeriodId := dex.GetFirstMinedVxPeriodId(db); err != nil {
		return 0, err
	} else {
		return firstPeriodId, nil
	}
}

func (f DexPrivateApi) GetLastSettledMakerMiningInfo() (map[string]uint64, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	lastSettleInfo := make(map[string]uint64)
	lastSettleInfo["period"] = dex.GetLastSettledMakerMinedVxPeriod(db)
	lastSettleInfo["page"] = uint64(dex.GetLastSettledMakerMinedVxPage(db))
	return lastSettleInfo, nil
}

func (f DexPrivateApi) GetMarketInfoById(marketId int32) (ordersRes *apidex.RpcMarketInfo, err error) {
	if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if marketInfo, ok := dex.GetMarketInfoById(tradeDb, marketId); ok {
			return apidex.MarketInfoToRpc(marketInfo), nil
		} else {
			return nil, nil
		}
	}
}

func (f DexPrivateApi) GetTradeTimestamp() (timestamp int64, err error) {
	if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return -1, err
	} else {
		return dex.GetTradeTimestamp(tradeDb), nil
	}
}

func (f DexPrivateApi) GetDelegateStakeInfoById(id types.Hash) (*apidex.DelegateStakeInfo, error) {
	if db, err := getVmDb(f.chain, types.AddressDexFund); err != nil {
		return nil, err
	} else {
		if info, ok := dex.GetDelegateStakeInfo(db, id.Bytes()); ok {
			return apidex.DelegateStakeInfoToRpc(info), nil
		} else {
			return nil, nil
		}
	}
}

func StakeListToDexRpc(stakeInfos []*dex.DelegateStakeInfo, totalAmount *big.Int, count int, chain chain.Chain) (*apidex.StakeInfoList, error) {
	list := new(apidex.StakeInfoList)
	list.StakeAmount = totalAmount.String()
	list.Count = count
	if db, err := getVmDb(chain, types.AddressQuota); err != nil {
		return nil, err
	} else {
		var snapshotBlock *ledger.SnapshotBlock
		for _, stakeInfo := range stakeInfos {
			info := new(apidex.StakeInfo)
			stakeAddr, _ := types.BytesToAddress(stakeInfo.Address)
			info.Amount = apidex.AmountBytesToString(stakeInfo.Amount)
			info.Beneficiary = types.AddressDexFund.String()
			info.IsDelegated = true
			info.DelegateAddress = types.AddressDexFund.String()
			info.StakeAddress = stakeAddr.String()
			info.Bid = uint8(stakeInfo.StakeType)
			if snapshotBlock == nil {
				if snapshotBlock, err = db.LatestSnapshotBlock(); err != nil {
					return nil, err
				}
			}
			if info.Id, info.ExpirationHeight, info.ExpirationTime, err = getStakeExpirationInfo(db, stakeInfo.Id, stakeAddr, info.Bid, snapshotBlock); err != nil {
				return nil, err
			}
			if stakeInfo.StakeType == dex.StakeForPrincipalSuperVIP {
				principal, _ := types.BytesToAddress(stakeInfo.Principal)
				info.Principal = principal.String()
			}
			list.StakeList = append(list.StakeList, info)
		}
	}
	return list, nil
}

func VIPStakingToRpc(chain chain.Chain, address types.Address, info *dex.VIPStaking, bid uint8, amount *big.Int) (vipStakingRpc *apidex.VIPStakingRpc, err error) {
	var (
		db            vm_db.VmDb
		snapshotBlock *ledger.SnapshotBlock
		id            []byte
	)
	if len(info.StakingHashes) > 0 {
		id = info.StakingHashes[0]
	}
	if db, err = getVmDb(chain, types.AddressQuota); err != nil {
		return
	}
	if snapshotBlock, err = db.LatestSnapshotBlock(); err != nil {
		return nil, err
	}
	vipStakingRpc = new(apidex.VIPStakingRpc)
	vipStakingRpc.Amount = amount.String()
	vipStakingRpc.Id, vipStakingRpc.ExpirationHeight, vipStakingRpc.ExpirationTime, err = getStakeExpirationInfo(db, id, address, bid, snapshotBlock)
	return
}

func getStakeExpirationInfo(db vm_db.VmDb, id []byte, address types.Address, bid uint8, snapshotBlock *ledger.SnapshotBlock) (idStr string, expirationHeight string, expirationTime int64, err error) {
	var quotaInfo *types.StakeInfo
	if len(id) > 0 {
		idHash, _ := types.BytesToHash(id)
		idStr = idHash.String()
		if quotaInfo, err = abi.GetStakeInfoById(db, id); err != nil {
			return
		}
	} else {
		if quotaInfo, err = abi.GetStakeInfo(db, address, types.AddressDexFund, types.AddressDexFund, true, bid); err != nil {
			return
		}
	}
	expirationHeight = strconv.FormatInt(int64(quotaInfo.ExpirationHeight), 10)
	expirationTime = getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, quotaInfo.ExpirationHeight)
	return
}
