package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

type Worker struct {
	vite          Vite
	address       types.Address
	breaker       chan struct{}
	newSignedTask chan struct{}
	log           log15.Logger

	waitSendTasks   []*sendTask
	addressUnlocked bool
	isWorking       bool
	flagMutex       sync.Mutex
	isClosed        bool
}

func NewWorker(vite Vite, addrese types.Address) *Worker {
	slave := &Worker{vite: vite,
		address:       addrese,
		breaker:       make(chan struct{}, 1),
		newSignedTask: make(chan struct{}, 100),
		log:           slog.New("Worker addr", addrese),
	}

	return slave
}

func (w *Worker) Close() error {

	w.vite.Ledger().Ac().RemoveListener(w.address)

	w.flagMutex.Lock()
	w.isClosed = true
	w.isWorking = false
	w.flagMutex.Unlock()

	w.breaker <- struct{}{}
	close(w.breaker)
	return nil
}

func (w *Worker) IsWorking() bool {
	w.flagMutex.Lock()
	defer w.flagMutex.Unlock()
	return w.isWorking
}

func (w *Worker) AddressUnlocked(unlocked bool) {
	w.addressUnlocked = unlocked
	if unlocked {
		w.log.Info("AddressUnlocked", "w.newSignedTask", w.newSignedTask)
		w.vite.Ledger().Ac().AddListener(w.address, w.newSignedTask)
	} else {
		w.log.Info("AddressUnlocked RemoveListener ")
		w.vite.Ledger().Ac().RemoveListener(w.address)
	}
}

func (w *Worker) sendNextUnConfirmed() (hasmore bool, err error) {
	w.log.Info("slaver auto send confirm task")
	ac := w.vite.Ledger().Ac()
	hashes, e := ac.GetUnconfirmedTxHashs(0, 1, 1, &w.address)

	if e != nil {
		w.log.Error("slaver auto GetUnconfirmedTxHashs", "err", e)
		return false, e
	}

	if len(hashes) == 0 {
		return false, nil
	}

	w.log.Info("slaver sendNextUnConfirmed: send receive transaction. ", "hash ", hashes[0])
	err = ac.CreateTx(&ledger.AccountBlock{
		AccountAddress: &w.address,
		FromHash:       hashes[0],
	})

	time.Sleep(2 * time.Second)
	return true, err
}


func (w *Worker) IsFirstSyncDone(done int) {
	if done == 1 {
		w.resume()
	} else {
		w.pause()
	}
}

func (w *Worker) StartWork() {

	w.log.Info("slaver StartWork is called")
	w.flagMutex.Lock()
	if w.isWorking {
		w.flagMutex.Unlock()
		w.log.Info("slaver is working")
		return
	}

	w.isWorking = true

	w.flagMutex.Unlock()
	w.log.Info("slaver start work")
	for {
		w.log.Debug("slaver working")

		if w.isClosed {
			break
		}
		if len(w.waitSendTasks) != 0 {
			for i, v := range w.waitSendTasks {
				w.log.Info("slaver send user task")
				err := w.vite.Ledger().Ac().CreateTxWithPassphrase(v.block, v.passphrase)
				if err == nil {
					w.log.Info("slaver send user task success")
					v.end <- ""
				} else {
					w.log.Info("slaver send user task", "err", err)
					v.end <- err.Error()
				}
				close(v.end)
				w.waitSendTasks = append(w.waitSendTasks[:i], w.waitSendTasks[i+1:]...)
			}
		}

		if w.addressUnlocked {
			hasmore, err := w.sendNextUnConfirmed()
			if err != nil {
				w.log.Error(err.Error())
			}
			w.log.Info("slaver sendNextUnConfirmed ", "hasmore", hasmore)
			if hasmore {
				continue
			} else {
				goto WAIT
			}
		}

	WAIT:
		w.log.Info("slaver start sleep")
		select {
		case <-w.newSignedTask:
			w.log.Info("slaver start awake")
			continue
		case <-w.breaker:
			w.log.Info("slaver start brake")
			break
		}

	}

	w.isWorking = false
	w.log.Info("slaver end work")
}

func (w *Worker) sendTask(task *sendTask) {
	w.waitSendTasks = append(w.waitSendTasks, task)

	w.log.Info("sendTask", "is working ", w.IsWorking())
	if w.IsWorking() {
		go func() { w.newSignedTask <- struct{}{} }()
	} else {
		go w.StartWork()
	}
}
func (w *Worker) resume() {

}
func (w *Worker) pause() {

}

func (w *Worker) StartContractWork() {
	// todo StartContractWork

}


func (w *Worker) StopContractWork() {

}
