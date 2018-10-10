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
	wt        *wallet.Manager
	pool      pool.SnapshotProducerWriter
	chain     chain.Chain
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
	trie, err := self.chain.GenStateTrie(head.StateHash, accounts)
	if err != nil {
		return nil, err
	}
	block := &ledger.SnapshotBlock{
		PrevHash:        head.Hash,
		Height:          head.Height + 1,
		Timestamp:       &e.Timestamp,
		StateTrie:       trie,
		StateHash:       *trie.Hash(),
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

func newChainRw(ch chain.Chain, sVerifier *verifier.SnapshotVerifier, wt *wallet.Manager) *tools {
	log := log15.New("module", "tools")
	return &tools{chain: ch, log: log, sVerifier: sVerifier, wt: wt}
}

func (self *tools) checkAddressLock(address types.Address) bool {
	unLocked := self.wt.KeystoreManager.IsUnLocked(address)
	return unLocked
}
func (self *tools) generateAccounts(head *ledger.SnapshotBlock) (ledger.SnapshotContent, error) {

	needSnapshotAccounts := self.chain.GetNeedSnapshotContent()

	// todo get block
	for k, b := range needSnapshotAccounts {
		err := self.sVerifier.VerifyAccountTimeout(k, head.Height+1)
		if err != nil {
			self.pool.RollbackAccountTo(k, b.Hash, b.Height)
		}
	}

	// todo get block
	needSnapshotAccounts = self.chain.GetNeedSnapshotContent()

	var finalAccounts ledger.SnapshotContent

	for k, b := range needSnapshotAccounts {
		err := self.sVerifier.VerifyAccountTimeout(k, head.Height+1)
		if err != nil {
			return nil, errors.New(fmt.Sprintf(
				"error account block, account:%s, blockHash:%s, blockHeight:%d",
				k.String(),
				b.Hash.String(),
				b.Height))
		}
		finalAccounts[k] = &ledger.HashHeight{Hash: b.Hash, Height: b.Height}
	}
	return finalAccounts, nil
}
