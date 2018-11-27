package chain_benchmark

import (
	"github.com/vitelabs/go-vite/vm_context"
	"math/rand"
	"testing"
	"time"
)

func Benchmark_InsertAccountBlock(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", true)
	const (
		ACCOUNT_NUMS        = 10
		ACCOUNT_BLOCK_LIMIT = 1000 * 10000

		PRINT_PER_COUNT               = 10 * 10000
		CREATE_REQUEST_TX_PROBABILITY = 50

		LOOP_INSERT_SNAPSHOTBLOCK = true
	)

	cTxOptions := &createTxOptions{
		mockVmContext: true,
		mockSignature: true,
	}

	tps := newTps(tpsOption{
		name:          "insertAccountBlock",
		printPerCount: PRINT_PER_COUNT,
	})

	accounts := makeAccounts(ACCOUNT_NUMS, chainInstance)
	accountLength := len(accounts)

	tps.Start()

	var loopTermial chan struct{}
	if LOOP_INSERT_SNAPSHOTBLOCK {
		loopTermial = loopInsertSnapshotBlock(chainInstance, time.Second)
	}

	for tps.Ops() < ACCOUNT_BLOCK_LIMIT {
		for _, account := range accounts {
			createRequestTx := true

			if account.HasUnreceivedBlock() {
				randNum := rand.Intn(100)
				if randNum > CREATE_REQUEST_TX_PROBABILITY {
					createRequestTx = false
				}
			}
			var tx []*vm_context.VmAccountBlock
			if createRequestTx {
				toAccount := accounts[rand.Intn(accountLength)]
				tx = account.createRequestTx(toAccount, cTxOptions)
			} else {
				tx = account.createResponseTx(cTxOptions)
			}

			chainInstance.InsertAccountBlocks(tx)
			tps.doOne()
		}
	}
	loopTermial <- struct{}{}
	tps.Stop()
	tps.Print()
}
