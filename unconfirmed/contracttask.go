package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/verifier"
	"sync"
	"time"
)

type ContractTask struct {
	taskId int

	blocksPool *model.UnconfirmedBlocksPool
	pool       PoolAccess
	verifier   *verifier.AccountVerifier
	genBuilder *generator.GenBuilder

	status      int
	statusMutex sync.Mutex

	stopListener chan struct{}
	breaker      chan struct{}

	isSleeping bool
	wakeup     chan struct{}

	accEvent producer.AccountStartEvent
	cworker  *ContractWorker

	getNewBlocksFunc func(index int) *model.FromItem

	log log15.Logger
}

func NewContractTask(worker *ContractWorker, index int, getNewBlocksFunc func(index int) *model.FromItem) *ContractTask {
	return &ContractTask{
		taskId:           index,
		blocksPool:       worker.uBlocksPool,
		pool:             worker.pool,
		verifier:         worker.manager.verifier,
		genBuilder:       worker.manager.genBuilder,
		status:           Create,
		stopListener:     make(chan struct{}),
		breaker:          make(chan struct{}),
		wakeup:           make(chan struct{}),
		accEvent:         worker.accEvent,
		cworker:          worker,
		getNewBlocksFunc: getNewBlocksFunc,

		log: worker.log.New("taskid", index),
	}
}
func (task *ContractTask) WakeUp() {
	if task.isSleeping {
		task.wakeup <- struct{}{}
	}
}

func (task *ContractTask) work() {
	for {
		task.isSleeping = false
		if task.status == Stop {
			goto END
		}

		fItem := task.getNewBlocksFunc(task.taskId)
		if fItem == nil {
			goto WAIT
		}

		if task.ProcessOneQueue(fItem) {
			task.cworker.addIntoBlackList(fItem.Key, fItem.Value.Front().ToAddress)
		}

		continue

	WAIT:
		task.isSleeping = true
		select {
		case <-task.wakeup:
		case <-task.breaker:
			goto END
		default:
			break
		}
	}
END:
	task.log.Info("ContractTask send stopDispatcherListener ")
	task.stopListener <- struct{}{}
	task.log.Info("ContractTask Stop")
}

func (task *ContractTask) Start() {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Start {
		task.isSleeping = false

		go task.work()

		task.status = Start
	}
}

func (task *ContractTask) Stop() {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Stop {

		task.breaker <- struct{}{}
		close(task.breaker)

		<-task.stopListener
		close(task.stopListener)

		close(task.wakeup)

		task.status = Stop
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

func (task *ContractTask) ProcessOneQueue(fItem *model.FromItem) (intoBlackList bool) {
	// get db.go block from wakeup
	task.log.Info("Process the fromQueue,", "fromAddress", fItem.Key, "index", fItem.Index, "priority", fItem.Priority)

	bQueue := fItem.Value

	for i := 0; i < bQueue.Size(); i++ {
		var priorBlock *ledger.AccountBlock
		sBlock := bQueue.Dequeue()
		task.log.Info("Process to make the receiveBlock, its'sendBlock detail:", task.log.New("hash", sBlock.Hash))

		if task.pool.ExistInPool(&sBlock.ToAddress, &sBlock.Hash) {
			// Don't deal with it for the time being
			return true
		}

		if task.verifier.VerifyReceiveReachLimit(sBlock) {
			task.log.Info("Delete the UnconfirmedMeta: the recvBlock reach the max-limit count of existence.")
			task.blocksPool.WriteUnconfirmed(false, nil, sBlock)
			continue
		}

		block, err := task.PackReceiveBlock(sBlock, &task.accEvent.SnapshotHash, task.accEvent.Timestamp)
		if err != nil {
			task.log.Error("PackReceiveBlock Error", err)
			return true
		}
		genBuilder, err := task.genBuilder.PrepareVm(&task.accEvent.SnapshotHash, &block.PrevHash, &block.AccountAddress)
		if err != nil {
			task.log.Error("NewGenerator Error", err)
			return true
		}

		genResult := genBuilder.Build().GenerateWithBlock(generator.SourceTypeUnconfirmed, block)
		if err != nil {
			task.log.Error("GenerateTx error ignore, ", "error", err)
		}

		if genResult.BlockGenList == nil {
			if genResult.IsRetry == true {
				return true
			} else {
				task.blocksPool.WriteUnconfirmed(false, nil, sBlock)
			}
		} else {
			if genResult.IsRetry == true {
				return true
			}

			nowTime := time.Now().Unix()
			if nowTime >= task.accEvent.Etime.Unix() {
				task.breaker <- struct{}{}
				return true
			}
			for k, v := range genResult.BlockGenList {
				if err := task.pool.AddDirectBlock(sBlock, v); err != nil {
					return true
				}
				if k == len(genResult.BlockGenList)-1 {
					priorBlock = v.AccountBlock
				}

			}
		}
	WaitForVmDB:
		if task.verifier.VerifyUnconfirmedPriorBlockReceived(&priorBlock.Hash) == false {
			goto WaitForVmDB
		}
	}

	return false
}

func (task *ContractTask) PackReceiveBlock(sendBlock *ledger.AccountBlock, snapshotHash *types.Hash, timestamp time.Time) (*ledger.AccountBlock, error) {
	//task.statusMutex.Lock()
	//defer task.statusMutex.Unlock()
	//if task.status != Start {
	//	task.status = Start
	//}
	//
	//task.log.Info("PackReceiveBlock", "sendBlock",
	//	task.log.New("sendBlock.Hash", sendBlock.Hash), task.log.New("sendBlock.To", sendBlock.ToAddress))
	//
	//// fixme：remaining Nonce to add
	//block := &ledger.AccountBlock{
	//	BlockType:      0,
	//	Hash:           types.Hash{},
	//	Height:         0,
	//	PrevHash:       types.Hash{},
	//	AccountAddress: sendBlock.ToAddress,
	//	PublicKey:      nil, // contractAddress's receiveBlock's publicKey is from consensus node
	//	ToAddress:      types.Address{},
	//	FromBlockHash:  sendBlock.Hash,
	//	Amount:         sendBlock.Amount,
	//	TokenId:        sendBlock.TokenId,
	//	Quota:          sendBlock.Quota,
	//	Fee:            sendBlock.Fee,
	//	SnapshotHash:   task.accEvent.SnapshotHash,
	//	Data:           sendBlock.Data,
	//	Timestamp:      task.accEvent.Timestamp,
	//	StateHash:      types.Hash{},
	//	LogHash:        nil,
	//	Nonce:          nil,
	//	Signature:      nil,
	//}
	//preBlock, err := task.chain.GetLatestAccountBlock(&block.AccountAddress)
	//if err != nil {
	//	return nil, errors.New("GetLatestAccountBlock error" + err.Error())
	//}
	//if preBlock != nil {
	//	block.Hash = preBlock.Hash
	//	block.Height = preBlock.Height + 1
	//}
	return nil, nil
}
