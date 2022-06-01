package integration

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/v2/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	chain_test_tools "github.com/vitelabs/go-vite/v2/ledger/chain/test_tools"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
	"github.com/vitelabs/go-vite/v2/ledger/verifier"
	"github.com/vitelabs/go-vite/v2/vm"
	"github.com/vitelabs/go-vite/v2/vm/quota"
)

func TestTmpInsert(t *testing.T) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	//snapshotPerNum := 0
	quota.InitQuotaConfig(true, true)
	vm.InitVMConfig(true, true, false, false, "")

	chainInstance, tempDir := test_tools.NewTestChainInstance(t.Name(), true, nil)
	defer test_tools.ClearChain(chainInstance, tempDir)

	//accountVerifier := verifier.NewAccountVerifier(chainInstance, &test_tools.MockConsensus{})
	//snapshotVerifier := verifier.NewSnapshotVerifier(chainInstance, &test_tools.MockCssVerifier{})
	//verify := verifier.NewVerifier(snapshotVerifier, accountVerifier)

	accounts := MakeAccounts(chainInstance, 100)

	sbCount := 0
	abCount := 0

	t.Run("CheckAndInsert", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			//if i%snapshotPerNum == 0 {
			sbCount++
			snapshotBlock := createSnapshotBlock(chainInstance, true)

			//stat := snapshotVerifier.VerifyReferred(snapshotBlock)

			//queryTime := snapshotBlock.Timestamp.Add(-75 * time.Second)
			//chainInstance.GetSnapshotHeaderBeforeTime(&queryTime)
			//chainInstance.GetRandomSeed(snapshotBlock.Hash, 25)
			//
			//if stat.VerifyResult() != verifier.SUCCESS {
			//	panic(stat.ErrMsg())
			//}
			_, err := chainInstance.InsertSnapshotBlock(snapshotBlock)

			if err != nil {
				panic(err)
			}

			if sbCount%1000 == 0 {
				fmt.Println("sbCount", sbCount)
			}

			//}

			randN := rand.Intn(100)
			if randN > 15 {
				continue
			}
			abCount++

			// get random account
			var vmBlock *interfaces.VmAccountBlock

			var account *Account

			for {
				account = getRandomAccount(accounts)

				// create vm block
				vmBlock, err = createVmBlock(account, accounts)
				if err != nil {
					panic(err)
				}

				if vmBlock != nil {
					break
				}
			}

			//latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
			//if vmBlock.AccountBlock.Height > 1 {
			//	_, blocks, err := verify.VerifyPoolAccountBlock(vmBlock.AccountBlock, &latestSnapshotBlock.Hash)
			//	if err != nil {
			//		panic(err)
			//	}
			//	if blocks == nil {
			//		panic("verify error!")
			//	}
			//}

			// insert vm block
			account.InsertBlock(vmBlock, accounts)

			// insert vm block to chain
			if err := chainInstance.InsertAccountBlock(vmBlock); err != nil {
				panic(err)
			}
		}
	})

	indexDB, _, stateDB := chainInstance.DBs()

	// compact
	indexDB.Store().CompactRange(util.Range{})
	// compact
	stateDB.Store().CompactRange(util.Range{})

	fmt.Println(chainInstance.GetLatestSnapshotBlock().Height)
	fmt.Println("sb count", sbCount, "ab count", abCount)
}

