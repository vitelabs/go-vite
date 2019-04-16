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

	"sync"
	"testing"
	"time"
)

func TestInsertAccountBlocks(t *testing.T) {
	chainInstance, accounts, snapshotBlockList := SetUp(5, 98, 2)
	addrList := make([]types.Address, 0, len(accounts))
	for addr := range accounts {
		addrList = append(addrList, addr)
	}

	t.Run("InsertAccountBlockAndSnapshot", func(t *testing.T) {
		snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, 17, 7, false)...)
	})

	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(chainInstance, accounts, snapshotBlockList)
	})

	//}

	TearDown(chainInstance)
}

func InsertSnapshotBlock(chainInstance *chain, snapshotAll bool) (*ledger.SnapshotBlock, []*ledger.AccountBlock, error) {
	sb := createSnapshotBlock(chainInstance, snapshotAll)
	invalidBlocks, err := chainInstance.InsertSnapshotBlock(sb)
	if err != nil {
		return nil, nil, err
	}

	return sb, invalidBlocks, nil
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
		addrList = append(addrList, account.Addr)
	}

	fmt.Printf("Account number is %d, snapshotPerNum is %d\n", accountNumber, snapshotPerBlockNum)

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
			tx, err = account.CreateSendBlock(toAccount, cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		} else {
			tx, err = account.CreateReceiveBlock(cTxOptions)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StartTimer()
		if snapshotPerBlockNum > 0 && i%snapshotPerBlockNum == 0 {
			_, _, err := InsertSnapshotBlock(chainInstance, false)
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

func BenchmarkChain_InsertAccountBlock(b *testing.B) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	for _, snapshotPerNum := range []int{0} {
		for _, accountNum := range []int{1, 10, 100, 10000, 10000} {
			b.Run(fmt.Sprintf("%d accounts, snapshotPerNum: %d", accountNum, snapshotPerNum), func(b *testing.B) {
				BmInsertAccountBlock(b, accountNum, snapshotPerNum)
			})
		}
	}

	//b.Run("1000000 accounts", func(b *testing.B) {
	//	BmInsertAccountBlock(b, 1000000)
	//})
}

func InsertAccountBlockAndSnapshot(chainInstance *chain, accounts map[types.Address]*Account, blockCount int, snapshotPerBlockNum int, testQuery bool) []*ledger.SnapshotBlock {
	addrList := make([]types.Address, 0, len(accounts))
	for _, account := range accounts {
		addrList = append(addrList, account.Addr)
	}

	snapshotBlockList := make([]*ledger.SnapshotBlock, 0)

	countInserted := 0

	for countInserted < blockCount {
		var insertCount = snapshotPerBlockNum

		if insertCount == 0 {
			insertCount = 10
		}

		if insertCount > blockCount-countInserted {
			insertCount = blockCount - countInserted
		}

		InsertAccountBlocks(nil, chainInstance, accounts, insertCount)

		countInserted += insertCount

		// snapshot
		snapshotBlock, invalidBlocks, err := InsertSnapshotBlock(chainInstance, false)
		if err != nil {
			panic(err)
		}

		snapshotBlockList = append(snapshotBlockList, snapshotBlock)

		// snapshot
		Snapshot(accounts, snapshotBlock)
		// delete
		DeleteInvalidBlocks(accounts, invalidBlocks)

		if testQuery {
			testUnconfirmedNoTesting(chainInstance, accounts)
			testAccountBlockNoTesting(chainInstance, accounts)
			testSnapshotBlockNoTesting(chainInstance, accounts, snapshotBlockList)
		}
	}

	return snapshotBlockList
}

func InsertAccountBlocks(mu *sync.RWMutex, chainInstance *chain, accounts map[types.Address]*Account, blockCount int) {
	for i := 1; i <= blockCount; i++ {
		if mu != nil {
			mu.Lock()
		}

		// get random account
		account := getRandomAccount(accounts)

		// create vm block
		vmBlock, err := createVmBlock(account, accounts)
		if err != nil {
			panic(err)
		}

		// insert vm block
		account.InsertBlock(vmBlock, accounts)

		if mu != nil {
			mu.Unlock()
		}

		// insert vm block to chain
		if err := chainInstance.InsertAccountBlock(vmBlock); err != nil {
			panic(err)
		}
	}

}
func Snapshot(accounts map[types.Address]*Account, snapshotBlock *ledger.SnapshotBlock) {
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		if account, ok := accounts[addr]; ok {
			account.Snapshot(snapshotBlock.Hash, hashHeight)
		}
	}

}

