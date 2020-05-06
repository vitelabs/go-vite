package chain

import (
	"sync"
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
	ac       sync.Map
	sc       *snapshotChain
	store    store.BlockStore
	listener face.ChainListener

	cfg *config.Chain

	mu sync.Mutex // account chain init
}

func NewChain(cfg *config.Chain) BlockChain {
	self := &blockchain{cfg: cfg}
	self.store = store.NewStore(cfg)
	self.sc = newSnapshotChain(self.store)
	self.listener = &defaultChainListener{}
	return self
}
func (bc *blockchain) selfAc(addr string) *accountChain {
	chain, ok := bc.ac.Load(addr)
	if !ok {
		c := newAccountChain(addr, bc.listener, bc.store)
		bc.mu.Lock()
		defer bc.mu.Unlock()
		bc.ac.Store(addr, c)
		chain, _ = bc.ac.Load(addr)
	}
	return chain.(*accountChain)
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
	bc.ac.Range(func(k, v interface{}) bool {
		a, e := v.(*accountChain).NextSnapshotPoint()
		if e != nil {
			return true
		}
		accounts = append(accounts, a)
		return true
	})
	if len(accounts) == 0 {
		accounts = nil
	}

	return common.HashHeight{Hash: head.Hash(), Height: head.Height()}, accounts, nil
}

func (bc *blockchain) FindAccountAboveSnapshotHeight(address string, snapshotHeight uint64) *common.AccountStateBlock {
	return bc.selfAc(address).findAccountAboveSnapshotHeight(snapshotHeight)
}

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
	err := bc.sc.insertChain(block)
	if err == nil {
		// update next snapshot index
		for _, account := range block.Accounts {
			err := bc.selfAc(account.Addr).SnapshotPoint(block.Height(), block.Hash(), account)
			if err != nil {
				log.Error("update snapshot point fail.")
				return err
			}
		}
	}
	return err
}

func (bc *blockchain) RemoveSnapshotHead(block *common.SnapshotBlock) error {
	return bc.sc.removeChain(block)
}

func (bc *blockchain) HeadAccount(address string) (*common.AccountStateBlock, error) {
	return bc.selfAc(address).Head(), nil
}

func (bc *blockchain) GetAccountByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHashH", time.Now())
	return bc.selfAc(address).GetBlockByHashH(hashH)
}

func (bc *blockchain) GetAccountByHash(address string, hash string) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHash", time.Now())
	return bc.selfAc(address).GetBlockByHash(address, hash)
}

func (bc *blockchain) GetAccountByHeight(address string, height uint64) *common.AccountStateBlock {
	defer monitor.LogTime("chain", "accountByHeight", time.Now())
	return bc.selfAc(address).GetBlockByHeight(height)
}

func (bc *blockchain) InsertAccountBlock(address string, block *common.AccountStateBlock) error {
	return bc.selfAc(address).insertChain(block)
}

func (bc *blockchain) RemoveAccountHead(address string, block *common.AccountStateBlock) error {
	return bc.selfAc(address).removeChain(block)
}
func (bc *blockchain) RollbackSnapshotPoint(address string, start *common.SnapshotPoint, end *common.SnapshotPoint) error {
	return bc.selfAc(address).RollbackSnapshotPoint(start, end)
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
