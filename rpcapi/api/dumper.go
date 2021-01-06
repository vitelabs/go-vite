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
	"sort"
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

type DumpedAmount struct {
	Address      *types.Address
	Sum          *big.Int
	WalletAmount *big.Int
	DexAvailable *big.Int
	DexLocked    *big.Int
	DexOther     *big.Int
}

func (f Dumper) DumpBalance(token types.TokenTypeId, snapshotHeight uint64) (error) {
	var snapshotBlock *ledger.SnapshotBlock
	var err error
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return err
	}
	res := make(map[types.Address]*DumpedAmount, 100)
	f.chain.IterateAccounts(func(addr types.Address, accountId uint64, err1 error) bool {
		if err1 != nil {
			f.log.Error("GetLatestAccountBlock IterateAccounts failed, error is "+err.Error(), "method", "DumpBalance")
			return false
		}
		if balances, err2 := f.chain.GetConfirmedBalanceList([]types.Address{addr}, token, snapshotBlock.Hash); err2 != nil {
			f.log.Error("GetLatestAccountBlock GetConfirmedBalanceList failed, error is "+err2.Error(), "method", "DumpBalance")
			return false
		} else if balance, ok := balances[addr]; ok && balance.Sign() > 0 {
			dumpAmt := &DumpedAmount{}
			dumpAmt.Address = &addr
			dumpAmt.WalletAmount = balance
			dumpAmt.Sum = balance
			res[addr] = dumpAmt
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
						dexAvailable := new(big.Int).SetBytes(acc.Available)
						dexLocked := new(big.Int).SetBytes(acc.Locked)
						dexOther := big.NewInt(0)
						if token == dex.VxTokenId {
							dexOther = dexOther.Add(new(big.Int).SetBytes(acc.VxLocked), new(big.Int).SetBytes(acc.VxUnlocking))
						} else if token == ledger.ViteTokenId && len(acc.CancellingStake) > 0 {
							dexOther = dexOther.Add(dexOther, new(big.Int).SetBytes(acc.CancellingStake))
						}
						dexAmt := new(big.Int).Add(new(big.Int).Add(dexAvailable, dexLocked), dexOther)
						if dexAmt.Sign() > 0 {
							address, _ := types.BytesToAddress(fund.Address)
							walletAmt := new(big.Int)
							if accAmt, ok := res[address]; ok {
								walletAmt.Set(accAmt.WalletAmount)
							}
							sum := new(big.Int)
							sum = sum.Add(sum, walletAmt)
							sum = sum.Add(sum, dexAvailable)
							sum = sum.Add(sum, dexLocked)
							sum = sum.Add(sum, dexOther)
							newAmt := &DumpedAmount{&address, sum, walletAmt, dexAvailable, dexLocked, dexOther}
							res[address] = newAmt
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

	resList := make([]*DumpedAmount, len(res))
	i := 0
	for _, v := range res {
		resList[i] = v
		i++
	}
	sort.Sort(DumpedAmountSorter(resList))
	fmt.Println("address, sum , wallet , dexAvailable, dexLocked, dexOther")
	overAllSum := big.NewInt(0)
	for _, v := range resList {
		fmt.Printf("%s,%s,%s,%s,%s,%s\n", v.Address.String(), v.Sum.String(), v.WalletAmount.String(), v.DexAvailable.String(), v.DexLocked.String(), v.DexOther.String())
		overAllSum.Add(overAllSum, v.Sum)
	}
	fmt.Printf(">>>>>>>>>>>>>>>>>>>>> valid size %d, overAllSum %s\n", len(resList), overAllSum.String())
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

func (f Dumper) DumpStakeForMiningAddresses(snapshotHeight uint64) (err error) {
	var (
		snapshotBlock *ledger.SnapshotBlock
		lastKey = types.ZERO_HASH.Bytes()
		addresses []*types.Address
		addressesRes = make(map[types.Address]string)
		pageSize = 100
		v1Size = 0
		v2Size = 0
	)
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return
	}
	for {
		if addresses, lastKey, err = f.chain.GetDexFundStakeForMiningV1ListByPage(snapshotBlock.Hash, lastKey, pageSize); err != nil {
			f.log.Error("DumpStakeForMiningAddresses failed, error is "+err.Error(), "method", "GetDexFundStakeForMiningV1ListByPage")
			return
		}
		v1Size += len(addresses)
		for _, addr := range addresses {
			addressesRes[*addr] = ""
		}
		if len(addresses) < pageSize {
			break
		}
	}
	lastKey = types.ZERO_HASH.Bytes()
	for {
		if addresses, lastKey, err = f.chain.GetDexFundStakeForMiningV2ListByPage(snapshotBlock.Hash, lastKey, pageSize); err != nil {
			f.log.Error("DumpStakeForMiningAddresses failed, error is "+err.Error(), "method", "GetDexFundStakeForMiningV2ListByPage")
			return
		}
		v2Size += len(addresses)
		for _, addr := range addresses {
			addressesRes[*addr] = ""
		}
		if len(addresses) < pageSize {
			break
		}
	}
	fmt.Printf(">>>>>>>>>>>>>>>>>>>>> DumpStakeForMiningAddresses totalSize %d, v1Size %d, v2Size %d\n", len(addressesRes), v1Size, v2Size)
	for k, _ := range addressesRes {
		fmt.Printf("%s\n", k.String())
	}
	return
}

type SnapshotBalance struct {
	WalletBalance *big.Int `json:"walletBalance"`
	DexBalance    *big.Int `json:"dexBalance"`
}

type DumpedAmountSorter []*DumpedAmount

func (st DumpedAmountSorter) Len() int {
	return len(st)
}

func (st DumpedAmountSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st DumpedAmountSorter) Less(i, j int) bool {
	addCmp := st[i].Sum.Cmp(st[j].Sum)
	if addCmp > 0 {
		return true
	} else {
		return false
	}
}
