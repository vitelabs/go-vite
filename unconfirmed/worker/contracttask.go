package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed"
	"sync"
)

const (
	Idle = iota
	Running
	Waiting
	Dead
)
const MAX_ERR_RECV_COUNT = 3

type ContractTask struct {
	vite Vite

	statusMutex sync.Mutex
	status      int

	subQueue chan *ledger.AccountBlock
	args     *unconfirmed.RightEventArgs

	reRetry bool

	log log15.Logger
}

func (c *ContractTask) InitContractTask(vite Vite, args *unconfirmed.RightEventArgs) {
	c.vite = vite
	c.status = Idle
	c.subQueue = make(chan *ledger.AccountBlock, CACHE_SIZE)
	c.args = args
	c.log = log15.New("ContractTask")
}

func (c *ContractTask) Start(blackList *map[types.Hash]bool) {
	for {
		c.statusMutex.Lock()
		defer c.statusMutex.Unlock()

		if c.status == Dead {
			goto END
		}

		// get unconfirmed block from subQueue
		sendBlock := c.GetBlock()

		if ExistInPool(sendBlock.To, sendBlock.Hash) {
			c.log.Error("ExistInPool Check failed")
			(*blackList)[*sendBlock.Hash] = true
			continue
		}

		if c.CheckReceiveErrCount(sendBlock.To, sendBlock.Hash) > MAX_ERR_RECV_COUNT {
			c.vite.Ledger().Ac().DeleteUnconfirmed(sendBlock)
			continue
		}

		block, err := c.PackReceiveBlock(sendBlock)
		if err != nil {
			c.log.Error("PackReceiveBlock Failed", "Error", err.Error())
			continue
		}

		isRetry, blockList := c.GenerateBlocks(block)


		// todo: maintain the gid-contractAddress into VmDB
		// AddConsensusGroup(group ConsensusGroup,)

		if blockList == nil {
			if isRetry == true {
				(*blackList)[*sendBlock.Hash] = true
			} else {
				c.vite.Ledger().Ac().DeleteUnconfirmed(sendBlock)
			}
		} else {
			if isRetry == true {
				(*blackList)[*sendBlock.Hash] = true
			}
			c.InertBlockListIntoPool(sendBlock, blockList)
		}

		c.status = Idle
	}
END:
	c.log.Info("ContractTask Stop")
}

func (c *ContractTask) CheckReceiveErrCount(fromAddress *types.Address, fromHash *types.Hash) (count int) {

	return count
}

func (c *ContractTask) InertBlockListIntoPool(sendBlock *ledger.AccountBlock, blockList []*ledger.AccountBlock) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	if c.status != Running {
		c.status = Running
	}

	c.log.Info("InertBlockListIntoPool")

	// todo: insert into Pool
}

func (c *ContractTask) GenerateBlocks(recvBlock *ledger.AccountBlock) (isRetry bool, blockList []*ledger.AccountBlock) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	if c.status != Running {
		c.status = Running
	}
	return false, nil
}

func (c *ContractTask) PackReceiveBlock(sendBlock *ledger.AccountBlock) (*ledger.AccountBlock, error) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	if c.status != Running {
		c.status = Running
	}

	c.log.Info("PackReceiveBlock", "sendBlock",
		c.log.New("sendBlock.Hash", sendBlock.Hash), c.log.New("sendBlock.To", sendBlock.To))

	// todo pack the block with c.args, comput hash, Sign,
	block := &ledger.AccountBlock{}

	hash, err := block.ComputeHash()
	if err != nil {
		c.log.Error("ComputeHash Error")
		return nil, err
	}
	block.Hash = hash

	return block, nil
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
