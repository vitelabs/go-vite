package worker

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

const (
	Idle    = iota
	Running
	Waiting
	Dead
)

type ContractTask struct {
	vite Vite

	statusMutex sync.Mutex
	status      int

	subQueue chan *ledger.AccountBlock

	log log15.Logger
}

func NewContractTask() *ContractTask {
	return &ContractTask{
		vite:   nil,
		status: Idle,
		log:    log15.New("ContractTask"),
	}
}

func (c *ContractTask) Start(timestamp uint64) {
	c.subQueue = make(chan *ledger.AccountBlock, CACHE_SIZE)
LOOP:
	for {
		c.statusMutex.Lock()
		defer c.statusMutex.Unlock()

		if c.status == Dead {
			break LOOP
		}

		// get unconfirmed block from subQueue
		block := c.GetBlock()

		// generate block
		isRetry, blockList := c.GenerateBlock(block, timestamp)

		if blockList == nil {
			if !isRetry {
				if err := c.vite.Ledger().Ac().DeleteUnconfirmed(block); err != nil {
					c.log.Error("ContractTask.DeleteUnconfirmed Error", "Error", err)
				}
			}
			continue
		} else {
			// todo 6.pack block, comput hash, Sign, pack block, insert into Pool

		}
		c.status = Idle
	}
}

func (c *ContractTask) GenerateBlock(block *ledger.AccountBlock, timestamp uint64) (isRetry bool, blockList []*ledger.AccountBlock) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()

	if c.status != Running {
		c.status = Running
	}

	// todo 1. package the block with timestamp

	// todo 2. generate the new received TxBlock
	// todo 3.NewVM(stateDb Database, createBlockFunc CreateBlockFunc, config VMConfig) *VM
	// todo 4.(vm *VM) Run(block VmBlock) (blockList []VmBlock, logList []*Log, isRetry bool, err error)
	// todo 5.(vm *VM) Cancel()

	return isRetry, blockList
}

func (c *ContractTask) GetBlock() *ledger.AccountBlock {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()

	block := <-c.subQueue
	c.status = Running
	return block
}

func (c *ContractTask) Stop() {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()

	// stop all chan
	close(c.subQueue)

	if c.status != Dead {
		// todo: stop all
		c.status = Dead
	}
}

func (c *ContractTask) Close() error {
	c.Stop()
	return nil
}

func (c *ContractTask) Status() int {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	return c.status
}
