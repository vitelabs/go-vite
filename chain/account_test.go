package chain

import (
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestChain_Account(t *testing.T) {

	chainInstance, accounts, _ := SetUp(t, 1000, 1000, 8)

	testAccount(t, chainInstance, accounts)
	TearDown(chainInstance)
}

func testAccount(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {

	accountIdList := make([]uint64, 0, len(accounts))

	for addr := range accounts {
		accountId, err := chainInstance.GetAccountId(addr)
		if err != nil {
			t.Fatal(err)
		}
		if accountId <= 0 {
			t.Fatal("accountId <= 0")
		}
		queryAddr2, err := chainInstance.GetAccountAddress(accountId)
		if err != nil {
			t.Fatal(err)
		}

		if *queryAddr2 != addr {
			t.Fatal("error")
		}

		accountIdList = append(accountIdList, accountId)
	}

	for _, accountId := range accountIdList {
		addr, err := chainInstance.GetAccountAddress(accountId)
		if err != nil {
			t.Fatal(err)
		}
		if addr == nil {
			t.Fatal("addr is nil")
		}
		queryAccountId, err := chainInstance.GetAccountId(*addr)
		if err != nil {
			t.Fatal(err)
		}

		if queryAccountId != accountId {
			t.Fatal("error")
		}
	}

}

type KvSet struct {
	Height uint64
	Kv     map[string][]byte
}
type KvSetList []*KvSet

func (list KvSetList) Len() int      { return len(list) }
func (list KvSetList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list KvSetList) Less(i, j int) bool {
	return list[i].Height < list[j].Height
}

type Account struct {
	addr          types.Address
	privateKey    ed25519.PrivateKey
	publicKey     ed25519.PublicKey
	chainInstance Chain
	unreceivedMu  sync.RWMutex

	contractMetaMu sync.RWMutex
	contractMeta   *ledger.ContractMeta

	Code []byte

	UnreceivedBlocks map[types.Hash]*vm_db.VmAccountBlock

	BlocksMap         map[types.Hash]*vm_db.VmAccountBlock
	SendBlocksMap     map[types.Hash]*vm_db.VmAccountBlock
	ReceiveBlocksMap  map[types.Hash]*vm_db.VmAccountBlock
	ConfirmedBlockMap map[types.Hash]map[types.Hash]struct{}

	BalanceMap map[types.Hash]*big.Int
	KvSetMap   map[types.Hash]map[string][]byte

	LogListMap map[types.Hash]ledger.VmLogList

	unconfirmedBlocks map[types.Hash]struct{}

	latestBlock *ledger.AccountBlock
}

type CreateTxOptions struct {
	MockVmContext bool
	MockSignature bool
	Quota         uint64
	ContractMeta  *ledger.ContractMeta
	VmLogList     ledger.VmLogList

	KeyValue map[string][]byte
}

func MakeAccounts(num int, chainInstance Chain) map[types.Address]*Account {
	accountMap := make(map[types.Address]*Account, num)

	for i := 0; i < num; i++ {
		addr, privateKey, _ := types.CreateAddress()

		accountMap[addr] = &Account{
			addr:          addr,
			privateKey:    privateKey,
			publicKey:     privateKey.PubByte(),
			chainInstance: chainInstance,

			UnreceivedBlocks:  make(map[types.Hash]*vm_db.VmAccountBlock),
			BlocksMap:         make(map[types.Hash]*vm_db.VmAccountBlock),
			SendBlocksMap:     make(map[types.Hash]*vm_db.VmAccountBlock),
			ReceiveBlocksMap:  make(map[types.Hash]*vm_db.VmAccountBlock),
			ConfirmedBlockMap: make(map[types.Hash]map[types.Hash]struct{}),

			BalanceMap: make(map[types.Hash]*big.Int),
			LogListMap: make(map[types.Hash]ledger.VmLogList),

			KvSetMap:          make(map[types.Hash]map[string][]byte),
			unconfirmedBlocks: make(map[types.Hash]struct{}),
		}
	}
	return accountMap
}

// No state_bak hash
func (acc *Account) CreateRequestTx(toAccount *Account, options *CreateTxOptions) (*vm_db.VmAccountBlock, error) {
	chainInstance := acc.chainInstance
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	prevHash := acc.latestHash()
	vmDb, err := vm_db.NewVmDb(chainInstance, &acc.addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}
	balance, err := vmDb.GetBalance(&ledger.ViteTokenId)
	if err != nil {
		return nil, err
	}

	balance.Add(balance, big.NewInt(189))

	vmDb.SetBalance(&ledger.ViteTokenId, balance)
	if options.ContractMeta != nil {
		vmDb.SetContractMeta(toAccount.addr, options.ContractMeta)
		toAccount.SetContractMeta(options.ContractMeta)
	}
	var logHash *types.Hash
	if len(options.VmLogList) > 0 {
		for _, vmLog := range options.VmLogList {
			vmDb.AddLog(vmLog)
		}
		logHash = vmDb.GetLogListHash()
		acc.LogListMap[*logHash] = vmDb.GetLogList()
	}

	tx := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: acc.addr,
		ToAddress:      toAccount.addr,
		Height:         acc.latestHeight() + 1,
		PrevHash:       prevHash,
		Amount:         big.NewInt(rand.Int63n(100)),
		TokenId:        ledger.ViteTokenId,
		PublicKey:      acc.publicKey,
		Quota:          options.Quota,
		LogHash:        logHash,
	}

	// compute hash
	tx.Hash = tx.ComputeHash()

	if len(options.KeyValue) > 0 {
		acc.KvSetMap[tx.Hash] = make(map[string][]byte)
		for key, value := range options.KeyValue {
			if err := vmDb.SetValue([]byte(key), value); err != nil {
				return nil, err
			}
			acc.KvSetMap[tx.Hash][string(key)] = value
		}
	}
	vmDb.Finish()

	// sign
	if options != nil && options.MockSignature {
		tx.Signature = []byte("This is chain mock signature")
	} else {
		tx.Signature = ed25519.Sign(acc.privateKey, tx.Hash.Bytes())
	}

	acc.latestBlock = tx

	vmTx := &vm_db.VmAccountBlock{
		AccountBlock: tx,
		VmDb:         vmDb,
	}
	toAccount.AddUnreceivedBlock(vmTx)

	acc.BalanceMap[tx.Hash] = balance
	acc.addSendBlock(vmTx)
	return vmTx, nil
}

// No state_bak hash
func (acc *Account) CreateResponseTx(options *CreateTxOptions) (*vm_db.VmAccountBlock, error) {

	UnreceivedBlock := acc.PopUnreceivedBlock()
	if UnreceivedBlock == nil {
		return nil, nil
	}
	chainInstance := acc.chainInstance
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	prevHash := acc.latestHash()
	vmDb, err := vm_db.NewVmDb(acc.chainInstance, &acc.addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}

	balance, err := vmDb.GetBalance(&ledger.ViteTokenId)

	if err != nil {
		return nil, err
	}

	balance.Add(balance, big.NewInt(126))

	vmDb.SetBalance(&ledger.ViteTokenId, balance)
	if UnreceivedBlock.VmDb.GetUnsavedContractMeta() != nil {
		code := crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())))

		vmDb.SetContractCode(code)
		acc.Code = code
	}

	var logHash *types.Hash

	if len(options.VmLogList) > 0 {
		for _, vmLog := range options.VmLogList {
			vmDb.AddLog(vmLog)
		}
		logHash = vmDb.GetLogListHash()
		acc.LogListMap[*logHash] = vmDb.GetLogList()
	}

	receiveTx := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: acc.addr,
		FromBlockHash:  UnreceivedBlock.AccountBlock.Hash,
		Height:         acc.latestHeight() + 1,
		PrevHash:       prevHash,
		PublicKey:      acc.publicKey,

		Quota:   options.Quota,
		LogHash: logHash,
	}

	// compute hash
	receiveTx.Hash = receiveTx.ComputeHash()

	if len(options.KeyValue) > 0 {
		acc.KvSetMap[receiveTx.Hash] = make(map[string][]byte)
		for key, value := range options.KeyValue {
			if err := vmDb.SetValue([]byte(key), value); err != nil {
				return nil, err
			}
			acc.KvSetMap[receiveTx.Hash][string(key)] = value
		}
	}
	vmDb.Finish()

	// sign
	if options != nil && options.MockSignature {
		receiveTx.Signature = []byte("This is chain mock signature")
	} else {
		receiveTx.Signature = ed25519.Sign(acc.privateKey, receiveTx.Hash.Bytes())
	}

	acc.latestBlock = receiveTx
	vmTx := &vm_db.VmAccountBlock{
		AccountBlock: receiveTx,
		VmDb:         vmDb,
	}

	acc.BalanceMap[receiveTx.Hash] = balance
	acc.addReceiveBlock(vmTx)

	return vmTx, nil
}

