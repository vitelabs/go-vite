package pmchain

import (
	"fmt"
	"github.com/vitelabs/go-vite/vm_db"
	"math/rand"
	"testing"
)

func BmInsertAccountBlock(b *testing.B, accountNumber int) {
	b.StopTimer()
	const (
		requestTxPercent = 50
	)
	chainInstance, err := NewChainInstance("benchmark")
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
		BmInsertAccountBlock(b, 10)
	})
	b.Run("100 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100)
	})
	b.Run("1000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 1000)
	})
	b.Run("10000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10000)
	})
	b.Run("100000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100000)
	})

	b.Run("1000000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 1000000)
	})
}
