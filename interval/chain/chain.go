package chain

import (
	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/store"
)

type BlockChain interface {
	face.SnapshotReader
	face.SnapshotWriter
	face.AccountReader
	face.AccountWriter
	SetChainListener(listener face.ChainListener)
}

type blockchain struct {
	ac       *accountsChain
	sc       *snapshotChain
	store    store.BlockStore
	listener face.ChainListener

	cfg *config.Chain
}

func NewChain(cfg *config.Chain) BlockChain {
	self := &blockchain{cfg: cfg}
	self.store = store.NewStore(cfg)
	self.listener = &defaultChainListener{}
	self.sc = newSnapshotChain(self.store)
	self.ac = newAccountsChain(self.store, self.listener)
	return self
}

// query received block by send block
func (bc *blockchain) GetAccountBySourceHash(address string, source string) *common.AccountStateBlock {
	b := bc.store.GetAccountBySourceHash(source)
	return b
}

func (bc *blockchain) NextAccountSnapshot() (common.HashHeight, []*common.AccountHashH, error) {
	head := bc.sc.head
	//common.SnapshotBlock{}
	var accounts []*common.AccountHashH
	var err error
	bc.ac.rangeFn(func(acct *accountChain) bool {
		a, e := acct.NextSnapshotPoint()
		if e != nil {
			err = e
			return false
		}
		accounts = append(accounts, a)
		return true
	})
	if len(accounts) == 0 {
		accounts = nil
	}

	return common.HashHeight{Hash: head.Hash(), Height: head.Height()}, accounts, err
}

//func (bc *blockchain) FindAccountAboveSnapshotHeight(address string, snapshotHeight uint64) *common.AccountStateBlock {
//	return bc.one(address).findAccountAboveSnapshotHeight(snapshotHeight)
//}

func (bc *blockchain) SetChainListener(listener face.ChainListener) {
	if listener == nil {
		return
	}
	bc.listener = listener
}

func (bc *blockchain) GenesisSnapshot() (*common.SnapshotBlock, error) {
	return GetGenesisSnapshot(), nil
}

func (bc *blockchain) HeadSnapshot() (*common.SnapshotBlock, error) {
	return bc.sc.Head(), nil
}

func (bc *blockchain) GetSnapshotByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	return bc.sc.GetBlockByHashH(hashH)
}

func (bc *blockchain) GetSnapshotByHash(hash string) *common.SnapshotBlock {
	return bc.sc.getBlockByHash(hash)
}

func (bc *blockchain) GetSnapshotByHeight(height uint64) *common.SnapshotBlock {
	return bc.sc.GetBlockHeight(height)
}

func (bc *blockchain) InsertSnapshotBlock(block *common.SnapshotBlock) error {
	err := bc.sc.insertBlock(block)
	if err == nil {
		// update next snapshot index
		for _, account := range block.Accounts {
			err := bc.ac.one(account.Addr).SnapshotPoint(block.Height(), block.Hash(), account)
			if err != nil {
				log.Error("update snapshot point fail.")
				return err
			}
		}
	}
	return err
}

func (bc *blockchain) RollbackSnapshotBlockTo(block *common.SnapshotBlock) ([]*common.SnapshotBlock, map[string]*common.AccountStateBlock) {
	bc.sc.GetBlockHeight()
	bc.ac.rangeFn(func(acctCh *accountChain) bool {
		acctCh.RollbackSnapshotPoint()
	})
	bc.ac.RollbackSnapshotTo(block)
}

func (bc *blockchain) RemoveSnapshotHead(block *common.SnapshotBlock) error {
	return bc.sc.removeBlock(block)
}

func (bc *blockchain) HeadAccount(address string) (*common.AccountStateBlock, error) {
	return bc.ac.one(address).Head(), nil
}

func (bc *blockchain) GetAccountByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHashH", time.Now())
	return bc.ac.one(address).GetBlockByHashH(hashH)
}

func (bc *blockchain) GetAccountByHash(address string, hash string) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHash", time.Now())
	return bc.ac.one(address).GetBlockByHash(address, hash)
}

func (bc *blockchain) GetAccountByHeight(address string, height uint64) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHeight", time.Now())
	return bc.ac.one(address).GetBlockByHeight(height)
}

func (bc *blockchain) InsertAccountBlock(address string, block *common.AccountStateBlock) error {
	return bc.ac.one(address).insertBlock(block)
}

//func (bc *blockchain) RemoveAccountHead(address string, block *common.AccountStateBlock) error {
//	return bc.one(address).removeBlock(block)
//}
func (bc *blockchain) RollbackSnapshotPoint(address string, start *common.SnapshotPoint, end *common.SnapshotPoint) error {
	return bc.ac.one(address).RollbackSnapshotPoint(start, end)
}

type defaultChainListener struct {
}

func (*defaultChainListener) SnapshotInsertCallback(block *common.SnapshotBlock) {

}

func (*defaultChainListener) SnapshotRemoveCallback(block *common.SnapshotBlock) {

}

func (*defaultChainListener) AccountInsertCallback(address string, block *common.AccountStateBlock) {

}

func (*defaultChainListener) AccountRemoveCallback(address string, block *common.AccountStateBlock) {
}
