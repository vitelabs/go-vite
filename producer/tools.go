package producer

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

type tools struct {
	log       log15.Logger
	wt        wallet.Manager
	pool      pool.SnapshotProducerWriter
	chain     *chain.Chain
	sVerifier *verifier.SnapshotVerifier
}

func (self *tools) ledgerLock() {
	self.pool.Lock()
}
func (self *tools) ledgerUnLock() {
	self.pool.UnLock()
}

func (self *tools) generateSnapshot(e *consensus.Event) (*ledger.SnapshotBlock, error) {
	head := self.chain.GetLatestSnapshotBlock()
	accounts, err := self.generateAccounts(head)

	if err != nil {
		return nil, err
	}
	block := &ledger.SnapshotBlock{
		PrevHash:  head.Hash,
		Height:    head.Height + 1,
		Timestamp: &e.Timestamp,
		//SnapshotHash:    &accounts.Hash(), todo add hash support
		SnapshotContent: accounts,
	}

	block.Hash = block.ComputeHash()
	signedData, pubkey, err := self.wt.KeystoreManager.SignData(e.Address, block.Hash.Bytes())

	if err != nil {
		return nil, err
	}
	block.Signature = signedData
	block.PublicKey = pubkey
	return block, nil
}
func (self *tools) insertSnapshot(block *ledger.SnapshotBlock) error {
	// todo insert pool ?? dead lock
	return nil
}

func newChainRw(ch *chain.Chain) *tools {
	log := log15.New("module", "tools")
	return &tools{chain: ch, log: log}
}

func (self *tools) checkAddressLock(address types.Address) bool {
	unLocked := self.wt.KeystoreManager.IsUnLocked(address)
	return unLocked
}
func (self *tools) generateAccounts(head *ledger.SnapshotBlock) (ledger.SnapshotContent, error) {
	var needSnapshotAccounts []*ledger.AccountBlock

	// todo get block
	for _, b := range needSnapshotAccounts {
		err := self.sVerifier.VerifyAccountTimeout(b.AccountAddress, head.Height+1)
		if err != nil {
			self.pool.RollbackAccountTo(b.AccountAddress, b.Hash, b.Height)
		}
	}

	// todo get block
	needSnapshotAccounts = nil

	var finalAccounts ledger.SnapshotContent

	for _, b := range needSnapshotAccounts {
		err := self.sVerifier.VerifyAccountTimeout(b.AccountAddress, head.Height+1)
		if err != nil {
			return nil, errors.New(fmt.Sprintf(
				"error account block, account:%s, blockHash:%s, blockHeight:%d",
				b.AccountAddress.String(),
				b.Hash.String(),
				b.Height))
		}
		finalAccounts[b.AccountAddress] = &ledger.SnapshotContentItem{AccountBlockHash: b.Hash, AccountBlockHeight: b.Height}
	}
	return finalAccounts, nil
}
