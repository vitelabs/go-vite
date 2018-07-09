package keystore

import (
	"time"
)

const scanInterval = 3000 * time.Millisecond

// every scanInterval it will read the files which in keydir
type keystoreObserver struct {
	kc       *keyCache
	starting bool
	running  bool
	exit     chan struct{}
}

func newObserver(kc *keyCache) *keystoreObserver {
	return &keystoreObserver{
		kc:   kc,
		exit: make(chan struct{}),
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

	w.kc.mutex.Lock()
	w.running = true
	w.kc.mutex.Unlock()

	t := time.NewTicker(scanInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			w.kc.refreshAndFixAddressFile()

		case <-w.exit:
			return
		}
	}
}
