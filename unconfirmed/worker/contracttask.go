package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed"
	"sync"
	"time"
)

const (
	Idle = iota
	Running
	Waiting
	Dead
	MAX_ERR_RECV_COUNT = 3
)

type ContractTask struct {
	vite     Vite
	log      log15.Logger
	dbAccess *unconfirmed.UnconfirmedAccess

	status       int
	reRetry      bool
	stopListener chan struct{}
	breaker      chan struct{}

	subQueue chan *fromItem
	args     *unconfirmed.RightEvent

	statusMutex sync.Mutex
}

func (task *ContractTask) InitContractTask(vite Vite, dbAccess *unconfirmed.UnconfirmedAccess, args *unconfirmed.RightEvent) {
	task.vite = vite
	task.log = log15.New("ContractTask")
	task.dbAccess = dbAccess
	task.status = Idle
	task.reRetry = false
	task.stopListener = make(chan struct{}, 1)
	task.breaker = make(chan struct{}, 1)
	task.args = args
	task.subQueue = make(chan *fromItem, 1)
}

func (task *ContractTask) Start(blackList *map[string]bool) {
	for {
		task.statusMutex.Lock()
		defer task.statusMutex.Unlock()

		if task.status == Dead {
			goto END
		}

		fItem := task.GetFromItem()
		if intoBlackListBool := task.ProcessAQueue(fItem); intoBlackListBool == true {
			var blKey = fItem.value.Front().To.String() + fItem.key.String()
			if _, ok := (*blackList)[blKey]; !ok {
				(*blackList)[blKey] = true
			}
		}

		select {
		case <-task.breaker:
			goto END
		default:
			break
		}
		task.status = Idle
	}
END:
	task.log.Info("ContractTask send stopDispatcherListener ")
	task.stopListener <- struct{}{}
	task.log.Info("ContractTask Stop")
}

func (task *ContractTask) Stop() {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Dead {

		task.breaker <- struct{}{}
		close(task.breaker)

		<-task.stopListener
		close(task.stopListener)

		close(task.subQueue)

		task.status = Dead
	}
}

func (task *ContractTask) Close() error {
	task.Stop()
	return nil
}

func (task *ContractTask) Status() int {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	return task.status
}

func (task *ContractTask) ProcessAQueue(fItem *fromItem) (intoBlackList bool) {
	// get unconfirmed block from subQueue
	task.log.Info("Process the fromQueue,", task.log.New("fromQueueDetail",
		task.log.New("fromAddress", fItem.key),
		task.log.New("index", fItem.index),
		task.log.New("priority", fItem.priority)))

	// todo newVmDb
	// NewVmDB(snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) error

	bQueue := fItem.value

	for i := 0; i < bQueue.Size(); i++ {
		sBlock := bQueue.Dequeue()
		task.log.Info("Process to make the receiveBlock, its'sendBlock detail:", task.log.New("hash", sBlock.Hash))
		if ExistInPool(sBlock.To, sBlock.FromHash) {
			// Don't deal with it for the time being
			return true
		}

		recvType, count := task.CheckChainReceiveCount(sBlock.To, sBlock.Hash)
		if recvType == 5 && count > MAX_ERR_RECV_COUNT {
			task.log.Info("Delete the UnconfirmedMeta: the recvErrBlock reach the max-limit count of existence.")
			task.dbAccess.WriteUnconfirmed(false, nil, sBlock)
			continue
		}

		block := task.PackReceiveBlock(sBlock)

		isRetry, blockList, err := task.GenerateBlocks(block)
		if err != nil {
			task.log.Error("GenerateBlocks Error", err)
		}

		// todo: maintain the gid-contractAddress into VmDB
		// AddConsensusGroup(group ConsensusGroup...)

		if blockList == nil {
			if isRetry == true {
				return true
			} else {
				task.dbAccess.WriteUnconfirmed(false, nil, sBlock)
			}
		} else {
			if isRetry == true {
				return true
			}
			// todo 1.Check the time
			// If time out of the out-block period,
			// need to 1.stop the task 2.delete the DB and then 3.return true, else continue insert pool
			nowTime := uint64(time.Now().Unix())
			if nowTime >= task.args.EndTs {
				task.breaker <- struct{}{}
				return true
			}

			if err := task.InertBlockListIntoPool(sBlock, blockList); err != nil {
				return true
			}
		}

	WaitForVmDB:
		if _, c := task.CheckChainReceiveCount(sBlock.To, sBlock.Hash); c < 1 {
			task.log.Info("Wait for VmDB: the prev SendBlock's receiveBlock hasn't existed in Chain. ")
			goto WaitForVmDB
		}
	}

	return false
}

func (task *ContractTask) GetFromItem() *fromItem {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	fItem := <-task.subQueue
	task.status = Running
	return fItem
}

func (task *ContractTask) CheckChainReceiveCount(fromAddress *types.Address, fromHash *types.Hash) (recvType int, count int) {
	return recvType, count
}

func (task *ContractTask) InertBlockListIntoPool(sendBlock *unconfirmed.AccountBlock, blockList []*ledger.AccountBlock) error {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Running {
		task.status = Running
	}

	task.log.Info("InertBlockListIntoPool")

	// todo 2.Insert into Pool
	// if insert return err, this func return false, else return true

	return nil
}

func (task *ContractTask) GenerateBlocks(recvBlock *unconfirmed.AccountBlock) (isRetry bool, blockList []*ledger.AccountBlock, err error) {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Running {
		task.status = Running
	}
	return false, nil, nil
}

func (task *ContractTask) PackReceiveBlock(sendBlock *unconfirmed.AccountBlock) *unconfirmed.AccountBlock {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Running {
		task.status = Running
	}

	task.log.Info("PackReceiveBlock", "sendBlock",
		task.log.New("sendBlock.Hash", sendBlock.Hash), task.log.New("sendBlock.To", sendBlock.To))

	// todo pack the block with task.args, comput hash, Sign,
	block := &unconfirmed.AccountBlock{
		From:            nil,
		To:              nil,
		Height:          nil,
		Type:            0,
		PrevHash:        nil,
		FromHash:        nil,
		Amount:          nil,
		TokenId:         nil,
		CreateFee:       nil,
		Data:            nil,
		StateHash:       types.Hash{},
		SummaryHashList: nil,
		LogHash:         types.Hash{},
		SnapshotHash:    types.Hash{},
		Depth:           0,
		Quota:           0,
		Hash:            nil,
		Balance:         nil,
	}

	hash, err := block.ComputeHash()
	if err != nil {
		task.log.Error("ComputeHash Error")
		return nil
	}
	block.Hash = hash

	return block
}
