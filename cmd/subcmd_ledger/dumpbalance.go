package subcmd_ledger

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
)

type dumpedAmount struct {
	Address      *types.Address
	Sum          *big.Int
	WalletAmount *big.Int
	DexAvailable *big.Int
	DexLocked    *big.Int
	DexOther     *big.Int

	UnReceiveAmount *big.Int
}

func newDumpedAmount(addr types.Address) *dumpedAmount {
	return &dumpedAmount{&addr, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
}

type dumpedAmountSorter []*dumpedAmount

func (st dumpedAmountSorter) Len() int {
	return len(st)
}

func (st dumpedAmountSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st dumpedAmountSorter) Less(i, j int) bool {
	addCmp := st[i].Sum.Cmp(st[j].Sum)
	if addCmp > 0 {
		return true
	} else {
		return false
	}
}

func dumpBalance(chain chain.Chain, token types.TokenTypeId, snapshotHeight uint64) error {
	var snapshotBlock *core.SnapshotBlock
	var err error
	if snapshotBlock, err = chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return err
	}
	if snapshotBlock == nil {
		return fmt.Errorf("snapshot block is not exist, height:%d", snapshotHeight)
	}
	res := make(map[types.Address]*dumpedAmount, 100)
	chain.IterateAccounts(func(addr types.Address, accountId uint64, err1 error) bool {
		if err1 != nil {
			log.Error("GetLatestAccountBlock IterateAccounts failed, error is "+err.Error(), "method", "DumpBalance")
			return false
		}
		if balances, err2 := chain.GetConfirmedBalanceList([]types.Address{addr}, token, snapshotBlock.Hash); err2 != nil {
			log.Error("GetLatestAccountBlock GetConfirmedBalanceList failed, error is "+err2.Error(), "method", "DumpBalance")
			return false
		} else if balance, ok := balances[addr]; ok && balance.Sign() > 0 {
			dumpAmt := newDumpedAmount(addr)
			if accAmt, ok := res[addr]; ok {
				dumpAmt = accAmt
			} else {
				res[addr] = dumpAmt
			}
			dumpAmt.WalletAmount = balance
			dumpAmt.Sum = balance
			res[addr] = dumpAmt
		}
		if roadInfo, err := chain.GetAccountOnRoadInfo(addr); err != nil {
			log.Error(fmt.Sprintf("GetAccountOnRoadInfo failed, error is %s", err.Error()), "method", "DumpBalance")
		} else if roadTokenInfo, ok := roadInfo.TokenBalanceInfoMap[token]; ok {
			dumpAmt := newDumpedAmount(addr)
			if accAmt, ok := res[addr]; ok {
				dumpAmt = accAmt
			} else {
				res[addr] = dumpAmt
			}
			dumpAmt.UnReceiveAmount = &roadTokenInfo.TotalAmount
		}
		return true
	})
	pageSize := 10
	startAddress := types.ZERO_ADDRESS
	for {
		if funds, err := chain.GetDexFundsByPage(snapshotBlock.Hash, startAddress, pageSize); err != nil {
			return err
		} else {
			for _, fund := range funds {
				for _, acc := range fund.Accounts {
					if bytes.Equal(acc.Token, token.Bytes()) {
						dexAvailable := new(big.Int).SetBytes(acc.Available)
						dexLocked := new(big.Int).SetBytes(acc.Locked)
						dexOther := big.NewInt(0)
						if token == dex.VxTokenId {
							dexOther = dexOther.Add(new(big.Int).SetBytes(acc.VxLocked), new(big.Int).SetBytes(acc.VxUnlocking))
						} else if token == core.ViteTokenId && len(acc.CancellingStake) > 0 {
							dexOther = dexOther.Add(dexOther, new(big.Int).SetBytes(acc.CancellingStake))
						}
						dexAmt := new(big.Int).Add(new(big.Int).Add(dexAvailable, dexLocked), dexOther)
						if dexAmt.Sign() > 0 {
							address, _ := types.BytesToAddress(fund.Address)
							dumpAmt := newDumpedAmount(address)
							if accAmt, ok := res[address]; ok {
								dumpAmt = accAmt
							} else {
								res[address] = dumpAmt
							}

							walletAmt := dumpAmt.WalletAmount
							sum := new(big.Int)
							sum = sum.Add(sum, walletAmt)
							sum = sum.Add(sum, dexAvailable)
							sum = sum.Add(sum, dexLocked)
							sum = sum.Add(sum, dexOther)

							dumpAmt.DexAvailable = dexAvailable
							dumpAmt.DexLocked = dexLocked
							dumpAmt.DexOther = dexOther
							dumpAmt.Sum = sum
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

	resList := make([]*dumpedAmount, len(res))
	i := 0
	for _, v := range res {
		resList[i] = v
		i++
	}
	sort.Sort(dumpedAmountSorter(resList))
	fmt.Println("address, sum , wallet , dexAvailable, dexLocked, dexOther, unReceived")
	overAllSum := big.NewInt(0)
	for _, v := range resList {
		fmt.Printf("%s,%s,%s,%s,%s,%s,%s\n", v.Address.String(), v.Sum.String(), v.WalletAmount.String(), v.DexAvailable.String(), v.DexLocked.String(), v.DexOther.String(), v.UnReceiveAmount.String())
		overAllSum.Add(overAllSum, v.Sum)
	}
	fmt.Printf(">>>>>>>>>>>>>>>>>>>>> valid size %d, overAllSum %s\n", len(resList), overAllSum.String())
	return nil
}
