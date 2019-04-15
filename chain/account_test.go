package chain

import (
	rand2 "crypto/rand"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"sort"
	"sync"

	"fmt"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/crypto"
	"testing"
	"time"
)

func TestChain_Account(t *testing.T) {

	chainInstance, accounts, _ := SetUp(t, 1000, 1000, 8)
	t.Run("testAccount", func(t *testing.T) {
		testAccount(chainInstance, accounts)
	})
	TearDown(chainInstance)
}
func testAccount(chainInstance *chain, accounts map[types.Address]*Account) {

	accountIdList := make([]uint64, 0, len(accounts))

	for addr, account := range accounts {
		accountId, err := chainInstance.GetAccountId(addr)
		if err != nil {
			panic(err)
		}
		if accountId <= 0 {
			if account.LatestBlock == nil {
				continue
			}
			panic(fmt.Sprintf("accountId <= 0, %s, %+v\n", addr, account.LatestBlock))
		}
		queryAddr2, err := chainInstance.GetAccountAddress(accountId)
		if err != nil {
			panic(err)
		}

		if *queryAddr2 != addr {
			panic("error")
		}

		accountIdList = append(accountIdList, accountId)
	}

	for _, accountId := range accountIdList {
		addr, err := chainInstance.GetAccountAddress(accountId)
		if err != nil {
			panic(err)
		}
		if addr == nil {
			panic("Addr is nil")
		}
		queryAccountId, err := chainInstance.GetAccountId(*addr)
		if err != nil {
			panic(err)
		}

		if queryAccountId != accountId {
			panic("error")
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
	Addr       types.Address      // encode
	PrivateKey ed25519.PrivateKey // encode
	PublicKey  ed25519.PublicKey  // encode

	chainInstance Chain

	InitBalance *big.Int // encode

	contractMetaMu sync.RWMutex
	ContractMeta   *ledger.ContractMeta // encode

	Code []byte // encode

	OnRoadBlocks map[types.Hash]*ledger.AccountBlock // encode
	unreceivedMu sync.RWMutex

	BlocksMap         map[types.Hash]*ledger.AccountBlock    // encode
	SendBlocksMap     map[types.Hash]*ledger.AccountBlock    // encode
	ReceiveBlocksMap  map[types.Hash]*ledger.AccountBlock    // encode
	ConfirmedBlockMap map[types.Hash]map[types.Hash]struct{} // encode
	UnconfirmedBlocks map[types.Hash]struct{}                // encode

	BalanceMap map[types.Hash]*big.Int          // encode
	KvSetMap   map[types.Hash]map[string][]byte // encode

	LogListMap map[types.Hash]ledger.VmLogList // encode

	LatestBlock *ledger.AccountBlock // encode
}

type CreateTxOptions struct {
	MockVmContext bool
	MockSignature bool
	Quota         uint64
	ContractMeta  *ledger.ContractMeta
	VmLogList     ledger.VmLogList

	KeyValue map[string][]byte
}

func NewAccount(chainInstance Chain, pubKey ed25519.PublicKey, privateKey ed25519.PrivateKey) *Account {
	addr := types.PubkeyToAddress(pubKey)

	account := &Account{
		Addr:          addr,
		PrivateKey:    privateKey,
		PublicKey:     pubKey,
		chainInstance: chainInstance,
		InitBalance:   big.NewInt(10000000),

		OnRoadBlocks:      make(map[types.Hash]*ledger.AccountBlock),
		BlocksMap:         make(map[types.Hash]*ledger.AccountBlock),
		SendBlocksMap:     make(map[types.Hash]*ledger.AccountBlock),
		ReceiveBlocksMap:  make(map[types.Hash]*ledger.AccountBlock),
		ConfirmedBlockMap: make(map[types.Hash]map[types.Hash]struct{}),

		BalanceMap: make(map[types.Hash]*big.Int),
		LogListMap: make(map[types.Hash]ledger.VmLogList),

		KvSetMap:          make(map[types.Hash]map[string][]byte),
		UnconfirmedBlocks: make(map[types.Hash]struct{}),
	}

	latestBlock, err := chainInstance.GetLatestAccountBlock(addr)

	if err != nil {
		panic(err)
	}

	account.LatestBlock = latestBlock
	return account
}

func MakeAccounts(chainInstance Chain, num int) map[types.Address]*Account {
	accountMap := make(map[types.Address]*Account, num)

	for i := 0; i < num; i++ {
		pub, pri, err := ed25519.GenerateKey(rand2.Reader)
		if err != nil {
			panic(err)
		}
		addr := types.PubkeyToAddress(pub)

		accountMap[addr] = NewAccount(chainInstance, pub, pri)
	}
	return accountMap
}

// No state_bak hash
func (acc *Account) CreateSendBlock(toAccount *Account, options *CreateTxOptions) (*vm_db.VmAccountBlock, error) {
	// get latest hash
	prevHash := acc.latestHash()

	// get latest snapshot block
	chainInstance := acc.chainInstance
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	// new vm_db
	vmDb, err := vm_db.NewVmDb(chainInstance, &acc.Addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}

	// set balance
	//amount := big.NewInt(rand.Int63n(150))
	//vmDb.SetBalance(&ledger.ViteTokenId, new(big.Int).Sub(acc.Balance(), amount))
	amount := big.NewInt(1)
	vmDb.SetBalance(&ledger.ViteTokenId, big.NewInt(int64(acc.GetLatestHeight()+1)))

	// set contract meta
	if options.ContractMeta != nil {
		vmDb.SetContractMeta(toAccount.Addr, options.ContractMeta)
	}

	// set log list
	var logHash *types.Hash
	if len(options.VmLogList) > 0 {
		for _, vmLog := range options.VmLogList {
			vmDb.AddLog(vmLog)
		}
		logHash = vmDb.GetLogListHash()
	}

	// set key value
	for key, value := range options.KeyValue {
		if err := vmDb.SetValue([]byte(key), value); err != nil {
			return nil, err
		}
	}

	// finish
	vmDb.Finish()

	// new block
	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: acc.Addr,
		ToAddress:      toAccount.Addr,
		Height:         acc.GetLatestHeight() + 1,
		PrevHash:       prevHash,
		Amount:         amount,
		TokenId:        ledger.ViteTokenId,
		PublicKey:      acc.PublicKey,
		Quota:          options.Quota,
		LogHash:        logHash,
	}

	// compute hash
	block.Hash = block.ComputeHash()

	// sign
	if options != nil && options.MockSignature {
		block.Signature = []byte("This is chain mock signature")
	} else {
		block.Signature = ed25519.Sign(acc.PrivateKey, block.Hash.Bytes())
	}

	vmBlock := &vm_db.VmAccountBlock{
		AccountBlock: block,
		VmDb:         vmDb,
	}

	return vmBlock, nil
}

// No state_bak hash
func (acc *Account) CreateReceiveBlock(options *CreateTxOptions) (*vm_db.VmAccountBlock, error) {
	// pop onRoad block
	UnreceivedBlock := acc.PopOnRoadBlock()
	if UnreceivedBlock == nil {
		return nil, nil
	}

	// prevHash
	prevHash := acc.latestHash()

	// latest block
	chainInstance := acc.chainInstance

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	// new vm db
	vmDb, err := vm_db.NewVmDb(acc.chainInstance, &acc.Addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}

	// set balance
	vmDb.SetBalance(&ledger.ViteTokenId, big.NewInt(int64(acc.GetLatestHeight()+1)))
	//vmDb.SetBalance(&ledger.ViteTokenId, new(big.Int).Add(acc.Balance(), UnreceivedBlock.Amount))

	// set contract code
	if acc.GetContractMeta() != nil && acc.LatestBlock == nil {
		code := crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano())))

		vmDb.SetContractCode(code)
	}

	// set log list
	var logHash *types.Hash
	if len(options.VmLogList) > 0 {
		for _, vmLog := range options.VmLogList {
			vmDb.AddLog(vmLog)
		}
		logHash = vmDb.GetLogListHash()
	}

	// set key value
	for key, value := range options.KeyValue {
		if err := vmDb.SetValue([]byte(key), value); err != nil {
			return nil, err
		}
	}

	// vm db finish
	vmDb.Finish()

	// new block
	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: acc.Addr,
		FromBlockHash:  UnreceivedBlock.Hash,
		Height:         acc.GetLatestHeight() + 1,
		PrevHash:       prevHash,
		PublicKey:      acc.PublicKey,

		Quota:   options.Quota,
		LogHash: logHash,
	}

	// compute hash
	block.Hash = block.ComputeHash()

	// sign
	if options != nil && options.MockSignature {
		block.Signature = []byte("This is chain mock signature")
	} else {
		block.Signature = ed25519.Sign(acc.PrivateKey, block.Hash.Bytes())
	}

	vmBlock := &vm_db.VmAccountBlock{
		AccountBlock: block,
		VmDb:         vmDb,
	}

	return vmBlock, nil
}

