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

func (f Dumper) DumpBalance(token types.TokenTypeId, threshold *big.Int) (error) {
	res := make(map[types.Address]*big.Int, 100)
	f.chain.IterateAccounts(func(addr types.Address, accountId uint64, err1 error) bool {
		if err1 != nil {
			return false
		}
		if balance, err2 := f.chain.GetBalance(addr, token); err2 == nil {
			if balance != nil && balance.Sign() > 0 {
				res[addr] = balance
			}
		} else {
			f.log.Error("GetLatestAccountBlock failed, error is "+err2.Error(), "method", "DumpBalance")
			return false
		}
		return true
	})
	db, err := getVmDb(f.chain, types.AddressDexFund)
	if err != nil {
		return err
	}
	iterator, err := db.NewStorageIterator([]byte("fd:"))
	if err != nil {
		return err
	}
	defer iterator.Release()
	var userAccountValue []byte
	for {
		if ok := iterator.Next(); ok {
			userAccountValue = iterator.Value()
			if len(userAccountValue) == 0 {
				continue
			}
		} else {
			break
		}
		userFund := &dex.Fund{}
		if err = userFund.DeSerialize(userAccountValue); err != nil {
			return err
		}
		for _, acc := range userFund.Accounts {
			if bytes.Equal(acc.Token, token.Bytes()) {
				dexAmt := dex.AddBigInt(acc.Available, acc.Locked)
				if token == dex.VxTokenId {
					vxLocked := dex.AddBigInt(acc.VxLocked, acc.VxUnlocking)
					dexAmt = dex.AddBigInt(dexAmt, vxLocked)
				} else if token == ledger.ViteTokenId && len(acc.CancellingStake) > 0 {
					dexAmt = dex.AddBigInt(dexAmt, acc.CancellingStake)
				}
				if len(dexAmt) == 0 {
					continue
				}
				address, _ := types.BytesToAddress(userFund.Address)
				if accAmt, ok := res[address]; ok {
					res[address] = new(big.Int).Add(accAmt, new(big.Int).SetBytes(dexAmt))
				} else {
					res[address] = new(big.Int).SetBytes(dexAmt)
				}
			}
		}
	}
	var validNum = 0
	for k, v := range res {
		if v.Cmp(threshold) > 0 {
			fmt.Printf("%s,%s\n", k.String(), v.String())
			validNum++
		}
	}
	fmt.Printf(">>>>>>>>>>>>>>>>>>>>> valid size %d\n", validNum)
	return nil
}
