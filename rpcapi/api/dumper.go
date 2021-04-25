package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sort"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
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

	UnReceiveAmount *big.Int
}

func newDumpedAmount(addr types.Address) *DumpedAmount {
	return &DumpedAmount{&addr, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
}

func (f Dumper) DumpBalance(token types.TokenTypeId, snapshotHeight uint64) error {
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
		if roadInfo, err := f.chain.GetAccountOnRoadInfo(addr); err != nil {
			f.log.Error(fmt.Sprintf("GetAccountOnRoadInfo failed, error is %s", err.Error()), "method", "DumpBalance")
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

	resList := make([]*DumpedAmount, len(res))
	i := 0
	for _, v := range res {
		resList[i] = v
		i++
	}
	sort.Sort(DumpedAmountSorter(resList))
	fmt.Println("address, sum , wallet , dexAvailable, dexLocked, dexOther, unReceived")
	overAllSum := big.NewInt(0)
	for _, v := range resList {
		fmt.Printf("%s,%s,%s,%s,%s,%s,%s\n", v.Address.String(), v.Sum.String(), v.WalletAmount.String(), v.DexAvailable.String(), v.DexLocked.String(), v.DexOther.String(), v.UnReceiveAmount.String())
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

func (f Dumper) DumpStakeForMiningAddresses(snapshotHeight uint64, outputFile string) (err error) {
	var (
		snapshotBlock *ledger.SnapshotBlock
		lastKey       = types.ZERO_HASH.Bytes()
		addresses     []*types.Address
		addressesRes  = make(map[types.Address]string)
		addressesRes2 = make(map[types.Address]string)
		pageSize      = 100
		v1Size        = 0
		v2Size        = 0
	)
	f.log.Error("DumpStakeForMiningAddresses start!!!")
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		f.log.Error("DumpStakeForMiningAddresses failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
		return
	}
	if snapshotBlock, err = f.chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		f.log.Error("DumpStakeForMiningAddresses failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
		return
	}
	for {
		if addresses, lastKey, err = f.chain.GetDexFundStakeForMiningV1ListByPage(snapshotBlock.Hash, lastKey, pageSize); err != nil {
			f.log.Error("DumpStakeForMiningAddresses failed, error is "+err.Error(), "method", "GetDexFundStakeForMiningV1ListByPage")
			return
		}
		v1Size += len(addresses)
		f.log.Error("DumpStakeForMiningAddresses success", "method", "GetDexFundStakeForMiningV1ListByPage", "len(addresses)s", len(addresses))
		duplicated := false
		for _, addr := range addresses {
			f.log.Error("DumpStakeForMiningAddresses success", "method", "GetDexFundStakeForMiningV1ListByPage", "address", addr.String())
			if _, ok := addressesRes[*addr]; ok {
				duplicated = true
				break
			} else {
				addressesRes[*addr] = ""
			}
		}
		if len(addresses) < pageSize || duplicated {
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
		f.log.Error("DumpStakeForMiningAddresses success", "method", "GetDexFundStakeForMiningV2ListByPage", "len(addresses)s", len(addresses))
		duplicated := false
		for _, addr := range addresses {
			f.log.Error("DumpStakeForMiningAddresses success", "method", "GetDexFundStakeForMiningV2ListByPage", "address", addr.String())
			if _, ok := addressesRes2[*addr]; ok {
				duplicated = true
				break
			} else {
				addressesRes2[*addr] = ""
				addressesRes[*addr] = ""
			}
		}
		if len(addresses) < pageSize || duplicated {
			break
		}
	}

	if err = os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
		return err
	}
	var file *os.File
	file, err = ioutil.TempFile(filepath.Dir(outputFile), "."+filepath.Base(outputFile)+".tmp")
	if err != nil {
		return err
	}
	if err = writeLineToFile(file, fmt.Sprintf(">>>>>>>>>>>>>>>>>>>>>  totalSize %d, v1Size %d, v2Size %d\n", len(addressesRes), v1Size, v2Size)); err != nil {
		return
	}
	for k, _ := range addressesRes {
		if err = writeLineToFile(file, fmt.Sprintf("%s\n", k.String())); err != nil {
			return
		}
	}
	file.Close()
	return os.Rename(file.Name(), outputFile)
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

func writeLineToFile(file *os.File, content string) error {
	if _, err := file.Write([]byte(content)); err != nil {
		file.Close()
		os.Remove(file.Name())
		return err
	}
	return nil
}