func (acc *Account) InsertBlock(vmBlock *vm_db.VmAccountBlock, accounts map[types.Address]*Account) {
	block := vmBlock.AccountBlock
	vmDb := vmBlock.VmDb

	// set latest block
	acc.LatestBlock = block

	// set key value
	acc.KvSetMap[block.Hash] = make(map[string][]byte)
	for _, kv := range vmDb.GetUnsavedStorage() {
		acc.KvSetMap[block.Hash][string(kv[0])] = kv[1]
	}

	// set balance
	acc.BalanceMap[block.Hash] = vmDb.GetUnsavedBalanceMap()[ledger.ViteTokenId]

	// set contract meta
	if contractMetaMap := vmDb.GetUnsavedContractMeta(); len(contractMetaMap) > 0 {
		for addr, contractMeta := range contractMetaMap {
			toAccount := accounts[addr]
			toAccount.SetContractMeta(contractMeta)
		}
	}

	// set code
	if code := vmDb.GetUnsavedContractCode(); len(code) > 0 {
		acc.Code = code
	}

	// set log
	if vmLogList := vmDb.GetLogList(); len(vmLogList) > 0 {
		acc.LogListMap[*vmDb.GetLogListHash()] = vmLogList
	}

	// set send block
	if block.IsSendBlock() {
		// set send block
		acc.addSendBlock(block)

		// set on road
		toAccount := accounts[block.ToAddress]
		toAccount.AddOnRoadBlock(block)

	} else {
		// add receive block
		acc.addReceiveBlock(block)
	}

	// set send block list
	for _, sendBlock := range block.SendBlockList {
		toAccount := accounts[sendBlock.ToAddress]
		toAccount.AddOnRoadBlock(sendBlock)
	}

}

