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
	"sync"
	"testing"
	"time"
)

func TestChain_Account(t *testing.T) {

	chainInstance, _, _, addrList, _, _ := SetUp(t, 1000, 1000, 8)

	testAccount(t, chainInstance, addrList)
	TearDown(chainInstance)
}

func testAccount(t *testing.T, chainInstance *chain, addrList []types.Address) {

	accountIdList := make([]uint64, len(addrList))

	for index, addr := range addrList {
		accountId, err := chainInstance.GetAccountId(addr)
		if err != nil {
			t.Fatal(err)
		}
		if accountId <= 0 {
			t.Fatal("accountId <= 0")
		}

		accountIdList[index] = accountId
	}

	t.Run("GetAccountId", func(t *testing.T) {
		GetAccountId(t, chainInstance, addrList, accountIdList)
	})

	t.Run("GetAccountAddress", func(t *testing.T) {
		GetAccountAddress(t, chainInstance, addrList, accountIdList)
	})
}

func MakeAccounts(num int, chainInstance Chain) (map[types.Address]*Account, []types.Address) {
	accountMap := make(map[types.Address]*Account, num)
	addrList := make([]types.Address, 0, num)

	for i := 0; i < num; i++ {
		addr, privateKey, _ := types.CreateAddress()

		accountMap[addr] = &Account{
			addr:          addr,
			privateKey:    privateKey,
			publicKey:     privateKey.PubByte(),
			chainInstance: chainInstance,

			SendBlocksMap:     make(map[types.Hash]*vm_db.VmAccountBlock),
			ReceiveBlocksMap:  make(map[types.Hash]*vm_db.VmAccountBlock),
			BalanceMap:        make(map[types.Hash]*big.Int),
			ConfirmedBlockMap: make(map[types.Hash]map[types.Hash]struct{}),
			LogListMap:        make(map[types.Hash]ledger.VmLogList),
			KeyValue:          make(map[string][]byte),

			unconfirmedBlocks: make(map[types.Hash]struct{}),
		}
		addrList = append(addrList, addr)
	}
	return accountMap, addrList
}

func GetAccountId(t *testing.T, chainInstance *chain, addrList []types.Address, accountIdList []uint64) {
	for index, addr := range addrList {
		accountId, err := chainInstance.GetAccountId(addr)
		if err != nil {
			t.Fatal(err)
		}
		if accountIdList[index] != accountId {
			t.Fatal("error")
		}
	}
}
func GetAccountAddress(t *testing.T, chainInstance *chain, addrList []types.Address, accountIdList []uint64) {
	for index, accountId := range accountIdList {
		addr, err := chainInstance.GetAccountAddress(accountId)
		if err != nil {
			t.Fatal(err)
		}
		if addrList[index] != *addr {
			t.Fatal("error")
		}
	}
}

type Account struct {
	addr       types.Address
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	UnreceivedBlocks []*vm_db.VmAccountBlock

	SendBlocksMap     map[types.Hash]*vm_db.VmAccountBlock
	ReceiveBlocksMap  map[types.Hash]*vm_db.VmAccountBlock
	ConfirmedBlockMap map[types.Hash]map[types.Hash]struct{}
	BalanceMap        map[types.Hash]*big.Int

	ContractMeta *ledger.ContractMeta
	Code         []byte
	LogListMap   map[types.Hash]ledger.VmLogList
	KeyValue     map[string][]byte

	unconfirmedBlocks map[types.Hash]struct{}

	latestBlock *ledger.AccountBlock

	unreceivedLock sync.Mutex

	chainInstance Chain
}

type CreateTxOptions struct {
	MockVmContext bool
	MockSignature bool
	Quota         uint64
	ContractMeta  *ledger.ContractMeta
	VmLogList     ledger.VmLogList

	KeyValue map[string][]byte
}

func (acc *Account) Height() uint64 {
	if acc.latestBlock != nil {
		return acc.latestBlock.Height
	}
	return 0
}

func (acc *Account) Hash() types.Hash {
	if acc.latestBlock != nil {
		return acc.latestBlock.Hash
	}
	return types.Hash{}
}

func (acc *Account) HasUnreceivedBlock() bool {
	return len(acc.UnreceivedBlocks) > 0
}

