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
	"github.com/vitelabs/go-vite/interval/utils"
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

func (l *ledger) GetAccountBalance(address string) int {
	head, _ := l.bc.HeadAccount(address)
	if head != nil {
		return head.Amount
	} else {
		return 0
	}
}

func (l *ledger) Chain() chain.BlockChain {
	return l.bc
}

func (l *ledger) Pool() pool.BlockPool {
	return l.bpool
}

func (l *ledger) MiningSnapshotBlock(address string, timestamp int64) error {
	//l.pendingSc.AddDirectBlock(block)
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()

	hashH, accounts, err := l.bc.NextAccountSnapshot()

	if err != nil {
		log.Error("get next accounts snapshot err. ", err)
		return err
	}

	block := common.NewSnapshotBlock(hashH.Height+1, "", hashH.Hash, address, time.Unix(timestamp, 0), accounts)
	block.SetHash(utils.CalculateSnapshotHash(block))

	err = l.bpool.AddDirectSnapshotBlock(block)
	if err != nil {
		log.Error("add direct block error. ", err)
		return err
	}
	l.syncer.Sender().BroadcastSnapshotBlocks([]*common.SnapshotBlock{block})
	return nil
}

func (l *ledger) RequestAccountBlock(from string, to string, amount int) error {
	defer monitor.LogTime("ledger", "requestAccount", time.Now())
	headAccount, _ := l.bc.HeadAccount(from)
	headSnaphost, _ := l.bc.HeadSnapshot()

	newBlock := common.NewAccountBlockFrom(headAccount, from, time.Now(), amount, headSnaphost,
		common.SEND, from, to, nil)
	newBlock.SetHash(utils.CalculateAccountHash(newBlock))
	err := l.bpool.AddDirectAccountBlock(from, newBlock)
	if err == nil {
		l.syncer.Sender().BroadcastAccountBlocks(from, []*common.AccountStateBlock{newBlock})
	}
	return err
}
func (l *ledger) ResponseAccountBlock(from string, to string, reqHash string) error {
	defer monitor.LogTime("ledger", "responseAccount", time.Now())
	b := l.bc.GetAccountByHash(from, reqHash)
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
	prev, _ := l.bc.HeadAccount(to)
	if prev != nil {
		height = prev.Height() + 1
		prevHash = prev.Hash()
		prevAmount = prev.Amount
	}
	snapshotBlock, _ := l.bc.HeadSnapshot()

	modifiedAmount := -reqBlock.ModifiedAmount
	block := common.NewAccountBlock(height, "", prevHash, to, time.Now(), prevAmount+modifiedAmount, modifiedAmount, snapshotBlock.Height(), snapshotBlock.Hash(),
		common.RECEIVED, from, to, &common.HashHeight{Hash: reqHash, Height: reqBlock.Height()})
	block.SetHash(utils.CalculateAccountHash(block))

	err := l.bpool.AddDirectAccountBlock(to, block)
	if err == nil {
		l.syncer.Sender().BroadcastAccountBlocks(to, []*common.AccountStateBlock{block})
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

func (l *ledger) Init(syncer syncer.Syncer) {
	l.syncer = syncer

	l.bpool.Init(syncer.Fetcher())
	l.reqPool = newReqPool()
	l.bc.SetChainListener(l.reqPool)
}

func (l *ledger) ListRequest(address string) []*Req {
	reqs := l.reqPool.getReqs(address)
	return reqs
}

func (l *ledger) Start() {
	l.bpool.Start()
}
func (l *ledger) Stop() {
	l.bpool.Stop()
}
func (l *ledger) ListSnapshotBlock() []*common.SnapshotBlock {
	var blocks []*common.SnapshotBlock
	head, _ := l.bc.HeadSnapshot()
	if head == nil {
		return blocks
	}
	for i := uint64(0); i < head.Height(); i++ {
		blocks = append(blocks, l.bc.GetSnapshotByHeight(i))
	}
	if head.Height() >= 0 {
		blocks = append(blocks, head)
	}
	return blocks
}

func (l *ledger) ListAccountBlock(address string) []*common.AccountStateBlock {
	var blocks []*common.AccountStateBlock
	head, _ := l.bc.HeadAccount(address)
	if head == nil {
		return blocks
	}
	for i := uint64(0); i < head.Height(); i++ {
		blocks = append(blocks, l.bc.GetAccountByHeight(address, i))
	}
	if head.Height() >= 0 {
		blocks = append(blocks, head)
	}
	return blocks
}
