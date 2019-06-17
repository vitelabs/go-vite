package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type DexFundApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewDexFundApi(vite *vite.Vite) *DexFundApi {
	return &DexFundApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dexfund_api"),
	}
}

func (f DexFundApi) String() string {
	return "DexFundApi"
}

type AccountFundInfo struct {
	TokenInfo *RpcTokenInfo `json:"tokenInfo,omitempty"`
	Available string        `json:"available"`
	Locked    string        `json:"locked"`
}

func (f DexFundApi) GetAccountFundInfo(addr types.Address, tokenId *types.TokenTypeId) (map[types.TokenTypeId]*AccountFundInfo, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	dexFund, _ := dex.GetUserFund(db, addr)
	fundInfo, err := dex.GetAccountFundInfo(dexFund, tokenId)
	if err != nil {
		return nil, err
	}

	accFundInfo := make(map[types.TokenTypeId]*AccountFundInfo, 0)
	for _, v := range fundInfo {
		tokenInfo, err := f.chain.GetTokenInfoById(v.Token)
		if err != nil {
			return nil, err
		}
		info := &AccountFundInfo{TokenInfo: RawTokenInfoToRpc(tokenInfo, v.Token)}
		a := "0"
		if v.Available != nil {
			a = v.Available.String()
		}
		info.Available = a

		l := "0"
		if v.Locked != nil {
			l = v.Locked.String()
		}
		info.Locked = l

		accFundInfo[v.Token] = info
	}
	return accFundInfo, nil
}

func (f DexFundApi) GetAccountFundInfoByStatus(addr types.Address, tokenId *types.TokenTypeId, status byte) (map[types.TokenTypeId]string, error) {
	if status != 0 && status != 1 && status != 2 {
		return nil, errors.New("args's status error, 1 for available, 2 for locked, 0 for total")
	}

	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	dexFund, _ := dex.GetUserFund(db, addr)
	fundInfo, err := dex.GetAccountFundInfo(dexFund, tokenId)
	if err != nil {
		return nil, err
	}

	fundInfoMap := make(map[types.TokenTypeId]string, 0)
	for _, v := range fundInfo {
		amount := big.NewInt(0)
		if a := v.Available; a != nil {
			if status == 0 || status == 1 {
				amount.Add(amount, a)
			}
		}
		if l := v.Locked; l != nil {
			if status == 0 || status == 2 {
				amount.Add(amount, l)
			}
		}
		fundInfoMap[v.Token] = amount.String()
	}
	return fundInfoMap, nil
}

func (f DexFundApi) GetMarketInfo(tradeToken, quoteToken types.TokenTypeId) (*RpcMarketInfo, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, quoteToken); ok {
		return MarketInfoToRpc(marketInfo), nil
	} else {
		return nil, dex.TradeMarketNotExistsErr
	}
}

func (f DexFundApi) IsPledgeVip(address types.Address) (bool, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return false, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return false, err
	}
	_, ok := dex.GetPledgeForVip(db, address)
	return ok, nil
}

func (f DexFundApi) VerifyFundBalance() (*dex.FundVerifyRes, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	return dex.VerifyDexFundBalance(db), nil
}

func (f DexFundApi) GetOwner() (*types.Address, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	return dex.GetOwner(db)
}

func (f DexFundApi) GetTime() (int64, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return -1, err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return -1, err
	}
	return dex.GetTimestampInt64(db), nil
}

func (f DexFundApi) GetPledgeForVX(address types.Address) (string, error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexFund)
	if err != nil {
		return "", err
	}
	db, err := vm_db.NewVmDb(f.chain, &types.AddressDexFund, &f.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return "", err
	}
	return dex.GetPledgeForVx(db, address).String(), nil
}

type RpcMarketInfo struct {
	MarketId           int32  `json:"marketId"`
	MarketSymbol       string  `json:"marketSymbol"`
	TradeToken         string  `json:"tradeToken"`
	QuoteToken         string  `json:"quoteToken"`
	QuoteTokenType     int32   `json:"quoteTokenType"`
	TradeTokenDecimals int32   `json:"tradeTokenDecimals,omitempty"`
	QuoteTokenDecimals int32   `json:"quoteTokenDecimals"`
	TakerBrokerFeeRate int32 `json:"takerBrokerFeeRate,omitempty"`
	MakerBrokerFeeRate int32 `json:"makerBrokerFeeRate,omitempty"`
	AllowMine          bool    `json:"allowMine"`
	Owner              string  `json:"owner"`
	Creator            string  `json:"creator"`
	Stopped            bool    `json:"stopped"`
	Timestamp          int64   `json:"timestamp"`
}

func MarketInfoToRpc(mkInfo *dex.MarketInfo) *RpcMarketInfo {
	var rmk *RpcMarketInfo = nil
	if mkInfo != nil {
		tradeToken, _ := types.BytesToTokenTypeId(mkInfo.TradeToken)
		quoteToken, _ := types.BytesToTokenTypeId(mkInfo.QuoteToken)
		owner, _ := types.BytesToAddress(mkInfo.Owner)
		creator, _ := types.BytesToAddress(mkInfo.Creator)
		rmk = &RpcMarketInfo{
			MarketId:           mkInfo.MarketId,
			MarketSymbol:       mkInfo.MarketSymbol,
			TradeToken:         tradeToken.String(),
			QuoteToken:         quoteToken.String(),
			QuoteTokenType:     mkInfo.QuoteTokenType,
			TradeTokenDecimals: mkInfo.TradeTokenDecimals,
			QuoteTokenDecimals: mkInfo.QuoteTokenDecimals,
			TakerBrokerFeeRate: mkInfo.TakerBrokerFeeRate,
			MakerBrokerFeeRate: mkInfo.MakerBrokerFeeRate,
			AllowMine:          mkInfo.AllowMine,
			Owner:              owner.String(),
			Creator:            creator.String(),
			Stopped:            mkInfo.Stopped,
			Timestamp:          mkInfo.Timestamp,
		}
	}
	return rmk
}
