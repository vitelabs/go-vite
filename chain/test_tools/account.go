package test_tools

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"sync"
	"time"
)

type Account struct {
	addr       types.Address
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	unreceivedBlocks []*ledger.AccountBlock
	latestBlock      *ledger.AccountBlock

	unreceivedLock sync.Mutex

	chainInstance chain.Chain
}

type CreateTxOptions struct {
	MockVmContext bool
	MockSignature bool
	Quota         uint64
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
	return len(acc.unreceivedBlocks) > 0
}

func (acc *Account) AddUnreceivedBlock(block *ledger.AccountBlock) {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	acc.unreceivedBlocks = append(acc.unreceivedBlocks, block)
}

func (acc *Account) PopUnreceivedBlock() *ledger.AccountBlock {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	if len(acc.unreceivedBlocks) <= 0 {
		return nil
	}

	block := acc.unreceivedBlocks[0]
	acc.unreceivedBlocks = acc.unreceivedBlocks[1:]
	return block
}

// No state hash
func (acc *Account) CreateRequestTx(toAccount *Account, options *CreateTxOptions) []*vm_context.VmAccountBlock {
	now := time.Now()

	var vmContext vmctxt_interface.VmDatabase
	if options != nil && options.MockVmContext {
		vmContext = &MockVmDatabase{}
	} else {
		vmContext, _ = vm_context.NewVmContext(acc.chainInstance, nil, nil, &acc.addr)
	}
	tx := &ledger.AccountBlock{
		AccountAddress: acc.addr,
		ToAddress:      toAccount.addr,
		Height:         acc.Height() + 1,
		PrevHash:       acc.Hash(),
		Amount:         big.NewInt(now.Unix()),
		TokenId:        ledger.ViteTokenId,
		Timestamp:      &now,
		PublicKey:      acc.publicKey,
		Quota:          options.Quota,
	}

	// compute hash
	tx.Hash = tx.ComputeHash()

	// sign
	if options != nil && options.MockSignature {
		tx.Signature = []byte("This is a mock signature")
	} else {
		tx.Signature = ed25519.Sign(acc.privateKey, tx.Hash.Bytes())
	}

	acc.latestBlock = tx

	toAccount.AddUnreceivedBlock(tx)
	return []*vm_context.VmAccountBlock{{
		AccountBlock: tx,
		VmContext:    vmContext,
	}}
}

// No state hash
func (acc *Account) CreateResponseTx(options *CreateTxOptions) []*vm_context.VmAccountBlock {

	UnreceivedBlock := acc.PopUnreceivedBlock()
	if UnreceivedBlock == nil {
		return nil
	}

	now := time.Now()

	var vmContext vmctxt_interface.VmDatabase
	if options != nil && options.MockVmContext {
		vmContext = &MockVmDatabase{}
	} else {
		vmContext, _ = vm_context.NewVmContext(acc.chainInstance, nil, nil, &acc.addr)
	}

	receiveTx := &ledger.AccountBlock{
		AccountAddress: acc.addr,
		FromBlockHash:  UnreceivedBlock.Hash,
		Height:         acc.Height() + 1,
		PrevHash:       acc.Hash(),
		Timestamp:      &now,
		PublicKey:      acc.publicKey,

		Quota: options.Quota,
	}

	// compute hash
	receiveTx.Hash = receiveTx.ComputeHash()

	// sign
	if options != nil && options.MockSignature {
		receiveTx.Signature = []byte("This is a mock signature")
	} else {
		receiveTx.Signature = ed25519.Sign(acc.privateKey, receiveTx.Hash.Bytes())
	}

	acc.latestBlock = receiveTx

	return []*vm_context.VmAccountBlock{{
		AccountBlock: receiveTx,
		VmContext:    vmContext,
	}}
}

func MakeAccounts(num uint64, chainInstance chain.Chain) []*Account {
	accountList := make([]*Account, num)
	for i := uint64(0); i < num; i++ {
		addr, privateKey, _ := types.CreateAddress()

		accountList[i] = &Account{
			addr:          addr,
			privateKey:    privateKey,
			publicKey:     privateKey.PubByte(),
			chainInstance: chainInstance,
		}
	}
	return accountList
}
