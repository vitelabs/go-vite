package subcmd_ledger

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"sort"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
)

type dumpedAmount struct {
	Address      *types.Address
	IsContract bool
	Sum          *big.Int
	WalletAmount *big.Int
	DexAvailable *big.Int
	DexLocked    *big.Int
	DexOther     *big.Int // 1. VITE - stake mining in cancelling status; 2. VX - locked for reward or in cancelling status
	DexMining  	 *big.Int
	DexVIP     	 *big.Int
	DexPending   *big.Int // stake mining submitted but not confirmed yet

	UnReceiveAmount *big.Int
	StakeForQuotaAmount *big.Int
	LockForSBPAmount *big.Int
}

func newDumpedAmount(addr types.Address) *dumpedAmount {
	return &dumpedAmount{&addr, false, big.NewInt(0), big.NewInt(0),  big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
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

func dumpBalance(chain chain.Chain, token types.TokenTypeId, snapshotHeight uint64, minAirdrop string, allowContract bool, showBuiltIn bool, ignoreList string, totalAirdrop string, airdropQuota string) error {
	var snapshotBlock *core.SnapshotBlock
	var err error
	if snapshotBlock, err = chain.GetSnapshotBlockByHeight(snapshotHeight); err != nil {
		return err
	}
	if snapshotBlock == nil {
		return fmt.Errorf("snapshot block is not exist, height:%d", snapshotHeight)
	}
	minSingleAirdrop, ok := big.NewInt(0).SetString(minAirdrop, 10)
	if !ok {
		return fmt.Errorf("invalid minimum balance :%s", minSingleAirdrop)
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
			dumpAmt.Sum = big.NewInt(0).Add(dumpAmt.Sum, balance)
			//res[addr] = dumpAmt
		}
		if roadInfo, err := chain.GetAccountOnRoadInfo(addr); err != nil {
			log.Error(fmt.Sprintf("GetAccountOnRoadInfo failed, error is %s", err.Error()), "method", "DumpBalance")
		} else if roadTokenInfo, ok := roadInfo.TokenBalanceInfoMap[token]; ok && roadTokenInfo.TotalAmount.Sign() > 0 {
			dumpAmt := newDumpedAmount(addr)
			if accAmt, ok := res[addr]; ok {
				dumpAmt = accAmt
			} else {
				res[addr] = dumpAmt
			}
			dumpAmt.UnReceiveAmount = &roadTokenInfo.TotalAmount
			dumpAmt.Sum = big.NewInt(0).Add(dumpAmt.Sum, dumpAmt.UnReceiveAmount)
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

							//walletAmt := dumpAmt.WalletAmount
							sum := dumpAmt.Sum
							//sum = sum.Add(sum, walletAmt)
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

	// stake for quota & sbp
	if token == core.ViteTokenId {
		// stake for quota
		lastKeyBytes, err := hex.DecodeString("")
		if err != nil {
			return err
		}
		for {
			if funds, nextKeyBytes, err := chain.GetStakeListByPage(snapshotBlock.Hash, lastKeyBytes, uint64(pageSize)); err != nil {
				return err
			} else {
				for _, fund := range funds {
					stakedAmt := fund.Amount
					if stakedAmt.Sign() > 0 {
						address := fund.StakeAddress
						dumpAmt := newDumpedAmount(address)
						if accAmt, ok := res[address]; ok {
							dumpAmt = accAmt
						} else {
							res[address] = dumpAmt
						}

						//walletAmt := dumpAmt.WalletAmount
						sum := dumpAmt.Sum
						//sum = sum.Add(sum, walletAmt)

						dumpAmt.StakeForQuotaAmount = big.NewInt(0).Add(dumpAmt.StakeForQuotaAmount, stakedAmt)
						sum = sum.Add(sum, stakedAmt)
						dumpAmt.Sum = sum
					}
				}
				fundsLen := len(funds)
				if fundsLen < pageSize {
					break
				}

				lastKeyBytes = nextKeyBytes
			}
		}

		// dex stake for mining & vip
		lastKeyBytes, err = hex.DecodeString("")
		if err != nil {
			return err
		}
		for {
			if funds, nextKeyBytes, err := chain.GetDexStakeListByPage(snapshotBlock.Hash, lastKeyBytes, pageSize); err != nil {
				return err
			} else {
				for _, fund := range funds {
					stakedAmt := new(big.Int).SetBytes(fund.Amount)
					stakeType := fund.StakeType
					status := fund.Status
					if stakedAmt.Sign() > 0 {
						address, _ := types.BytesToAddress(fund.Address)
						dumpAmt := newDumpedAmount(address)
						if accAmt, ok := res[address]; ok {
							dumpAmt = accAmt
						} else {
							res[address] = dumpAmt
						}

						sum := dumpAmt.Sum

						if status == 2 { // confirmed
							if stakeType == 1 {
								dumpAmt.DexMining = big.NewInt(0).Add(dumpAmt.DexMining, stakedAmt)
							} else {
								dumpAmt.DexVIP = big.NewInt(0).Add(dumpAmt.DexVIP, stakedAmt)
							}
						} else if status == 1 { // submitted
							dumpAmt.DexPending = big.NewInt(0).Add(dumpAmt.DexPending, stakedAmt)
						}

						sum = sum.Add(sum, stakedAmt)
						dumpAmt.Sum = sum
					}
				}
				fundsLen := len(funds)
				if fundsLen < pageSize {
					break
				}

				lastKeyBytes = nextKeyBytes
			}
		}

		// sbp
		if funds, err := chain.GetRegisterList(snapshotBlock.Hash, types.SNAPSHOT_GID); err != nil {
			return err
		} else {
			for _, fund := range funds {
				if fund.IsActive() {
					stakedAmt := fund.Amount
					if stakedAmt.Sign() > 0 {
						address := fund.StakeAddress
						dumpAmt := newDumpedAmount(address)
						if accAmt, ok := res[address]; ok {
							dumpAmt = accAmt
						} else {
							res[address] = dumpAmt
						}

						sum := dumpAmt.Sum

						dumpAmt.LockForSBPAmount = big.NewInt(0).Add(dumpAmt.LockForSBPAmount, stakedAmt)
						sum = sum.Add(sum, stakedAmt)
						dumpAmt.Sum = sum
					}
				}
			}
		}
	}

	resList := make([]*dumpedAmount, len(res))
	overAllSum := big.NewInt(0)
	overAllQuotaSum := big.NewInt(0)
	i := 0
	for k, v := range res {
		v.IsContract = types.IsContractAddr(k)
		resList[i] = v
		i++
		if !types.IsBuiltinContractAddr(k) { // exclude built-in contracts from total amount
			overAllSum.Add(overAllSum, v.Sum) // all sum
			if token == core.ViteTokenId {
				overAllQuotaSum.Add(overAllQuotaSum, v.StakeForQuotaAmount) // all quota sum
			}
		}
	}
	sort.Sort(dumpedAmountSorter(resList))

	bList, err := readFile(ignoreList)
	if err != nil {
		return err
	}

	airdropAmt, ok := big.NewInt(0).SetString(totalAirdrop, 10)
	oneToOne := false
	if !ok {
		return fmt.Errorf("invalid airdrop amount :%s", airdropAmt)
	}
	if airdropAmt.Cmp(big.NewInt(0)) == 0 {
		oneToOne = true
	}
	quotaAmt, ok := big.NewInt(0).SetString(airdropQuota, 10)
	if !ok {
		return fmt.Errorf("invalid airdrop amount :%s", quotaAmt)
	}
	// print result
	printLog := make([]string, 0)
	airdropList := make([]string, 0)
	quotaList := make([]string, 0)

	printLog = append(printLog, fmt.Sprint("address, isContract, sum , wallet , dexAvailable, dexLocked, dexOther, dexMining, dexVIP, unReceived, stakedQuota, sbpLock"))

	airdropSum := big.NewInt(0)
	validSum := big.NewInt(0)
	validNum := 0
	for _, v := range resList {
		if v.IsContract {
			if types.IsBuiltinContractAddr(types.HexToAddressPanic(v.Address.String())) && !showBuiltIn { // exclude built-in contracts
				continue
			}
			if !allowContract { // remove contract address
				continue
			}
		}

		found := false
		for _, addr := range bList { // remove blacklist addresses
			if addr == v.Address.String() {
				found = true
				break
			}
		}
		if found {
			continue
		}

		// print address balance
		printLog = append(printLog, fmt.Sprintf("%s,%t,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s", v.Address.String(), v.IsContract, v.Sum.String(), v.WalletAmount.String(),
			v.DexAvailable.String(), v.DexLocked.String(), v.DexOther.String(), v.DexMining.String(), v.DexVIP.String(),
			v.UnReceiveAmount.String(), v.StakeForQuotaAmount.String(), v.LockForSBPAmount.String()))

		// calculate airdrop for staking quota
		if token == core.ViteTokenId && v.StakeForQuotaAmount.Sign() > 0 {
			quotaAirdrop := new(big.Int).Set(v.StakeForQuotaAmount)
			quotaAirdrop.Mul(quotaAirdrop, quotaAmt)
			quotaAirdrop.Div(quotaAirdrop, overAllQuotaSum)

			quotaList = append(quotaList, fmt.Sprintf("%s,%s", v.Address.String(), quotaAirdrop))
		}

		// calculate airdrop for eligible addresses
		addressAirdrop := new(big.Int).Set(v.Sum)
		if !oneToOne {
			addressAirdrop.Mul(addressAirdrop, airdropAmt)
			addressAirdrop.Div(addressAirdrop, overAllSum)
		}
		if addressAirdrop.Cmp(minSingleAirdrop) < 0 {
			continue
		}
		validNum++
		airdropList = append(airdropList, fmt.Sprintf("%s,%s", v.Address.String(), addressAirdrop))

		validSum.Add(validSum, v.Sum)
		airdropSum.Add(airdropSum, addressAirdrop)
	}
	printLog = append(printLog, fmt.Sprintf(">>>>>>>>>>>>>>>>>>>>> valid size %d, validSum %s, airdropSum %s", validNum, validSum.String(), airdropSum.String()))
	printLog = append(printLog, fmt.Sprintf(">>>>>>>>>>>>>>>>>>>>> all size %d, overAllSum %s", len(resList), overAllSum.String()))

	err = writeFile(fmt.Sprintf("balances.%s.%d.%t.csv", token.String(), snapshotHeight, allowContract), printLog)
	if err != nil {
		return err
	}
	if !showBuiltIn { // airdrop amount makes no sense when built-in contracts are displayed
		err = writeFile(fmt.Sprintf("airdrop.%s.%d.%t.%s.csv", token.String(), snapshotHeight, allowContract, totalAirdrop), airdropList)
		if err != nil {
			return err
		}
	}
	if token == core.ViteTokenId && quotaAmt.Sign() > 0 {
		err = writeFile(fmt.Sprintf("quota.%d.%t.%s.csv", snapshotHeight, allowContract, airdropQuota), quotaList)
		if err != nil {
			return err
		}
	}
	return nil
}

func readFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func writeFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}
