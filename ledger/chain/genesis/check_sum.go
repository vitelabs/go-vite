package chain_genesis

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"sort"
)

func CheckSum(accountBlocks []*vm_db.VmAccountBlock) types.Hash {
	var sumHash types.Hash

	content := make([]byte, 0)

	sortedAccountBlocks := make(SortVmBlocks, len(accountBlocks))
	copy(sortedAccountBlocks, accountBlocks)

	sort.Sort(sortedAccountBlocks)

	for _, vmBlock := range sortedAccountBlocks {
		content = append(content, vmBlock.AccountBlock.Hash.Bytes()...)

		// balance
		vmdb := vmBlock.VmDb
		unsavedBalanceMap := vmdb.GetUnsavedBalanceMap()

		sortedBalances := make(SortBalances, 0, len(unsavedBalanceMap))
		for tokenId, balance := range unsavedBalanceMap {
			sortedBalances = append(sortedBalances, struct {
				tokenId types.TokenTypeId
				balance *big.Int
			}{tokenId: tokenId, balance: balance})
		}
		sort.Sort(sortedBalances)
		for _, item := range sortedBalances {
			content = append(content, item.tokenId.Bytes()...)
			content = append(content, item.balance.Bytes()...)
		}

		// key, value
		storage := vmdb.GetUnsavedStorage()
		for _, kv := range storage {
			content = append(content, kv[0]...)
			content = append(content, kv[1]...)
		}

	}
	sumHash, err := types.BytesToHash(crypto.Hash256(content))
	if err != nil {
		panic(err)
	}

	return sumHash
}

type SortVmBlocks []*vm_db.VmAccountBlock

func (a SortVmBlocks) Len() int      { return len(a) }
func (a SortVmBlocks) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortVmBlocks) Less(i, j int) bool {
	return bytes.Compare(a[i].AccountBlock.Hash.Bytes(), a[j].AccountBlock.Hash.Bytes()) < 0
}

type SortBalances []struct {
	tokenId types.TokenTypeId
	balance *big.Int
}

func (a SortBalances) Len() int      { return len(a) }
func (a SortBalances) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortBalances) Less(i, j int) bool {
	return bytes.Compare(a[i].tokenId.Bytes(), a[j].tokenId.Bytes()) < 0
}
