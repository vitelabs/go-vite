package pmchain

import (
	"fmt"
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

func InsertSnapshotBlock(chainInstance Chain) error {
	sb := createSnapshotBlock(chainInstance)
	if _, err := chainInstance.InsertSnapshotBlock(sb); err != nil {
		return err
	}

	return nil

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
	accounts := MakeAccounts(accountNumber, chainInstance)

	fmt.Printf("Account number is %d\n", accountNumber)

	cTxOptions := &CreateTxOptions{
		MockSignature: true,
	}

	for i := 0; i < b.N; i++ {
		account := accounts[rand.Intn(accountNumber)]
		createRequestTx := true

		if account.HasUnreceivedBlock() {
			randNum := rand.Intn(100)
			if randNum > requestTxPercent {
				createRequestTx = false
			}
		}

		var tx *vm_db.VmAccountBlock
		if createRequestTx {
			toAccount := accounts[rand.Intn(accountNumber)]
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
		if i%snapshotPerBlockNum == 0 {
			if err := InsertSnapshotBlock(chainInstance); err != nil {
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
		BmInsertAccountBlock(b, 10, 10000)
	})
	b.Run("100 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100, 10000)
	})
	b.Run("1000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 1000, 10000)
	})
	b.Run("10000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10000, 10000)
	})
	b.Run("100000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100000, 10000)
	})

	//b.Run("1000000 accounts", func(b *testing.B) {
	//	BmInsertAccountBlock(b, 1000000)
	//})
}
