package keystore

import (
	"github.com/rjeczalik/notify"
	"github.com/vitelabs/go-vite/log"
	"time"
)

const scanInterval = 567 * time.Millisecond

type keystoreObserver struct {
	kc       *keyCache
	starting bool
	running  bool
	event    chan notify.EventInfo
	exit     chan struct{}
}

func newObserver(kc *keyCache) *keystoreObserver {
	return &keystoreObserver{
		kc:    kc,
		event: make(chan notify.EventInfo, 10),
		exit:  make(chan struct{}),
	}
}

func (w *keystoreObserver) start() {
	if w.starting || w.running {
		return
	}
	w.starting = true
	go w.loop()
}

func (w *keystoreObserver) close() {
	close(w.exit)
}

func (w *keystoreObserver) loop() {
	defer func() {
		w.kc.mutex.Lock()
		w.running = false
		w.starting = false
		w.kc.mutex.Unlock()
	}()

	if err := notify.Watch(w.kc.keydir, w.event, notify.All); err != nil {
		log.Trace("Failed to watch keystore folder", "err", err)
		return
	}
	defer notify.Stop(w.event)
	log.Info("Started watching keystore folder")
	defer log.Info("Stopped watching keystore folder")

	w.kc.mutex.Lock()
	w.running = true
	w.kc.mutex.Unlock()

	refreshTriggered := false
	t := time.NewTimer(0)

	if !t.Stop() {
		<-t.C
	}
	defer t.Stop()
	for {
		select {
		case <-w.event:
			if !refreshTriggered {
				t.Reset(scanInterval)
				refreshTriggered = true
			}

		case <-t.C:
			w.kc.refreshAndFixAddressFile()
			refreshTriggered = false

		case <-w.exit:
			return

		}
	}
}