func BenchmarkInsert(b *testing.B) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	snapshotPerNum := 1
	quota.InitQuotaConfig(true, true)
	vm.InitVMConfig(true, true, false, false, "")

	chainInstance, tempDir := test_tools.NewTestChainInstance(b.Name(), true, nil)
	defer test_tools.ClearChain(chainInstance, tempDir)

	verify := verifier.NewVerifier(chainInstance).Init(chain_test_tools.NewVerifier(), nil, nil)

	accounts := MakeAccounts(chainInstance, 100)

	b.Run("CheckAndInsert", func(b *testing.B) {
		for i := 1; i <= b.N; i++ {
			if i%snapshotPerNum == 0 {
				snapshotBlock := createSnapshotBlock(chainInstance, true)
				stat := verify.VerifyReferred(snapshotBlock)

				queryTime := snapshotBlock.Timestamp.Add(-75 * time.Second)
				chainInstance.GetSnapshotHeaderBeforeTime(&queryTime)
				chainInstance.GetRandomSeed(snapshotBlock.Hash, 25)

				if stat.VerifyResult() != verifier.SUCCESS {
					panic(stat.ErrMsg())
				}
				_, err := chainInstance.InsertSnapshotBlock(snapshotBlock)

				if err != nil {
					panic(err)
				}
				continue
			}

			// get random account
			var vmBlock *interfaces.VmAccountBlock

			var account *Account

			for {
				account = getRandomAccount(accounts)

				// create vm block
				vmBlock, err := createVmBlock(account, accounts)
				if err != nil {
					panic(err)
				}

				if vmBlock != nil {
					break
				}
			}

			latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
			if vmBlock.AccountBlock.Height > 1 {
				_, blocks, err := verify.VerifyPoolAccountBlock(vmBlock.AccountBlock, latestSnapshotBlock)
				if err != nil {
					panic(err)
				}
				if blocks == nil {
					panic("verify error!")
				}
			}

			// insert vm block
			account.InsertBlock(vmBlock, accounts)

			// insert vm block to chain
			if err := chainInstance.InsertAccountBlock(vmBlock); err != nil {
				panic(err)
			}
		}
	})
	fmt.Println(chainInstance.GetLatestSnapshotBlock().Height)
}

func createVmBlock(account *Account, accounts map[types.Address]*Account) (*interfaces.VmAccountBlock, error) {

	var vmBlock *interfaces.VmAccountBlock
	var createBlockErr error

	latestHeight := account.GetLatestHeight()
	if latestHeight < 1 {
		return account.CreateReceiveBlock()
	}

	isCreateSendBlock := true

	if account.HasOnRoadBlock() {
		randNum := rand.Intn(100)
		if randNum > 40 {
			isCreateSendBlock = false
		}
	}

	if isCreateSendBlock {
		// query to account
		toAccount := getRandomAccount(accounts)

		vmBlock, createBlockErr = account.CreateSendBlock(toAccount)
	} else {

		vmBlock, createBlockErr = account.CreateReceiveBlock()
	}

	if createBlockErr != nil {
		return nil, createBlockErr
	}
	return vmBlock, nil
}

func getRandomAccount(accounts map[types.Address]*Account) *Account {
	var account *Account

	for _, tmpAccount := range accounts {
		account = tmpAccount
		break
	}
	return account

}

func createSnapshotContent(chainInstance chain.Chain) ledger.SnapshotContent {
	unconfirmedBlocks := chainInstance.GetAllUnconfirmedBlocks()

	sc := make(ledger.SnapshotContent)

	for i := len(unconfirmedBlocks) - 1; i >= 0; i-- {
		block := unconfirmedBlocks[i]
		if _, ok := sc[block.AccountAddress]; !ok {
			sc[block.AccountAddress] = &ledger.HashHeight{
				Hash:   block.Hash,
				Height: block.Height,
			}
		}
	}

	return sc
}

func createSnapshotBlock(chainInstance chain.Chain, snapshotAll bool) *ledger.SnapshotBlock {
	latestSb := chainInstance.GetLatestSnapshotBlock()

	sbNow := latestSb.Timestamp.Add(time.Second)

	sb := &ledger.SnapshotBlock{
		PrevHash:        latestSb.Hash,
		Height:          latestSb.Height + 1,
		Timestamp:       &sbNow,
		SnapshotContent: createSnapshotContent(chainInstance),
	}
	sb.Hash = sb.ComputeHash()
	return sb

}
