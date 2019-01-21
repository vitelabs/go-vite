package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

type DexFundApi struct {
	chain     chain.Chain
	log       log15.Logger
	ledgerApi *LedgerApi
}

func NewDexFundApiApi(vite *vite.Vite) *DexFundApi {
	return &DexFundApi{
		chain:     vite.Chain(),
		log:       log15.New("module", "rpc_api/pledge_api"),
		ledgerApi: NewLedgerApi(vite),
	}
}

func (f DexFundApi) String() string {
	return "DexFundApi"
}

type AccountFundInfo struct {
	Token     types.TokenTypeId `json:"token"`
	Available string            `json:"available"`
	Locked    string            `json:"locked"`
}

func (f DexFundApi) GetAccountFundInfo(addr types.Address, tokenId *types.TokenTypeId) ([]*AccountFundInfo, error) {
	snapshotBlock := f.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(f.chain, &snapshotBlock.Hash, nil, &addr)
	if err != nil {
		return nil, err
	}
	dexFund, err := dex.GetUserFundFromStorage(vmContext, addr)
	if err != nil {
		return nil, err
	}
	fundInfo, err := dex.GetAccountFundInfo(dexFund, tokenId)
	if err != nil {
		return nil, err
	}

	tokenInfoList := make([]*AccountFundInfo, 0)
	for _, v := range fundInfo {
		info := &AccountFundInfo{Token: v.Token}
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

		tokenInfoList = append(tokenInfoList, info)
	}
	return tokenInfoList, nil
}

func (f DexFundApi) GetAccountFundInfoByStatus(addr types.Address, tokenId *types.TokenTypeId, status byte) (map[types.TokenTypeId]string, error) {
	if status != 0 || status != 1 || status != 2 {
		return nil, errors.New("args's status error, 1 for available, 2 for locked, 0 for total")
	}

	snapshotBlock := f.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(f.chain, &snapshotBlock.Hash, nil, &addr)
	if err != nil {
		return nil, err
	}
	dexFund, err := dex.GetUserFundFromStorage(vmContext, addr)
	if err != nil {
		return nil, err
	}
	fundInfo, err := dex.GetAccountFundInfo(dexFund, nil)
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
