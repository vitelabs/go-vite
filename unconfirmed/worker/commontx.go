package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type CommonTxWorker struct {
	vite    Vite
	address types.Address
	log     log15.Logger

	status    int
	isWorking bool

	breaker                  chan struct{}
	newUnconfirmedTxListener chan struct{}

	lifecycleMutex sync.Mutex
}

func NewCommonTxWorker(vite Vite, addrese types.Address) *CommonTxWorker {
	return &CommonTxWorker{
		status:  Create,
		vite:    vite,
		address: addrese,
		log:     log15.New("CommonTxWorker addr", addrese),
	}
}

func (w *CommonTxWorker) Close() error {
	w.Stop()
	return nil
}

func (w *CommonTxWorker) IsWorking() bool {
	w.lifecycleMutex.Lock()
	defer w.lifecycleMutex.Unlock()
	return w.isWorking
}

func (w *CommonTxWorker) Start() {
	if w.status != Start {
		// 0. init the break chan
		w.breaker = make(chan struct{}, 1)

		// 1. listen unconfirmed tx in ledger
		w.newUnconfirmedTxListener = make(chan struct{}, 100)
		w.vite.Ledger().Ac().AddListener(w.address, w.newUnconfirmedTxListener)
		w.status = Start

		go w.startWork()

	} else {
		// 至少要让它跑一次start work

	}
}
func (w *CommonTxWorker) Stop() {
	if w.status != Stop {
		w.breaker <- struct{}{}
		close(w.breaker)

		w.vite.Ledger().Ac().RemoveListener(w.address)
		close(w.newUnconfirmedTxListener)

		w.status = Stop
	}
}
func (w CommonTxWorker) Status() int {
	return w.status
}
func (w *CommonTxWorker) startWork() {

	w.log.Info("worker startWork is called")
	w.lifecycleMutex.Lock()
	if w.isWorking {
		w.lifecycleMutex.Unlock()
		w.log.Info("worker is working")
		return
	}
	w.isWorking = true
	w.lifecycleMutex.Unlock()
	w.log.Info("worker Start work")

	for {
		w.log.Debug("worker working")

		if w.status == Stop {
			break
		}

		hasmore, err := w.sendNextUnConfirmed()
		if err != nil {
			w.log.Error(err.Error())
		}
		w.log.Info("worker sendNextUnConfirmed ", "hasmore", hasmore)
		if hasmore {
			continue
		} else {
			goto WAIT
		}

	WAIT:
		w.log.Info("worker Start sleep")
		select {
		case <-w.newUnconfirmedTxListener:
			w.log.Info("worker Start awake")
			continue
		case <-w.breaker:
			w.log.Info("worker Start brake")
			break
		}

	}

	w.isWorking = false
	w.log.Info("worker end work")
}

func (w *CommonTxWorker) sendNextUnConfirmed() (hasmore bool, err error) {
	w.log.Info("worker auto send confirm task")
	ac := w.vite.Ledger().Ac()
	hashes, e := ac.GetUnconfirmedTxHashs(0, 1, 1, &w.address)

	if e != nil {
		w.log.Error("worker auto GetUnconfirmedTxHashs", "err", e)
		return false, e
	}

	if len(hashes) == 0 {
		return false, nil
	}

	w.log.Info("worker sendNextUnConfirmed: send receive transaction. ", "hash ", hashes[0])
	err = ac.CreateTx(&ledger.AccountBlock{
		AccountAddress: &w.address,
		FromHash:       hashes[0],
	})

	return true, err
}
