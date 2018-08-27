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

	status     int
	isSleeping bool

	breaker                  chan struct{}
	newUnconfirmedTxListener chan struct{}
	stopListener             chan struct{} // make sure we can sync stop the worker

	statusMutex sync.Mutex
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

func (w *CommonTxWorker) Start() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {
		// 0. init the break chan
		w.breaker = make(chan struct{}, 1)

		// 1. listen unconfirmed tx in ledger
		w.newUnconfirmedTxListener = make(chan struct{}, 100)
		w.vite.Ledger().Ac().AddListener(w.address, w.newUnconfirmedTxListener)

		w.stopListener = make(chan struct{})

		w.status = Start
		w.statusMutex.Unlock()
		go w.startWork()
	} else {
		if w.isSleeping {
			// awake it to run at least once
			w.newUnconfirmedTxListener <- struct{}{}
		}
	}
}

func (w *CommonTxWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {
		w.breaker <- struct{}{}
		close(w.breaker)

		w.vite.Ledger().Ac().RemoveListener(w.address)
		close(w.newUnconfirmedTxListener)

		// make sure we can stop the worker
		<-w.stopListener
		close(w.stopListener)

		w.status = Stop
	}
}

func (w CommonTxWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *CommonTxWorker) startWork() {

	w.log.Info("worker startWork is called")
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
		w.isSleeping = true
		w.log.Info("worker Start sleep")
		select {
		case <-w.newUnconfirmedTxListener:
			w.log.Info("worker Start awake")
			continue
		case <-w.breaker:
			w.log.Info("worker broken")
			break
		}
	}

	w.log.Info("worker send stopListener ")
	w.stopListener <- struct{}{}
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