func (acc *Account) NewKvSetList(KvSetMap map[types.Hash]map[string][]byte) KvSetList {
	kvSetList := make(KvSetList, 0, len(KvSetMap))
	for hash, kv := range KvSetMap {
		kvSetList = append(kvSetList, &KvSet{
			Height: acc.BlocksMap[hash].AccountBlock.Height,
			Kv:     kv,
		})
	}
	sort.Sort(kvSetList)
	return kvSetList
}
func (acc *Account) KeyValue() map[string][]byte {
	kvSetList := acc.NewKvSetList(acc.KvSetMap)

	keyValue := make(map[string][]byte)
	for _, kvSet := range kvSetList {
		for key, value := range kvSet.Kv {
			keyValue[key] = value
		}
	}
	return keyValue
}
func (acc *Account) ContractMeta() *ledger.ContractMeta {
	acc.contractMetaMu.RLock()
	defer acc.contractMetaMu.RUnlock()

	return acc.contractMeta
}

func (acc *Account) SetContractMeta(contractMeta *ledger.ContractMeta) {
	acc.contractMetaMu.Lock()
	defer acc.contractMetaMu.Unlock()

	acc.contractMeta = contractMeta
}

func (acc *Account) HasUnreceivedBlock() bool {
	acc.unreceivedMu.RLock()
	defer acc.unreceivedMu.RUnlock()

	return len(acc.UnreceivedBlocks) > 0
}

