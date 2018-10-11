package onroad_test

import (
	"testing"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/common/types"
	"time"
	"fmt"
)

func TestContractWorker_Start(t *testing.T) {
	manager, addresses := startManager()
	worker := onroad.NewContractWorker(manager, producerevent.AccountStartEvent{
		Gid:            types.DELEGATE_GID,
		Address:        addresses,
		Stime:          time.Now(),
		Etime:          time.Now().Add(time.Minute),
		Timestamp:      time.Now(),
		SnapshotHash:   types.ZERO_HASH,
		SnapshotHeight: 0,
	})

	worker.Start()

	time.AfterFunc(10*time.Second, func() {
		fmt.Println("NewOnroadTxAlarm 1")
		worker.NewOnroadTxAlarm()
		time.AfterFunc(10*time.Second, func() {
			fmt.Println("NewOnroadTxAlarm 2")
			worker.NewOnroadTxAlarm()
		})
	})

	time.Sleep(5 * time.Minute)
}
