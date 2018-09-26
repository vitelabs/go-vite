package producer

import (
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
	pool      pool.PoolWriter
	chain     *chain.Chain
	sVerifier *verifier.SnapshotVerifier
}

func (self *tools) ledgerLock() {
	self.pool.Lock()
}
func (self *tools) ledgerUnLock() {
	self.pool.UnLock()
}

//Hash     types.Hash
//PrevHash types.Hash
//Height   uint64
//producer *types.Address
//
//PublicKey ed25519.PublicKey
//Signature []byte
//
//Timestamp *time.Time
//
//SnapshotHash    *types.Hash
//SnapshotContent SnapshotContent

func (self *tools) generateSnapshot(e *consensus.Event) (*ledger.SnapshotBlock, error) {
	head := self.chain.GetLatestSnapshotBlock()
	accounts := self.generateAccounts(head)

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
	self.log.Info("insert")
	return nil
}

func newChainRw() *tools {
	log := log15.New("module", "tools")
	return &tools{log: log}
}

func (self *tools) checkAddressLock(address types.Address) bool {
	unLocked := self.wt.KeystoreManager.IsUnLocked(address)
	return unLocked
}
func (self *tools) generateAccounts(head *ledger.SnapshotBlock) ledger.SnapshotContent {
	var needSnapshotAccounts []*ledger.AccountBlock

	var finalAccounts ledger.SnapshotContent

	for _, b := range needSnapshotAccounts {
		err := self.sVerifier.VerifyAccountTimeout(b.AccountAddress, head.Height+1)
		if err != nil {
			continue
		}
		finalAccounts[b.AccountAddress] = &ledger.SnapshotContentItem{AccountBlockHash: b.Hash, AccountBlockHeight: b.Height}
	}
	return finalAccounts
}