func (acc *Account) AddUnreceivedBlock(block *vm_db.VmAccountBlock) {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()

	acc.UnreceivedBlocks[block.AccountBlock.Hash] = block
}

func (acc *Account) PopUnreceivedBlock() *vm_db.VmAccountBlock {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()

	if len(acc.UnreceivedBlocks) <= 0 {
		return nil
	}

	var unreceivedBlock *vm_db.VmAccountBlock
	for _, block := range acc.UnreceivedBlocks {
		unreceivedBlock = block
		break
	}

	return unreceivedBlock
}

func (acc *Account) Snapshot(snapshotHash types.Hash) {
	acc.ConfirmedBlockMap[snapshotHash] = acc.unconfirmedBlocks
	acc.unconfirmedBlocks = make(map[types.Hash]struct{})
}

func (acc *Account) DeleteSnapshotBlocks(accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {
	// rollback unconfirmed blocks
	for hash := range acc.unconfirmedBlocks {
		acc.deleteAccountBlock(accounts, hash)
	}
	acc.unconfirmedBlocks = make(map[types.Hash]struct{})

	// rollback confirmed blocks
	for i := len(snapshotBlocks) - 1; i >= 0; i-- {
		snapshotBlock := snapshotBlocks[i]

		confirmedBlocks := acc.ConfirmedBlockMap[snapshotBlock.Hash]
		if len(confirmedBlocks) <= 0 {
			continue
		}

		for hash := range confirmedBlocks {
			acc.deleteAccountBlock(accounts, hash)
		}

		delete(acc.ConfirmedBlockMap, snapshotBlock.Hash)
	}

	// reset latest block
	acc.resetLatestBlock()
}

func (acc *Account) DeleteOnRoad(blockHash types.Hash) {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()
	delete(acc.UnreceivedBlocks, blockHash)
}

func (acc *Account) DeleteContractMeta() {
	acc.contractMetaMu.Lock()
	defer acc.contractMetaMu.Unlock()

	acc.contractMeta = nil
}

func (acc *Account) deleteAccountBlock(accounts map[types.Address]*Account, blockHash types.Hash) {
	accountBlock := acc.BlocksMap[blockHash].AccountBlock

	if accountBlock.IsSendBlock() {
		toAccount := accounts[accountBlock.ToAddress]
		toAccount.DeleteOnRoad(accountBlock.Hash)

		if accountBlock.BlockType == ledger.BlockTypeSendCreate {
			toAccount.DeleteContractMeta()
		}
	}

	delete(acc.BlocksMap, accountBlock.Hash)
	delete(acc.SendBlocksMap, accountBlock.Hash)
	delete(acc.ReceiveBlocksMap, accountBlock.Hash)
	delete(acc.BlocksMap, accountBlock.Hash)
	delete(acc.KvSetMap, accountBlock.Hash)
	if accountBlock.Height <= 1 {
		acc.Code = nil
	}
	if accountBlock.LogHash != nil {
		delete(acc.LogListMap, *accountBlock.LogHash)
	}
}

func (acc *Account) resetLatestBlock() {
	acc.latestBlock = nil
	var headBlock *ledger.AccountBlock
	for _, block := range acc.BlocksMap {
		if headBlock == nil || block.AccountBlock.Height > headBlock.Height {
			headBlock = block.AccountBlock
		}
	}

	acc.latestBlock = headBlock
}

func (acc *Account) latestHeight() uint64 {
	if acc.latestBlock != nil {
		return acc.latestBlock.Height
	}
	return 0
}

func (acc *Account) latestHash() types.Hash {
	if acc.latestBlock != nil {
		return acc.latestBlock.Hash
	}
	return types.Hash{}
}

func (acc *Account) addSendBlock(block *vm_db.VmAccountBlock) {
	acc.SendBlocksMap[block.AccountBlock.Hash] = block
	acc.BlocksMap[block.AccountBlock.Hash] = block
	acc.unconfirmedBlocks[block.AccountBlock.Hash] = struct{}{}
}
func (acc *Account) addReceiveBlock(block *vm_db.VmAccountBlock) {
	acc.ReceiveBlocksMap[block.AccountBlock.Hash] = block
	acc.BlocksMap[block.AccountBlock.Hash] = block
	acc.unconfirmedBlocks[block.AccountBlock.Hash] = struct{}{}
}
