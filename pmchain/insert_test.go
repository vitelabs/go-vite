package pmchain

import (
	"fmt"
	"math/rand"
	"testing"
)

func BenchmarkChain_InsertAccountBlock(b *testing.B) {
	const (
		accountNumber = 1000

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

	for _, account := range accounts {
		createRequestTx := true

		if account.HasUnreceivedBlock() {
			randNum := rand.Intn(100)
			if randNum > requestTxPercent {
				createRequestTx = false
			}
		}

		if createRequestTx {
			toAccount := accounts[rand.Intn(accountNumber)]
			_, err = account.CreateRequestTx(toAccount, cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		} else {
			_, err = account.CreateResponseTx(cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
