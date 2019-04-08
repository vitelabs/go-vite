package chain

import (
	"fmt"

	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"

	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"testing"
	"time"
)

func createSnapshotBlock(chainInstance Chain) *ledger.SnapshotBlock {
	latestSb := chainInstance.GetLatestSnapshotBlock()
	var now time.Time
	randomNum := rand.Intn(100)
	if randomNum < 70 {
		now = latestSb.Timestamp.Add(time.Second)
	} else if randomNum < 90 {
		now = latestSb.Timestamp.Add(2 * time.Second)
	} else {
		now = latestSb.Timestamp.Add(3 * time.Second)
	}

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
	accounts := MakeAccounts(chainInstance, accountNumber)

	addrList := make([]types.Address, 0, len(accounts))
	for _, account := range accounts {
		addrList = append(addrList, account.addr)
	}

	fmt.Printf("Account number is %d\n", accountNumber)

	cTxOptions := &CreateTxOptions{
		MockSignature: true,
	}

	for i := 0; i < b.N; i++ {
		account := accounts[addrList[rand.Intn(accountNumber)]]
		createRequestTx := true

		if account.HasOnRoadBlock() {
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

	if err := chainInstance.Stop(); err != nil {
		b.Fatal(err)
	}

	if err := chainInstance.Destroy(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkChain_InsertStateDB(b *testing.B) {

}

func BenchmarkChain_InsertAccountBlock(b *testing.B) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	b.Run("10 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10, 1)
	})
	b.Run("100 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100, 1)
	})
	b.Run("1000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 1000, 1)
	})
	b.Run("10000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 10000, 1)
	})
	b.Run("100000 accounts", func(b *testing.B) {
		BmInsertAccountBlock(b, 100000, 1)
	})
	//b.Run("1000000 accounts", func(b *testing.B) {
	//	BmInsertAccountBlock(b, 1000000)
	//})
}

func InsertAccountBlock(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, txCount int, snapshotPerBlockNum int) []*ledger.SnapshotBlock {
	addrList := make([]types.Address, 0, len(accounts))
	for _, account := range accounts {
		addrList = append(addrList, account.addr)
	}
	accountNumber := len(accounts)

	snapshotBlockList := make([]*ledger.SnapshotBlock, 0)

	for i := 1; i <= txCount; i++ {
		account := accounts[addrList[rand.Intn(accountNumber)]]
		tx, err := createVmBlock(account, accounts, addrList)
		if err != nil {
			t.Fatal(err)
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
				if account, ok := accounts[addr]; ok {
					account.Snapshot(sb.Hash)
				}
			}
			snapshotBlockList = append(snapshotBlockList, sb)
		}

	}
	return snapshotBlockList
}

func createVmBlock(account *Account, accounts map[types.Address]*Account, addrList []types.Address) (*vm_db.VmAccountBlock, error) {
	createRequestTx := true
	var vmLogList ledger.VmLogList

	topicHash1, err := types.BytesToHash(crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))))
	if err != nil {
		return nil, err
	}
	topicHash2, err := types.BytesToHash(crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))))
	if err != nil {
		return nil, err
	}
	vmLogList = append(vmLogList, &ledger.VmLog{
		Topics: []types.Hash{topicHash1},
		Data:   chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())),
	})
	vmLogList = append(vmLogList, &ledger.VmLog{
		Topics: []types.Hash{topicHash2},
		Data:   chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())),
	})

	keyValue := make(map[string][]byte, 10)
	for i := 0; i < 1; i++ {
		keyValue[strconv.FormatUint(uint64(time.Now().UnixNano()), 10)] = chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))
	}
	cTxOptions := &CreateTxOptions{
		MockSignature: true,
		VmLogList:     vmLogList,
		KeyValue:      keyValue,
		Quota:         rand.Uint64() % 10000,
	}

	if account.HasOnRoadBlock() {
		randNum := rand.Intn(100)
		if randNum > 50 {
			createRequestTx = false
		}
	}

	var tx *vm_db.VmAccountBlock
	var createTxErr error

	if createRequestTx {
		toAccount := accounts[addrList[rand.Intn(len(addrList))]]
		if len(toAccount.SendBlocksMap) <= 0 && len(toAccount.ReceiveBlocksMap) <= 0 {
			cTxOptions.ContractMeta = &ledger.ContractMeta{
				SendConfirmedTimes: 2,
				Gid:                types.DataToGid(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))),
			}
			tx, createTxErr = account.CreateRequestTx(toAccount, cTxOptions)
		} else {
			tx, createTxErr = account.CreateRequestTx(toAccount, cTxOptions)
		}

	} else {
		tx, createTxErr = account.CreateResponseTx(cTxOptions)
	}
	if createTxErr != nil {
		return nil, createTxErr
	}
	return tx, nil
}
