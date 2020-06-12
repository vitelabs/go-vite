package api

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)

type Dumper struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDumper(vite *vite.Vite) *Dumper {
	return &Dumper{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dumper_api"),
	}
}

func (f Dumper) String() string {
	return "DumperApi"
}

func (f Dumper) DumpBalance(token types.TokenTypeId, snapshotHeight uint64) (error) {
	var snapshotBlock *ledger.SnapshotBlock
	var err error
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return err
	}
	res := make(map[types.Address]*big.Int, 100)
	f.chain.IterateAccounts(func(addr types.Address, accountId uint64, err1 error) bool {
		if err1 != nil {
			f.log.Error("GetLatestAccountBlock IterateAccounts failed, error is "+err.Error(), "method", "DumpBalance")
			return false
		}
		if balances, err2 := f.chain.GetConfirmedBalanceList([]types.Address{addr}, token, snapshotBlock.Hash); err2 != nil {
			f.log.Error("GetLatestAccountBlock GetConfirmedBalanceList failed, error is "+err2.Error(), "method", "DumpBalance")
			return false
		} else if balance, ok := balances[addr]; ok && balance.Sign() > 0 {
			res[addr] = balance
		}
		return true
	})
	pageSize := 10
	startAddress := types.ZERO_ADDRESS
	for {
		if funds, err := f.chain.GetDexFundsByPage(snapshotBlock.Hash, startAddress, pageSize); err != nil {
			return err
		} else {
			for _, fund := range funds {
				for _, acc := range fund.Accounts {
					if bytes.Equal(acc.Token, token.Bytes()) {
						dexAmt := dex.AddBigInt(acc.Available, acc.Locked)
						if token == dex.VxTokenId {
							vxLocked := dex.AddBigInt(acc.VxLocked, acc.VxUnlocking)
							dexAmt = dex.AddBigInt(dexAmt, vxLocked)
						} else if token == ledger.ViteTokenId && len(acc.CancellingStake) > 0 {
							dexAmt = dex.AddBigInt(dexAmt, acc.CancellingStake)
						}
						if len(dexAmt) > 0 {
							address, _ := types.BytesToAddress(fund.Address)
							if accAmt, ok := res[address]; ok {
								res[address] = new(big.Int).Add(accAmt, new(big.Int).SetBytes(dexAmt))
							} else {
								res[address] = new(big.Int).SetBytes(dexAmt)
							}
						}
						break
					}
				}
			}
			fundsLen := len(funds)
			if fundsLen < pageSize {
				break
			}
			if startAddress, err = types.BytesToAddress(funds[fundsLen-1].Address); err != nil {
				return err
			}
		}
	}
	validNum := 0
	sum := big.NewInt(0)
	for k, v := range res {
		fmt.Printf("%s,%s\n", k.String(), v.String())
		validNum++
		sum.Add(sum, v)
	}
	fmt.Printf(">>>>>>>>>>>>>>>>>>>>> valid size %d, sum %s\n", validNum, sum.String())
	return nil
}

func (f Dumper) DumpAccountBalance(token types.TokenTypeId, snapshotHeight uint64, address types.Address) (snapshotBalance *SnapshotBalance, err error) {
	var (
		snapshotBlock *ledger.SnapshotBlock
		balances      map[types.Address]*big.Int
	)
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return
	}
	if balances, err = f.chain.GetConfirmedBalanceList([]types.Address{address}, token, snapshotBlock.Hash); err != nil {
		f.log.Error("GetLatestAccountBlock GetConfirmedBalanceList failed, error is "+err.Error(), "method", "DumpBalance")
		return
	}
	snapshotBalance = &SnapshotBalance{}
	snapshotBalance.WalletBalance, _ = balances[address]

	var fund *dex.Fund
	if fund, err = f.chain.GetDexFundByAddress(snapshotBlock.Hash, address); err != nil || fund == nil {
		return
	}
	for _, acc := range fund.Accounts {
		if bytes.Equal(acc.Token, token.Bytes()) {
			dexAmt := dex.AddBigInt(acc.Available, acc.Locked)
			if token == dex.VxTokenId {
				vxLocked := dex.AddBigInt(acc.VxLocked, acc.VxUnlocking)
				dexAmt = dex.AddBigInt(dexAmt, vxLocked)
			} else if token == ledger.ViteTokenId && len(acc.CancellingStake) > 0 {
				dexAmt = dex.AddBigInt(dexAmt, acc.CancellingStake)
			}
			snapshotBalance.DexBalance = new(big.Int).SetBytes(dexAmt)
			break
		}
	}
	return
}

type SnapshotBalance struct {
	WalletBalance *big.Int `json:"walletBalance"`
	DexBalance    *big.Int `json:"dexBalance"`
}
