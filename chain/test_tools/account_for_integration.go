package test_tools

import (
	rand2 "crypto/rand"

	"github.com/vitelabs/go-vite/chain"

	"github.com/vitelabs/go-vite/common/types"

	"encoding/binary"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"math/rand"
	"sync"
)

type Account struct {
	Addr       types.Address      // encode
	PrivateKey ed25519.PrivateKey // encode
	PublicKey  ed25519.PublicKey  // encode

	chainInstance chain.Chain

	//InitBalance *big.Int // encode

	//contractMetaMu sync.RWMutex
	//ContractMeta   *ledger.ContractMeta // encode

	//Code []byte // encode

	OnRoadBlocks map[types.Hash]*ledger.AccountBlock // encode
	onRoadMu     sync.RWMutex

	//
	//BlocksMap         map[types.Hash]*ledger.AccountBlock    // encode
	//SendBlocksMap     map[types.Hash]*ledger.AccountBlock    // encode
	//ReceiveBlocksMap  map[types.Hash]*ledger.AccountBlock    // encode
	//ConfirmedBlockMap map[types.Hash]map[types.Hash]struct{} // encode
	//UnconfirmedBlocks map[types.Hash]struct{}                // encode
	//
	//BalanceMap map[types.Hash]*big.Int          // encode
	//KvSetMap   map[types.Hash]map[string][]byte // encode
	//
	//LogListMap map[types.Hash]ledger.VmLogList // encode
	//
	LatestBlock *ledger.AccountBlock // encode
}

func NewAccount(chainInstance chain.Chain, pubKey ed25519.PublicKey, privateKey ed25519.PrivateKey) *Account {
	addr := types.PubkeyToAddress(pubKey)

	account := &Account{
		Addr:          addr,
		PrivateKey:    privateKey,
		PublicKey:     pubKey,
		chainInstance: chainInstance,
		//InitBalance:   big.NewInt(10000000),
		//
		OnRoadBlocks: make(map[types.Hash]*ledger.AccountBlock),
		//BlocksMap:         make(map[types.Hash]*ledger.AccountBlock),
		//SendBlocksMap:     make(map[types.Hash]*ledger.AccountBlock),
		//ReceiveBlocksMap:  make(map[types.Hash]*ledger.AccountBlock),
		//ConfirmedBlockMap: make(map[types.Hash]map[types.Hash]struct{}),
		//
		//BalanceMap: make(map[types.Hash]*big.Int),
		//LogListMap: make(map[types.Hash]ledger.VmLogList),
		//
		//KvSetMap:          make(map[types.Hash]map[string][]byte),
		//UnconfirmedBlocks: make(map[types.Hash]struct{}),
	}

	latestBlock, err := chainInstance.GetLatestAccountBlock(addr)

	if err != nil {
		panic(err)
	}

	account.LatestBlock = latestBlock
	return account
}

func MakeAccounts(chainInstance chain.Chain, num int) map[types.Address]*Account {
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

func (acc *Account) CreateSendBlock(toAccount *Account) (*vm_db.VmAccountBlock, error) {
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

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, rand.Uint64())
	if err := vmDb.SetValue([]byte(key), []byte(key)); err != nil {
		return nil, err
	}
	// finish
	vmDb.Finish()

	// new block
	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: acc.Addr,
		ToAddress:      toAccount.Addr,
		Amount:         big.NewInt(2),
		Height:         acc.GetLatestHeight() + 1,
		PrevHash:       prevHash,
		TokenId:        ledger.ViteTokenId,
		PublicKey:      acc.PublicKey,
	}

	// compute hash
	block.Hash = block.ComputeHash()

	// sign
	block.Signature = []byte("This is chain mock signature")

	//block.Signature = ed25519.Sign(acc.PrivateKey, block.Hash.Bytes())

	vmBlock := &vm_db.VmAccountBlock{
		AccountBlock: block,
		VmDb:         vmDb,
	}

	return vmBlock, nil
}

// No state_bak hash
func (acc *Account) CreateReceiveBlock() (*vm_db.VmAccountBlock, error) {
	latestHeight := acc.GetLatestHeight()

	// pop onRoad block
	blockType := ledger.BlockTypeReceive
	var fromBlockHash types.Hash
	if latestHeight >= 1 {
		UnreceivedBlock := acc.PopOnRoadBlock()
		if UnreceivedBlock == nil {
			return nil, nil
		}

		fromBlockHash = UnreceivedBlock.Hash
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
	key := make([]byte, 8)
	if err := vmDb.SetValue([]byte(key), []byte(key)); err != nil {
		return nil, err
	}

	if latestHeight < 1 {
		vmDb.SetBalance(&ledger.ViteTokenId, big.NewInt(1000*1000*1000))
	}

	// vm db finish
	vmDb.Finish()

	// new block
	block := &ledger.AccountBlock{
		BlockType:      blockType,
		AccountAddress: acc.Addr,
		FromBlockHash:  fromBlockHash,
		Height:         latestHeight + 1,
		PrevHash:       prevHash,
		PublicKey:      acc.PublicKey,
	}

	// compute hash
	block.Hash = block.ComputeHash()

	// sign

	block.Signature = []byte("This is chain mock signature")

	//block.Signature = ed25519.Sign(acc.PrivateKey, block.Hash.Bytes())

	vmBlock := &vm_db.VmAccountBlock{
		AccountBlock: block,
		VmDb:         vmDb,
	}

	return vmBlock, nil
}

func (acc *Account) InsertBlock(vmBlock *vm_db.VmAccountBlock, accounts map[types.Address]*Account) {
	block := vmBlock.AccountBlock

	// set latest block
	acc.LatestBlock = block

	// set send block
	if block.IsSendBlock() {
		// set on road
		toAccount := accounts[block.ToAddress]
		toAccount.AddOnRoadBlock(block)

	}

	// set send block list
	for _, sendBlock := range block.SendBlockList {
		toAccount := accounts[sendBlock.ToAddress]
		toAccount.AddOnRoadBlock(sendBlock)
	}

}

func (acc *Account) HasOnRoadBlock() bool {
	acc.onRoadMu.RLock()
	defer acc.onRoadMu.RUnlock()

	return len(acc.OnRoadBlocks) > 0
}

func (acc *Account) AddOnRoadBlock(block *ledger.AccountBlock) {
	acc.onRoadMu.Lock()
	defer acc.onRoadMu.Unlock()

	//fmt.Println("Mock AddOnRoadBlock", acc.Addr, block.Hash)
	acc.OnRoadBlocks[block.Hash] = block
}

func (acc *Account) DeleteOnRoad(blockHash types.Hash) {
	acc.onRoadMu.Lock()
	defer acc.onRoadMu.Unlock()

	//fmt.Println("Mock DeleteOnRoad", acc.Addr, blockHash)
	delete(acc.OnRoadBlocks, blockHash)
}

func (acc *Account) PopOnRoadBlock() *ledger.AccountBlock {
	acc.onRoadMu.Lock()
	defer acc.onRoadMu.Unlock()

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
