package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	apidex "github.com/vitelabs/go-vite/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)

type DexFundApi struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDexFundApi(vite *vite.Vite) *DexFundApi {
	return &DexFundApi{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dexfund_api"),
	}
}

func (f DexFundApi) String() string {
	return "DexFundApi"
}

func (f DexFundApi) GetAccountFundInfo(addr types.Address, tokenId *types.TokenTypeId) (map[types.TokenTypeId]*AccountBalanceInfo, error) {
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

func (f DexFundApi) GetTokenInfo(token types.TokenTypeId) (*apidex.RpcDexTokenInfo, error) {
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

func (f DexFundApi) GetMarketInfo(tradeToken, quoteToken types.TokenTypeId) (*apidex.RpcMarketInfo, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, quoteToken); ok {
		return apidex.MarketInfoToRpc(marketInfo), nil
	} else {
		return nil, dex.TradeMarketNotExistsErr
	}
}

func (f DexFundApi) GetCurrentDividendPools() (map[types.TokenTypeId]*apidex.DividendPoolInfo, error) {
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

func (f DexFundApi) IsPledgeVip(address types.Address) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	_, ok := dex.GetVIPStaking(db, address)
	return ok, nil
}

func (f DexFundApi) IsPledgeSuperVip(address types.Address) (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	_, ok := dex.GetSuperVIPStaking(db, address)
	return ok, nil
}

func (f DexFundApi) IsViteXStopped() (bool, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	return dex.IsDexStopped(db), nil
}

func (f DexFundApi) GetInviterCode(address types.Address) (uint32, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return 0, err
	}
	return dex.GetCodeByInviter(db, address), nil
}

func (f DexFundApi) GetInviteeCode(address types.Address) (uint32, error) {
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

func (f DexFundApi) IsMarketGrantedToAgent(principal, agent types.Address, tradeToken, quoteToken types.TokenTypeId) (bool, error) {
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

func (f DexFundApi) GetCurrentVxMineInfo() (mineInfo *apidex.RpcVxMineInfo, err error) {
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
	mineInfo = new(apidex.RpcVxMineInfo)
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
		mineInfo.PledgeMine = amount.String()
	} else {
		return
	}
	if amountForItems, available, success = dex.GetVxAmountsForEqualItems(db, periodId, available, dex.RateSumForMakerAndMaintainerMine, dex.MineForMaker, dex.MineForMaintainer); success {
		mineInfo.MakerMine = amountForItems[dex.MineForMaker].String()
	}
	return
}

func (f DexFundApi) GetCurrentFeesForMine() (fees map[int32]string, err error) {
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

func (f DexFundApi) GetCurrentPledgeForVxSum() (string, error) {
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return "0", err
	}
	if miningStakings, ok := dex.GetDexMiningStakings(db); !ok {
		return "0", nil
	} else {
		pledgesLen := len(miningStakings.Stakings)
		if pledgesLen == 0 {
			return "0", nil
		} else {
			return new(big.Int).SetBytes(miningStakings.Stakings[pledgesLen-1].Amount).String(), nil
		}
	}
}
