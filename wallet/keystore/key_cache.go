package keystore

import (
	"bufio"
	"encoding/json"
	"github.com/deckarep/golang-set"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"os"
	"sync"
	"time"
)

type keyCache struct {
	keydir          string
	kob             *keystoreObserver
	mutex           sync.Mutex
	throttle        *time.Timer
	notify          chan struct{}
	fileC           fileutils.FileChangeRecord
	cacheAddr       mapset.Set
	lasttryloadtime time.Time
}

func newKeyCache(keydir string) (*keyCache, chan struct{}) {
	kc := &keyCache{
		keydir: keydir,
		notify: make(chan struct{}, 1),
		fileC: fileutils.FileChangeRecord{AllCached: mapset.NewThreadUnsafeSet(), FileFilter: func(filepath string) bool {
			if _, err := addressFromKeyPath(filepath); err != nil {
				log.Info("Not address file path", filepath)
				return true
			}
			return false
		}},
		cacheAddr: mapset.NewThreadUnsafeSet(),
	}
	kc.kob = newObserver(kc)
	return kc, kc.notify
}

func (kc *keyCache) ListAllAddress() mapset.Set {
	kc.intervalRefresh()
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	return kc.cacheAddr.Clone()
}

func (kc *keyCache) refresh() error {
	creates, deletes, updates, err := kc.fileC.RefreshCache(kc.keydir)
	if err != nil {
		log.Info("Failed refreshCache keydir", "err", err)
		return err
	}

	if creates.Cardinality() == 0 && deletes.Cardinality() == 0 && updates.Cardinality() == 0 {
		log.Info("Nothing Changed")
		return nil
	}

	for _, c := range creates.ToSlice() {
		if a := readAddressFromFile(c.(string)); a != nil {
			kc.add(*a)
		}
	}
	for _, d := range deletes.ToSlice() {
		kc.deleteByFile(d.(string))
	}
	for _, u := range updates.ToSlice() {
		kc.deleteByFile(u.(string))
		if a := readAddressFromFile(u.(string)); a != nil {
			kc.add(*a)
		}
	}

	return nil
}
func (kc *keyCache) add(addr types.Address) {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()
	kc.cacheAddr.Add(addr)

}
func (kc *keyCache) deleteByFile(fullfilename string) {
	a, err := addressFromKeyPath(fullfilename)
	if err == nil {
		return
	}
	kc.mutex.Lock()
	defer kc.mutex.Unlock()
	kc.cacheAddr.Remove(a)
}

func readAddressFromFile(path string) *types.Address {
	buf := new(bufio.Reader)
	key := encryptedKeyJSON{}

	fd, err := os.Open(path)
	if err != nil {
		log.Trace("Can not to open ", "path", path, "err", err)
		return nil
	}
	defer fd.Close()
	buf.Reset(fd)
	key.HexAddress = ""
	err = json.NewDecoder(buf).Decode(&key)
	if err != nil {
		log.Trace("Decode keystore file failed ", "path", path, "err", err)
		return nil
	}
	addr, err := types.HexToAddress(key.HexAddress)
	if err != nil {
		log.Trace("Address is invalid ", "path", path, "err", err)
		return nil
	}
	return &addr

}

// min reload time is 2s that means if
func (kc *keyCache) intervalRefresh() {
	kc.mutex.Lock()

	if kc.kob.running {
		kc.mutex.Unlock()
		return // A watcher is running and will keep the cache up-to-date.
	}

	if kc.throttle == nil {
		kc.throttle = time.NewTimer(0)
	} else {
		select {
		case <-kc.throttle.C:
		default:
			kc.mutex.Unlock()
			return
		}
	}
	kc.kob.start()
	kc.throttle.Reset(2 * time.Second)
	kc.mutex.Unlock()
	kc.refresh()
}

func (kc *keyCache) close() {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	kc.kob.close()
	if kc.throttle != nil {
		kc.throttle.Stop()
	}
	if kc.notify != nil {
		close(kc.notify)
		kc.notify = nil
	}

}
