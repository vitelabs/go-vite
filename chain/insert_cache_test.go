package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/vm_db"
	"math/rand"
	"testing"

	"net/http"
	_ "net/http/pprof"
)

func BmInsertAccountBlockCache(b *testing.B, accountNumber int) {
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
		chainInstance.cache.InsertAccountBlock(tx.AccountBlock)

		if err := chainInstance.indexDB.InsertAccountBlock(tx); err != nil {
			b.Fatal(err)
		}

		if err := chainInstance.stateDB.Write(tx); err != nil {
			b.Fatal(err)
		}

		if i%2 == 0 {
			snapshotBlock := createSnapshotBlock(chainInstance)
			chainInstance.InsertSnapshotBlock(snapshotBlock)

		}
		b.StopTimer()
	}

	if err := chainInstance.Destroy(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkChain_InsertAccountBlockCache(b *testing.B) {
	go func() {
		http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", 8080), nil)
	}()

	b.Run("10 accounts", func(b *testing.B) {
		BmInsertAccountBlockCache(b, 10)
	})
	b.Run("100 accounts", func(b *testing.B) {
		BmInsertAccountBlockCache(b, 100)
	})
	b.Run("1000 accounts", func(b *testing.B) {
		BmInsertAccountBlockCache(b, 1000)
	})
	b.Run("10000 accounts", func(b *testing.B) {
		BmInsertAccountBlockCache(b, 1000)
	})
	b.Run("100000 accounts", func(b *testing.B) {
		BmInsertAccountBlockCache(b, 100000)
	})

	//b.Run("1000000 accounts", func(b *testing.B) {
	//	BmInsertAccountBlock(b, 1000000)
	//})
}
