package chain_benchmark

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sync"
	"time"
)

type account struct {
	Addr       types.Address
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey

	LatestBlock *ledger.AccountBlock

	UnreceivedBlocks  []*ledger.AccountBlock
	unreceivedLock sync.Mutex
}

func (acc *account) Height() uint64 {
	if acc.LatestBlock != nil {
		return acc.LatestBlock.Height
	}
	return 0
}

func (acc *account) Hash() types.Hash {
	if acc.LatestBlock != nil {
		return acc.LatestBlock.Hash
	}
	return types.Hash{}
}

func (acc *account) AddUnreceivedBlock(block *ledger.AccountBlock) {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	acc.UnreceivedBlocks = append(acc.UnreceivedBlocks, block)
}

func (acc *account) PopUnreceivedBlock() *ledger.AccountBlock {
	acc.unreceivedLock.Lock()
	defer acc.unreceivedLock.Unlock()

	if len(acc.UnreceivedBlocks) <= 0 {
		return nil
	}

	block = acc.UnreceivedBlocks[0]

}



func makeAccounts(num uint64) []account {
	accountList := make([]account, num)
	for i := uint64(0); i < num; i++ {
		addr, privateKey, _ := types.CreateAddress()

		accountList[i] = account{
			Addr:       addr,
			PrivateKey: privateKey,
			PublicKey:  privateKey.PubByte(),
		}
	}
	return accountList
}

// No state hash
func createSendTx(fromAccount *account, toAccount *account) *ledger.AccountBlock {
	now := time.Now()
	tx := &ledger.AccountBlock{
		AccountAddress: fromAccount.Addr,
		ToAddress:      toAccount.Addr,
		Height:         fromAccount.Height() + 1,
		PrevHash:       fromAccount.Hash(),
		Amount:         big.NewInt(now.Unix()),
		TokenId:        ledger.ViteTokenId,
		Timestamp:      &now,
		PublicKey:      fromAccount.PublicKey,
	}

	// compute hash
	tx.Hash = tx.ComputeHash()
	// sign
	tx.Signature = ed25519.Sign(fromAccount.PrivateKey, tx.Hash.Bytes())

	fromAccount.LatestBlock = tx
	toAccount.AddUnreceivedBlock(tx)
	return tx
}

func createReceiveTx(acc *account) *ledger.AccountBlock  {
	if len(acc.UnreceivedBlocks) <= 0 {
		return nil
	}


	receiveTx := &ledger.AccountBlock{
		AccountAddress: acc.Addr,
		FromBlockHash:
	}
}