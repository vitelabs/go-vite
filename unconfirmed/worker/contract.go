package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type ContractWorker struct {
	vite    Vite
	address types.Address
	gid     string
	log     log15.Logger

	status     int
	isSleeping bool

	breaker             chan struct{}
	newContractListener chan struct{}
	stopListener        chan struct{} // make sure we can sync stop the worker

	statusMutex sync.Mutex
}

func NewContractWorker(vite Vite, address types.Address, gid string) *ContractWorker {
	return &ContractWorker{
		vite:    nil,
		address: types.Address{},
		log:     log15.New("ContractWorker addr", address.String(), "gid", gid),
		status:  Create,
	}
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) Start() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.status = Start

}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.status = Stop
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}
