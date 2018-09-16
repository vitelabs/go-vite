package unconfirmed

import (
	"errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
	"sync"
	"time"
)

const (
	MaxErrRecvCount = 3
	Idle            = Create
)

type ContractTask struct {
	blocksPool *model.UnconfirmedBlocksPool
	wallet     *wallet.Manager
	pool       PoolAccess
	chain      *chain.Chain
	verifier   *verifier.AccountVerifier

	status      int
	statusMutex sync.Mutex

	reRetry bool

	stopListener chan struct{}
	breaker      chan struct{}

	subQueue chan *model.FromItem
	accevent producer.AccountStartEvent
	cworker  *ContractWorker

	log log15.Logger
}

func NewContractTask(worker *ContractWorker, index int) *ContractTask {
	return &ContractTask{
		blocksPool:   worker.uBlocksPool,
		wallet:       worker.wallet,
		pool:         worker.pool,
		chain:        worker.chain,
		verifier:     worker.verifer,
		status:       Idle,
		reRetry:      false,
		stopListener: make(chan struct{}),
		breaker:      make(chan struct{}),
		subQueue:     make(chan *model.FromItem),
		accevent:     worker.accevent,
		cworker:      worker,
		log:          worker.log.New("taskid", index),
	}
}

func (task *ContractTask) Start() {
	for {
		if task.status == Stop {
			goto END
		}

		fItem := task.GetFromItem()
		if task.ProcessOneQueue(fItem) {
			task.cworker.addIntoBlackList(fItem.Key, fItem.Value.Front().ToAddress)
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
	if task.status != Stop {

		task.breaker <- struct{}{}
		close(task.breaker)

		<-task.stopListener
		close(task.stopListener)

		close(task.subQueue)

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
	// get db.go block from subQueue
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

		block, err := task.PackReceiveBlock(sBlock)
		if err != nil {
			task.log.Error("PackReceiveBlock Error", err)
			return true
		}

		gen, err := generator.NewGenerator(task.chain, task.wallet.KeystoreManager,
			&task.accevent.SnapshotHash, &block.PrevHash, &block.AccountAddress)
		if err != nil {
			task.log.Error("NewGenerator Error", err)
			return true
		}
		genResult := gen.GenerateWithBlock(generator.SourceTypeUnconfirmed, block)
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
			if nowTime >= task.accevent.Etime.Unix() {
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
		if task.verifier.VerifyUnconfirmedPriorBlockReceived(&priorBlock.Hash) == verifier.PENDING {
			goto WaitForVmDB
		}
	}

	return false
}

func (task *ContractTask) GetFromItem() *model.FromItem {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	fItem := <-task.subQueue
	task.status = Start
	return fItem
}

func (task *ContractTask) PackReceiveBlock(sendBlock *ledger.AccountBlock) (*ledger.AccountBlock, error) {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Start {
		task.status = Start
	}

	task.log.Info("PackReceiveBlock", "sendBlock",
		task.log.New("sendBlock.Hash", sendBlock.Hash), task.log.New("sendBlock.To", sendBlock.ToAddress))

	// fixme：remaining Nonce to add
	block := &ledger.AccountBlock{
		BlockType:      0,
		Hash:           types.Hash{},
		Height:         0,
		PrevHash:       types.Hash{},
		AccountAddress: sendBlock.ToAddress,
		PublicKey:      nil, // contractAddress's receiveBlock's publicKey is from consensus node
		ToAddress:      types.Address{},
		FromBlockHash:  sendBlock.Hash,
		Amount:         sendBlock.Amount,
		TokenId:        sendBlock.TokenId,
		Quota:          sendBlock.Quota,
		Fee:            sendBlock.Fee,
		SnapshotHash:   task.accevent.SnapshotHash,
		Data:           sendBlock.Data,
		Timestamp:      task.accevent.Timestamp,
		StateHash:      types.Hash{},
		LogHash:        nil,
		Nonce:          nil,
		Signature:      nil,
	}
	preBlock, err := task.chain.GetLatestAccountBlock(&block.AccountAddress)
	if err != nil {
		return nil, errors.New("GetLatestAccountBlock error" + err.Error())
	}
	if preBlock != nil {
		block.Hash = preBlock.Hash
		block.Height = preBlock.Height + 1
	}
	return block, nil
}
