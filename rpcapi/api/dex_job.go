package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)

type DexJobApi struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDexJobApi(vite *vite.Vite) *DexJobApi {
	return &DexJobApi{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dexjob_api"),
	}
}

func (f DexJobApi) String() string {
	return "DexJobApi"
}

func (f DexJobApi) TriggerFeeMine(periodId uint64) error {
	db, err := getDb(f.chain, types.AddressDexFund)
	if err != nil {
		return err
	}
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10000))
	amtForMarkets := map[int32]*big.Int{dex.ViteTokenType : amount, dex.EthTokenType : amount, dex.BtcTokenType : amount, dex.UsdTokenType : amount}
	if refund, err := dex.DoMineVxForFee(db, getConsensusReader(f.vite), periodId, amtForMarkets, log); err != nil {
		return err
	} else {
		fmt.Printf("mine refund %s\n", refund.String())
	}
	return nil
}

func (f DexJobApi) TriggerPledgeMine(periodId uint64) error {
	db, err := getDb(f.chain, types.AddressDexFund)
	if err != nil {
		return err
	}
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10000))
	if refund, err := dex.DoMineVxForPledge(db, getConsensusReader(f.vite), periodId, amount); err != nil {
		return err
	} else {
		fmt.Printf("mine refund %s\n", refund.String())
	}
	return nil
}

func (f DexJobApi) TriggerDividend(periodId uint64) error {
	db, err := getDb(f.chain, types.AddressDexFund)
	if err != nil {
		return err
	}
	if err = dex.DoDivideFees(db, periodId); err != nil {
		return err
	}
	return nil
}