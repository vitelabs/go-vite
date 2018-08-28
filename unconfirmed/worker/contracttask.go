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
	vite        Vite

	statusMutex sync.Mutex
	status      int

	log         log15.Logger
}

func NewContractTask() *ContractTask {
	return nil
}

func (c *ContractTask) Start(queue *BlockQueue) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	select {
	case c.status == Idle:
		if queue.IsEmpty() || queue.Size() < 2*POMAXPROCS {
			queue.PullFromMem()
		}
		block := queue.Dequeue()
		// todo: package the block with timestamp

		// todo: generate the new received TxBlock
		// todo: NewVM(stateDb Database, createBlockFunc CreateBlockFunc, config VMConfig) *VM
		// todo: (vm *VM) Run(block VmBlock) (blockList []VmBlock, logList []*Log, isRetry bool, err error)
		// todo: (vm *VM) Cancel()

		var isRetry bool
		var blockList = []ledger.AccountBlock{}
		if blockList == nil {
			if !isRetry {
				if err := c.vite.Ledger().Ac().DeleteUnconfirmed(block); err != nil {
					log15.Error("ContractTask.DeleteUnconfirmed Error")
				}
			}
			c.status = Idle
			break
		} else {
			// todo: pack block, comput hash, Sign, pack block, insert into Pool
		}

		c.status = Running
	case c.status == Running:
		c.status = Waiting
	case c.status == Waiting:
		// todo: consider when run into Waiting
	case c.status == Dead:
		c.Stop()
	default:
		break
	}

}

func (c *ContractTask) Stop() {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	if c.status != Dead {
		// todo: stop all
		c.status = Dead
	}
}

func (c *ContractTask) Close() error {
	c.Stop()
	return nil
}
