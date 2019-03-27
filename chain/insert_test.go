package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/rand"
	"testing"
	"time"
)

func createSnapshotBlock(chainInstance Chain) *ledger.SnapshotBlock {
	latestSb := chainInstance.GetLatestSnapshotBlock()
	now := time.Now()

	sb := &ledger.SnapshotBlock{
		PrevHash:        latestSb.Hash,
		Height:          latestSb.Height + 1,
		Timestamp:       &now,
		SnapshotContent: chainInstance.GetContentNeedSnapshot(),
	}
	sb.Hash = sb.ComputeHash()
	return sb

}

func InsertSnapshotBlock(chainInstance Chain) (*ledger.SnapshotBlock, error) {
	sb := createSnapshotBlock(chainInstance)
	if _, err := chainInstance.InsertSnapshotBlock(sb); err != nil {
		return nil, err
	}

	return sb, nil
}

func BmInsertAccountBlock(b *testing.B, accountNumber int, snapshotPerBlockNum int) {
	b.StopTimer()
	const (
		requestTxPercent = 50
	)
	chainInstance, err := NewChainInstance("benchmark", true)
	if err != nil {
		b.Fatal(err)
	}
	accounts, addrList := MakeAccounts(accountNumber, chainInstance)

	fmt.Printf("Account number is %d\n", accountNumber)

	cTxOptions := &CreateTxOptions{
		MockSignature: true,
	}

	for i := 0; i < b.N; i++ {
		account := accounts[addrList[rand.Intn(accountNumber)]]
		createRequestTx := true

		if account.HasUnreceivedBlock() {
			randNum := rand.Intn(100)
			if randNum > requestTxPercent {
				createRequestTx = false
			}
		}

		var tx *vm_db.VmAccountBlock
		if createRequestTx {
			toAccount := accounts[addrList[rand.Intn(accountNumber)]]
			tx, err = account.CreateRequestTx(toAccount, cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		} else {
			tx, err = account.CreateResponseTx(cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StartTimer()
		if snapshotPerBlockNum > 0 && i%snapshotPerBlockNum == 0 {
			_, err := InsertSnapshotBlock(chainInstance)
			if err != nil {
				b.Fatal(err)
			}
		}

		if err := chainInstance.InsertAccountBlock(tx); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
	}

	if err := chainInstance.Destroy(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkChain_InsertAccountBlock(b *testing.B) {
	b.Run("10 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10, 1000)
	})
	b.Run("100 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100, 1000)
	})
	b.Run("1000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 1000, 1000)
	})
	b.Run("10000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10000, 1000)
	})
	b.Run("100000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100000, 1000)
	})
	//b.Run("1000000 accounts", func(b *testing.B) {
	//	BmInsertAccountBlock(b, 1000000)
	//})
}

func InsertAccountBlock(t *testing.T, accountNumber int, chainInstance Chain, txCount int, snapshotPerBlockNum int) (map[types.Address]*Account, []types.Hash, []types.Address, []uint64) {
	accounts, addrList := MakeAccounts(accountNumber, chainInstance)

	fmt.Printf("Account number is %d\n", accountNumber)

	cTxOptions := &CreateTxOptions{
		MockSignature: true,
	}

	var err error
	hashList := make([]types.Hash, 0, txCount)

	returnAddrList := make([]types.Address, 0, txCount)
	heightList := make([]uint64, 0, txCount)

	for i := 0; i < txCount; i++ {
		account := accounts[addrList[rand.Intn(accountNumber)]]
		createRequestTx := true

		if account.HasUnreceivedBlock() {
			randNum := rand.Intn(100)
			if randNum > 50 {
				createRequestTx = false
			}
		}

		var tx *vm_db.VmAccountBlock
		if createRequestTx {
			toAccount := accounts[addrList[rand.Intn(accountNumber)]]
			tx, err = account.CreateRequestTx(toAccount, cTxOptions)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			tx, err = account.CreateResponseTx(cTxOptions)
			if err != nil {
				t.Fatal(err)
			}
		}

		if err := chainInstance.InsertAccountBlock(tx); err != nil {
			t.Fatal(err)
		}

		if snapshotPerBlockNum > 0 && i%snapshotPerBlockNum == 0 {
			sb, err := InsertSnapshotBlock(chainInstance)
			if err != nil {
				t.Fatal(err)
			}
			for addr := range sb.SnapshotContent {
				account := accounts[addr]
				account.Snapshot(sb.Hash)
			}
		}

		hashList = append(hashList, tx.AccountBlock.Hash)
		returnAddrList = append(returnAddrList, account.addr)

		heightList = append(heightList, tx.AccountBlock.Height)

	}
	return accounts, hashList, returnAddrList, heightList
}
