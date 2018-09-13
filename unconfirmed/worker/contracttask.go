package worker

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"sync"
	"time"
)

const (
	Idle = iota
	Running
	Waiting
	Dead
	MaxErrRecvCount = 3
)

type ContractTask struct {
	vite    vite.Vite
	log     log15.Logger
	uAccess *model.UAccess
	wAccess *wallet.Manager

	status       int
	reRetry      bool
	stopListener chan struct{}
	breaker      chan struct{}

	subQueue chan *model.FromItem
	args     *RightEvent

	statusMutex sync.Mutex
}

func (task *ContractTask) InitContractTask(uAccess *model.UAccess, wAccess *wallet.Manager, args *RightEvent) {
	task.log = log15.New("ContractTask")
	task.uAccess = uAccess
	task.wAccess = wAccess
	task.status = Idle
	task.reRetry = false
	task.stopListener = make(chan struct{}, 1)
	task.breaker = make(chan struct{}, 1)
	task.args = args
	task.subQueue = make(chan *model.FromItem, 1)
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
			var blKey = fItem.Value.Front().ToAddress.String() + fItem.Key.String()
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

func (task *ContractTask) ProcessAQueue(fItem *model.FromItem) (intoBlackList bool) {
	// get db.go block from subQueue
	task.log.Info("Process the fromQueue,", task.log.New("fromQueueDetail",
		task.log.New("fromAddress", fItem.Key),
		task.log.New("index", fItem.Index),
		task.log.New("priority", fItem.Priority)))

	bQueue := fItem.Value

	for i := 0; i < bQueue.Size(); i++ {
		sBlock := bQueue.Dequeue()
		task.log.Info("Process to make the receiveBlock, its'sendBlock detail:", task.log.New("hash", sBlock.Hash))
		if ExistInPool(sBlock.ToAddress, sBlock.Hash) {
			// Don't deal with it for the time being
			return true
		}

		recvType, count := task.CheckChainReceiveCount(&sBlock.ToAddress, &sBlock.Hash)
		if recvType == 5 && count > MaxErrRecvCount {
			task.log.Info("Delete the UnconfirmedMeta: the recvErrBlock reach the max-limit count of existence.")
			task.uAccess.WriteUnconfirmed(false, nil, sBlock)
			continue
		}

		block, err := task.PackReceiveBlock(sBlock)
		if err != nil {
			task.log.Error("PackReceiveBlock Error", err)
			return true
		}

		gen, err := generator.NewGenerator(task.uAccess.Chain, task.args.SnapshotHash, &block.PrevHash, &block.AccountAddress)
		if err != nil {
			task.log.Error("NewGenerator Error", err)
			return true
		}
		genResult := gen.GenerateTx(generator.SourceTypeUnconfirmed, block)
		if err != nil {
			task.log.Error("GenerateTx error ignore, ", "error", err)
		}
		blockGen := genResult.BlockGenList[0]
		//todo: only sign the first block, special care for contract's BlockList returned
		var signErr error
		blockGen.Block.Signature, blockGen.Block.PublicKey, signErr =
			task.wAccess.KeystoreManager.SignData(blockGen.Block.AccountAddress, blockGen.Block.Hash.Bytes())
		if signErr != nil {
			task.log.Error("SignData Error", signErr)
			return true
		}

		if genResult.BlockGenList == nil {
			if genResult.IsRetry == true {
				return true
			} else {
				task.uAccess.WriteUnconfirmed(false, nil, sBlock)
			}
		} else {
			if genResult.IsRetry == true {
				return true
			}

			nowTime := time.Now().Unix()
			if nowTime >= task.args.EndTs.Unix() {
				task.breaker <- struct{}{}
				return true
			}

			if err := task.InertBlockListIntoPool(sBlock, blockGen); err != nil {
				return true
			}
		}

	WaitForVmDB:
		if _, c := task.CheckChainReceiveCount(&sBlock.ToAddress, &sBlock.Hash); c < 1 {
			task.log.Info("Wait for VmDB: the prev SendBlock's receiveBlock hasn't existed in Chain. ")
			goto WaitForVmDB
		}
	}

	return false
}

func (task *ContractTask) GetFromItem() *model.FromItem {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	fItem := <-task.subQueue
	task.status = Running
	return fItem
}

func (task *ContractTask) CheckChainReceiveCount(fromAddress *types.Address, fromHash *types.Hash) (recvType int, count int) {
	return recvType, count
}

func (task *ContractTask) PackReceiveBlock(sendBlock *ledger.AccountBlock) (*ledger.AccountBlock, error) {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Running {
		task.status = Running
	}

	task.log.Info("PackReceiveBlock", "sendBlock",
		task.log.New("sendBlock.Hash", sendBlock.Hash), task.log.New("sendBlock.To", sendBlock.ToAddress))

	// todo pack the block with task.args
	block := &ledger.AccountBlock{
		BlockType:         0,
		Hash:              types.Hash{},
		Height:            0,
		PrevHash:          types.Hash{},
		AccountAddress:    sendBlock.ToAddress,
		PublicKey:         nil, // contractAddress's receiveBlock's publicKey is from consensus node
		ToAddress:         types.Address{},
		FromBlockHash:     sendBlock.Hash,
		Amount:            sendBlock.Amount,
		TokenId:           sendBlock.TokenId,
		Quota:             sendBlock.Quota,
		Fee:               sendBlock.Fee,
		SnapshotHash:      *task.args.SnapshotHash,
		Data:              sendBlock.Data,
		Timestamp:         &task.args.Timestamp,
		StateHash:         types.Hash{},
		LogHash:           types.Hash{},
		Nonce:             nil,
		SendBlockHashList: nil,
		Signature:         nil,
	}
	preBlock, err := task.uAccess.Chain.GetLatestAccountBlock(&block.AccountAddress)
	if err != nil {
		return nil, errors.New("GetLatestAccountBlock error" + err.Error())
	}
	if preBlock != nil {
		block.Hash = preBlock.Hash
		block.Height = preBlock.Height + 1
	}
	return block, nil
}

