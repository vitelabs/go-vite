package ledger

import (
	"errors"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/interval/chain"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/pool"
	"github.com/vitelabs/go-vite/interval/syncer"
	"github.com/vitelabs/go-vite/interval/tools"
)

type Ledger interface {
	MiningSnapshotBlock(address string, timestamp int64) error
	// from self
	RequestAccountBlock(from string, to string, amount int) error
	ResponseAccountBlock(from string, to string, reqHash string) error
	// create account genesis block
	GetAccountBalance(address string) int

	ListRequest(address string) []*Req
	Start()
	Stop()
	Init(syncer syncer.Syncer)

	ListAccountBlock(address string) []*common.AccountStateBlock
	ListSnapshotBlock() []*common.SnapshotBlock
	Chain() chain.BlockChain
	Pool() pool.BlockPool
}

type ledger struct {
	bc      chain.BlockChain
	reqPool *reqPool
	bpool   pool.BlockPool

	syncer  syncer.Syncer
	rwMutex *sync.RWMutex
}

func (self *ledger) GetAccountBalance(address string) int {
	head, _ := self.bc.HeadAccount(address)
	if head != nil {
		return head.Amount
	} else {
		return 0
	}
}

func (self *ledger) Chain() chain.BlockChain {
	return self.bc
}

func (self *ledger) Pool() pool.BlockPool {
	return self.bpool
}

func (self *ledger) MiningSnapshotBlock(address string, timestamp int64) error {
	//self.pendingSc.AddDirectBlock(block)
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()

	hashH, accounts, err := self.bc.NextAccountSnapshot()

	if err != nil {
		log.Error("get next accounts snapshot err. ", err)
		return err
	}

	block := common.NewSnapshotBlock(hashH.Height+1, "", hashH.Hash, address, time.Unix(timestamp, 0), accounts)
	block.SetHash(tools.CalculateSnapshotHash(block))

	err = self.bpool.AddDirectSnapshotBlock(block)
	if err != nil {
		log.Error("add direct block error. ", err)
		return err
	}
	self.syncer.Sender().BroadcastSnapshotBlocks([]*common.SnapshotBlock{block})
	return nil
}

func (self *ledger) RequestAccountBlock(from string, to string, amount int) error {
	defer monitor.LogTime("ledger", "requestAccount", time.Now())
	headAccount, _ := self.bc.HeadAccount(from)
	headSnaphost, _ := self.bc.HeadSnapshot()

	newBlock := common.NewAccountBlockFrom(headAccount, from, time.Now(), amount, headSnaphost,
		common.SEND, from, to, nil)
	newBlock.SetHash(tools.CalculateAccountHash(newBlock))
	err := self.bpool.AddDirectAccountBlock(from, newBlock)
	if err == nil {
		self.syncer.Sender().BroadcastAccountBlocks(from, []*common.AccountStateBlock{newBlock})
	}
	return err
}
func (self *ledger) ResponseAccountBlock(from string, to string, reqHash string) error {
	defer monitor.LogTime("ledger", "responseAccount", time.Now())
	b := self.bc.GetAccountByHash(from, reqHash)
	if b == nil {
		return errors.New("not exist for account[" + from + "]block[" + reqHash + "]")
	}
	if b.Hash() != reqHash {
		return errors.New("GetByHashError, ReqHash:" + reqHash + ", RealHash:" + b.Hash())
	}

	reqBlock := b

	height := common.FirstHeight
	prevHash := ""
	prevAmount := 0
	prev, _ := self.bc.HeadAccount(to)
	if prev != nil {
		height = prev.Height() + 1
		prevHash = prev.Hash()
		prevAmount = prev.Amount
	}
	snapshotBlock, _ := self.bc.HeadSnapshot()

	modifiedAmount := -reqBlock.ModifiedAmount
	block := common.NewAccountBlock(height, "", prevHash, to, time.Now(), prevAmount+modifiedAmount, modifiedAmount, snapshotBlock.Height(), snapshotBlock.Hash(),
		common.RECEIVED, from, to, &common.HashHeight{Hash: reqHash, Height: reqBlock.Height()})
	block.SetHash(tools.CalculateAccountHash(block))

	err := self.bpool.AddDirectAccountBlock(to, block)
	if err == nil {
		self.syncer.Sender().BroadcastAccountBlocks(to, []*common.AccountStateBlock{block})
	}
	return err
}

func NewLedger(bc chain.BlockChain) *ledger {
	ledger := &ledger{}
	ledger.rwMutex = new(sync.RWMutex)
	ledger.bc = bc
	ledger.bpool = pool.NewPool(ledger.bc, ledger.rwMutex)
	return ledger
}

func (self *ledger) Init(syncer syncer.Syncer) {
	self.syncer = syncer

	self.bpool.Init(syncer.Fetcher())
	self.reqPool = newReqPool()
	self.bc.SetChainListener(self.reqPool)
}

func (self *ledger) ListRequest(address string) []*Req {
	reqs := self.reqPool.getReqs(address)
	return reqs
}

func (self *ledger) Start() {
	self.bpool.Start()
}
func (self *ledger) Stop() {
	self.bpool.Stop()
}
func (self *ledger) ListSnapshotBlock() []*common.SnapshotBlock {
	var blocks []*common.SnapshotBlock
	head, _ := self.bc.HeadSnapshot()
	if head == nil {
		return blocks
	}
	for i := uint64(0); i < head.Height(); i++ {
		blocks = append(blocks, self.bc.GetSnapshotByHeight(i))
	}
	if head.Height() >= 0 {
		blocks = append(blocks, head)
	}
	return blocks
}

func (self *ledger) ListAccountBlock(address string) []*common.AccountStateBlock {
	var blocks []*common.AccountStateBlock
	head, _ := self.bc.HeadAccount(address)
	if head == nil {
		return blocks
	}
	for i := uint64(0); i < head.Height(); i++ {
		blocks = append(blocks, self.bc.GetAccountByHeight(address, i))
	}
	if head.Height() >= 0 {
		blocks = append(blocks, head)
	}
	return blocks
}