func (acc *Account) NewKvSetList(KvSetMap map[types.Hash]map[string][]byte) KvSetList {
	kvSetList := make(KvSetList, 0, len(KvSetMap))
	for hash, kv := range KvSetMap {
		kvSetList = append(kvSetList, &KvSet{
			Height: acc.BlocksMap[hash].Height,
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
func (acc *Account) GetContractMeta() *ledger.ContractMeta {
	acc.contractMetaMu.RLock()
	defer acc.contractMetaMu.RUnlock()

	return acc.ContractMeta
}

func (acc *Account) SetContractMeta(contractMeta *ledger.ContractMeta) {
	acc.contractMetaMu.Lock()
	defer acc.contractMetaMu.Unlock()

	acc.ContractMeta = contractMeta
}

func (acc *Account) HasOnRoadBlock() bool {
	acc.unreceivedMu.RLock()
	defer acc.unreceivedMu.RUnlock()

	return len(acc.OnRoadBlocks) > 0
}

func (acc *Account) AddOnRoadBlock(block *ledger.AccountBlock) {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()

	//fmt.Println("Mock AddOnRoadBlock", acc.Addr, block.Hash)
	acc.OnRoadBlocks[block.Hash] = block
}

func (acc *Account) DeleteOnRoad(blockHash types.Hash) {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()

	//fmt.Println("Mock DeleteOnRoad", acc.Addr, blockHash)
	delete(acc.OnRoadBlocks, blockHash)
}

func (acc *Account) PopOnRoadBlock() *ledger.AccountBlock {
	acc.unreceivedMu.Lock()
	defer acc.unreceivedMu.Unlock()

	if len(acc.OnRoadBlocks) <= 0 {
		return nil
	}

	var unreceivedBlock *ledger.AccountBlock
	for _, block := range acc.OnRoadBlocks {
		unreceivedBlock = block
		break
	}
	//fmt.Println("Mock PopOnRoadBlock", acc.Addr, unreceivedBlock.Hash)
	delete(acc.OnRoadBlocks, unreceivedBlock.Hash)

	return unreceivedBlock
}

func (acc *Account) Snapshot(snapshotHash types.Hash, hashHeight *ledger.HashHeight) {
	// confirmed
	confirmedBlocks := make(map[types.Hash]struct{}, len(acc.UnconfirmedBlocks))

	// unconfirmed
	unconfirmedBlocks := make(map[types.Hash]struct{})

	deletedCache := make([]uint64, 0)

	for hash := range acc.UnconfirmedBlocks {
		block := acc.BlocksMap[hash]

		if block.Height <= hashHeight.Height {
			confirmedBlocks[hash] = struct{}{}
			deletedCache = append(deletedCache, block.Height)
		} else {
			unconfirmedBlocks[hash] = struct{}{}
		}
	}

	// check
	if len(deletedCache) <= 0 {
		panic(fmt.Sprintf("%s. %d %s", snapshotHash, hashHeight.Height, hashHeight.Hash))
	}
	minHeight := helper.MaxUint64
	maxHeight := uint64(0)
	for _, height := range deletedCache {
		if height < minHeight {
			minHeight = height
		}

		if height >= maxHeight {
			maxHeight = height
		}
	}
	if uint64(len(deletedCache)) != (maxHeight-minHeight)+1 {
		panic("error")
	}
	if maxHeight != hashHeight.Height {
		panic("error")
	}

	// set confirmed
	acc.ConfirmedBlockMap[snapshotHash] = confirmedBlocks

	// set unconfirmed
	acc.UnconfirmedBlocks = unconfirmedBlocks
}

func (acc *Account) DeleteSnapshotBlocks(accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock, hasRedoLog bool) {
	// rollback unconfirmed blocks
	unconfirmedBlockHashList := make([]types.Hash, 0, len(acc.UnconfirmedBlocks))
	for hash := range acc.UnconfirmedBlocks {
		unconfirmedBlockHashList = append(unconfirmedBlockHashList, hash)
	}
	for _, hash := range unconfirmedBlockHashList {
		//fmt.Printf("Mock delete unconfirmedBlockHashList %s %s\n", acc.Addr, hash)
		acc.deleteAccountBlock(accounts, hash)
	}

	// rollback confirmed blocks, from high to low
	for i := 0; i < len(snapshotBlocks); i++ {
		snapshotBlock := snapshotBlocks[i]

		confirmedBlocks := acc.ConfirmedBlockMap[snapshotBlock.Hash]
		if i == len(snapshotBlocks)-1 && hasRedoLog {
			acc.UnconfirmedBlocks = confirmedBlocks

		} else {
			if len(confirmedBlocks) <= 0 {
				continue
			}

			for hash := range confirmedBlocks {
				acc.deleteAccountBlock(accounts, hash)
			}
		}
		delete(acc.ConfirmedBlockMap, snapshotBlock.Hash)

	}

	// reset latest block
	acc.resetLatestBlock()

}

func (acc *Account) DeleteContractMeta() {
	acc.contractMetaMu.Lock()
	defer acc.contractMetaMu.Unlock()

	acc.ContractMeta = nil
}

func (acc *Account) GetInitBalance() *big.Int {
	return acc.InitBalance
}

func (acc *Account) Balance() *big.Int {
	if acc.LatestBlock == nil {
		return acc.InitBalance
	}

	return acc.BalanceMap[acc.LatestBlock.Hash]
}

func (acc *Account) deleteAccountBlock(accounts map[types.Address]*Account, blockHash types.Hash) {
	accountBlock := acc.BlocksMap[blockHash]

	if accountBlock.IsSendBlock() {
		toAccount := accounts[accountBlock.ToAddress]
		toAccount.DeleteOnRoad(accountBlock.Hash)

		if accountBlock.BlockType == ledger.BlockTypeSendCreate {
			toAccount.DeleteContractMeta()
		}
	} else {
		for _, account := range accounts {
			if fromBlock, ok := account.BlocksMap[accountBlock.FromBlockHash]; ok && fromBlock != nil {
				acc.AddOnRoadBlock(fromBlock)
				break
			}
		}
	}

	delete(acc.UnconfirmedBlocks, accountBlock.Hash)
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

	for _, sendBlock := range accountBlock.SendBlockList {
		acc.deleteAccountBlock(accounts, sendBlock.Hash)
	}
}

func (acc *Account) rollbackLatestBlock() {
	if acc.LatestBlock == nil {
		return
	}
	if acc.LatestBlock.Height <= 1 {
		acc.LatestBlock = nil
	} else {
		acc.LatestBlock = acc.BlocksMap[acc.LatestBlock.PrevHash]

	}
}

func (acc *Account) resetLatestBlock() {
	acc.LatestBlock = nil
	var headBlock *ledger.AccountBlock
	for _, block := range acc.BlocksMap {
		if headBlock == nil || block.Height > headBlock.Height {
			headBlock = block
		}
	}

	acc.LatestBlock = headBlock
}

func (acc *Account) GetLatestHeight() uint64 {
	if acc.LatestBlock != nil {
		return acc.LatestBlock.Height
	}
	return 0
}

func (acc *Account) latestHash() types.Hash {
	if acc.LatestBlock != nil {
		return acc.LatestBlock.Hash
	}
	return types.Hash{}
}

func (acc *Account) addSendBlock(block *ledger.AccountBlock) {
	if acc.SendBlocksMap == nil {
		acc.SendBlocksMap = make(map[types.Hash]*ledger.AccountBlock)
	}
	acc.SendBlocksMap[block.Hash] = block

	if acc.BlocksMap == nil {
		acc.BlocksMap = make(map[types.Hash]*ledger.AccountBlock)
	}
	acc.BlocksMap[block.Hash] = block

	if acc.UnconfirmedBlocks == nil {
		acc.UnconfirmedBlocks = make(map[types.Hash]struct{})
	}
	acc.UnconfirmedBlocks[block.Hash] = struct{}{}
}
func (acc *Account) addReceiveBlock(block *ledger.AccountBlock) {
	acc.ReceiveBlocksMap[block.Hash] = block

	if acc.BlocksMap == nil {
		acc.BlocksMap = make(map[types.Hash]*ledger.AccountBlock)
	}
	acc.BlocksMap[block.Hash] = block

	if acc.UnconfirmedBlocks == nil {
		acc.UnconfirmedBlocks = make(map[types.Hash]struct{})
	}
	acc.UnconfirmedBlocks[block.Hash] = struct{}{}
}
