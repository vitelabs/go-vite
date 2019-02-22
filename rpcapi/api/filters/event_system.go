package filters

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
)

type AccountChainEvent struct {
	AccountBlock *ledger.AccountBlock
	Logs         []*ledger.VmLog
}
type AccountChainDelEvent struct {
}

type SnapshotChainEvent struct {
	SnapshotBlock *ledger.SnapshotBlock
}
type SnapshotChainDelEvent struct {
}

type SubscribeInterface interface {
	// TODO
	GetLogs(addr types.Address, startSnapshotHeight uint64, endSnapshotHeight uint64) ([]*ledger.VmLog, error)
	SubscribeLogs()
}

type subscription struct{}

type EventSystem struct {
	vite      *vite.Vite
	install   chan *subscription         // install filter
	uninstall chan *subscription         // remove filter
	acCh      chan AccountChainEvent     // Channel to receive new account chain event
	spCh      chan SnapshotChainEvent    // Channel to receive new snapshot chain event
	acDelCh   chan AccountChainDelEvent  // Channel to receive new account chain delete event when account chain fork
	spDelCh   chan SnapshotChainDelEvent // Channel to receive new snapshot chain delete event when snapshot chain fork
	// TODO subscriptions
}

func NewEventSystem(v *vite.Vite) *EventSystem {
	es := &EventSystem{vite: v}
	// TODO subscribe default events

	go es.eventLoop()
	return es
}

func (es *EventSystem) eventLoop() {
	for {
		select {
		case acEvent := <-es.acCh:
			es.handleAcEvent(&acEvent)
		case spEvent := <-es.spCh:
			es.handleSpEvent(&spEvent)
		case acDelEvent := <-es.acDelCh:
			es.handleAcDelEvent(&acDelEvent)
		case spDelEvent := <-es.spDelCh:
			es.handleSpDelEvent(&spDelEvent)
		case i := <-es.install:
			// TODO
			fmt.Println(i)
		case u := <-es.uninstall:
			// TODO
			fmt.Println(u)
		}
	}
}

func (es *EventSystem) handleAcEvent(acEvent *AccountChainEvent) {
	// TODO
}
func (es *EventSystem) handleSpEvent(acEvent *SnapshotChainEvent) {
	// TODO
}
func (es *EventSystem) handleAcDelEvent(acEvent *AccountChainDelEvent) {
	// TODO
}
func (es *EventSystem) handleSpDelEvent(acEvent *SnapshotChainDelEvent) {
	// TODO
}