func DeleteInvalidBlocks(accounts map[types.Address]*Account, invalidBlocks []*ledger.AccountBlock) {
	// delete invalid
	for i := len(invalidBlocks) - 1; i >= 0; i-- {
		ab := invalidBlocks[i]

		accounts[ab.AccountAddress].deleteAccountBlock(accounts, ab.Hash)
		accounts[ab.AccountAddress].rollbackLatestBlock()
	}
}

func createVmBlock(account *Account, accounts map[types.Address]*Account) (*vm_db.VmAccountBlock, error) {

	// query latest height
	latestHeight := account.GetLatestHeight()

	// FOR DEBUG
	//fmt.Printf("%s add key value: %+v\n", account.Addr, keyValue)

	cTxOptions := &CreateTxOptions{
		MockSignature: true,                         // mock signature
		KeyValue:      createKeyValue(latestHeight), // create key value
		VmLogList:     createVmLogList(),            // create vm log list
		Quota:         rand.Uint64() % 10000,
	}

	var vmBlock *vm_db.VmAccountBlock
	var createBlockErr error

	isCreateSendBlock := true

	if account.HasOnRoadBlock() {
		randNum := rand.Intn(100)
		if randNum > 50 {
			isCreateSendBlock = false
		}
	}

	if isCreateSendBlock {
		// query to account
		toAccount := getRandomAccount(accounts)

		if len(toAccount.BlocksMap) <= 0 {
			// set contract meta
			cTxOptions.ContractMeta = createContractMeta()

		}
		vmBlock, createBlockErr = account.CreateSendBlock(toAccount, cTxOptions)
	} else {

		vmBlock, createBlockErr = account.CreateReceiveBlock(cTxOptions)
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

func createVmLogList() ledger.VmLogList {
	var vmLogList ledger.VmLogList

	topicHash1, err := types.BytesToHash(crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))))
	if err != nil {
		panic(err)
	}
	topicHash2, err := types.BytesToHash(crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))))
	if err != nil {
		panic(err)
	}
	vmLogList = append(vmLogList, &ledger.VmLog{
		Topics: []types.Hash{topicHash1},
		Data:   chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())),
	})
	vmLogList = append(vmLogList, &ledger.VmLog{
		Topics: []types.Hash{topicHash2},
		Data:   chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())),
	})
	return vmLogList
}

func createKeyValue(latestHeight uint64) map[string][]byte {
	num := rand.Intn(100)
	var kv map[string][]byte
	if num <= 50 {
		kv = map[string][]byte{
			string(chain_utils.Uint64ToBytes(latestHeight + 1)): chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())),
		}
	} else {
		kv = map[string][]byte{
			string(chain_utils.Uint64ToBytes(latestHeight)): nil,
		}
	}

	return kv
}

func createContractMeta() *ledger.ContractMeta {
	return &ledger.ContractMeta{
		SendConfirmedTimes: 2,
		Gid:                types.DataToGid(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))),
	}
}
func createSnapshotContent(chainInstance *chain, snapshotAll bool) ledger.SnapshotContent {
	unconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()

	// random snapshot
	if !snapshotAll && len(unconfirmedBlocks) > 1 {
		randomNum := rand.Intn(100)

		if randomNum > 90 {
			unconfirmedBlocks = []*ledger.AccountBlock{}

		} else if randomNum > 50 {
			unconfirmedBlocks = unconfirmedBlocks[:rand.Intn(len(unconfirmedBlocks))]

		}
	}

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

func createSnapshotBlock(chainInstance *chain, snapshotAll bool) *ledger.SnapshotBlock {
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
		SnapshotContent: createSnapshotContent(chainInstance, snapshotAll),
	}
	sb.Hash = sb.ComputeHash()
	return sb

}