func (acc *Account) AddUnreceivedBlock(block *vm_db.VmAccountBlock) {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	acc.UnreceivedBlocks = append(acc.UnreceivedBlocks, block)
}

func (acc *Account) Snapshot(snapshotHash types.Hash) {
	acc.ConfirmedBlockMap[snapshotHash] = acc.unconfirmedBlocks
	acc.unconfirmedBlocks = make(map[types.Hash]struct{})
}

func (acc *Account) PopUnreceivedBlock() *vm_db.VmAccountBlock {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	if len(acc.UnreceivedBlocks) <= 0 {
		return nil
	}

	block := acc.UnreceivedBlocks[0]
	acc.UnreceivedBlocks = acc.UnreceivedBlocks[1:]
	return block
}

// No state_bak hash
func (acc *Account) CreateRequestTx(toAccount *Account, options *CreateTxOptions) (*vm_db.VmAccountBlock, error) {
	chainInstance := acc.chainInstance
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	prevHash := acc.Hash()
	vmDb, err := vm_db.NewVmDb(chainInstance, &acc.addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}
	balance, err := vmDb.GetBalance(&ledger.ViteTokenId)
	if err != nil {
		return nil, err
	}
	vmDb.SetBalance(&ledger.ViteTokenId, balance.Abs(balance.Sub(balance, big.NewInt(100))))
	if options.ContractMeta != nil {
		vmDb.SetContractMeta(toAccount.addr, options.ContractMeta)
		toAccount.ContractMeta = options.ContractMeta
	}
	var logHash *types.Hash
	if len(options.VmLogList) > 0 {
		for _, vmLog := range options.VmLogList {
			vmDb.AddLog(vmLog)
		}
		logHash = vmDb.GetLogListHash()
		acc.LogListMap[*logHash] = vmDb.GetLogList()
	}
	if len(options.KeyValue) > 0 {
		for key, value := range options.KeyValue {
			if err := vmDb.SetValue([]byte(key), value); err != nil {
				return nil, err
			}
			acc.KeyValue[string(key)] = value

		}
	}
	vmDb.Finish()

	tx := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: acc.addr,
		ToAddress:      toAccount.addr,
		Height:         acc.Height() + 1,
		PrevHash:       prevHash,
		Amount:         big.NewInt(rand.Int63n(100)),
		TokenId:        ledger.ViteTokenId,
		PublicKey:      acc.publicKey,
		Quota:          options.Quota,
		LogHash:        logHash,
	}

	// compute hash
	tx.Hash = tx.ComputeHash()

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

	prevHash := acc.Hash()
	vmDb, err := vm_db.NewVmDb(acc.chainInstance, &acc.addr, &latestSnapshotBlock.Hash, &prevHash)
	if err != nil {
		return nil, err
	}

	balance, err := vmDb.GetBalance(&ledger.ViteTokenId)
	if err != nil {
		return nil, err
	}

	vmDb.SetBalance(&ledger.ViteTokenId, balance.Abs(balance.Add(balance, big.NewInt(100))))
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

	if len(options.KeyValue) > 0 {
		for key, value := range options.KeyValue {
			vmDb.SetValue([]byte(key), value)
			acc.KeyValue[string(key)] = value
		}
	}

	vmDb.Finish()

	receiveTx := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: acc.addr,
		FromBlockHash:  UnreceivedBlock.AccountBlock.Hash,
		Height:         acc.Height() + 1,
		PrevHash:       prevHash,
		PublicKey:      acc.publicKey,

		Quota:   options.Quota,
		LogHash: logHash,
	}

	// compute hash
	receiveTx.Hash = receiveTx.ComputeHash()

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

func (acc *Account) addSendBlock(block *vm_db.VmAccountBlock) {
	acc.SendBlocksMap[block.AccountBlock.Hash] = block
	acc.unconfirmedBlocks[block.AccountBlock.Hash] = struct{}{}
}
func (acc *Account) addReceiveBlock(block *vm_db.VmAccountBlock) {
	acc.ReceiveBlocksMap[block.AccountBlock.Hash] = block
	acc.unconfirmedBlocks[block.AccountBlock.Hash] = struct{}{}
}
