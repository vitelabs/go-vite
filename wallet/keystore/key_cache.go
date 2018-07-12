package keystore

import (
	"github.com/deckarep/golang-set"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"os"
	"strings"
	"sync"
)

// Every two seconds, it will check if there is a file in the keydir changed
type keyCache struct {
	keydir          string
	kob             *keystoreObserver
	mutex           sync.Mutex
	changed         chan struct{}
	fileC           fileutils.FileChangeRecord
	cacheAddr       mapset.Set
}

func newKeyCache(keydir string) (*keyCache, chan struct{}) {
	kc := &keyCache{
		keydir:  keydir,
		changed: make(chan struct{}, 1),
		fileC: fileutils.NewFileChangeRecord(func(dir string, file os.FileInfo) bool {
			if file.IsDir() || file.Mode()&os.ModeType != 0 {
				return true
			}
			fn := file.Name()
			if strings.HasPrefix(fn, ".") || strings.HasSuffix(fn, "~") {
				return true
			}
			return false
		}),
		cacheAddr: mapset.NewThreadUnsafeSet(),
	}
	kc.kob = newObserver(kc)
	return kc, kc.changed
}

func (kc *keyCache) ListAllAddress() mapset.Set {
	kc.maybeReload()
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	return kc.cacheAddr.Clone()
}

func (kc *keyCache) refreshAndFixAddressFile() error {
	log.Debug("refreshAndFixAddressFile")
	creates, deletes, updates, err := kc.fileC.RefreshCache(kc.keydir)
	if err != nil {
		log.Debug("Failed refreshCache keydir", "err", err)
		return err
	}

	if creates.Cardinality() == 0 && deletes.Cardinality() == 0 && updates.Cardinality() == 0 {
		log.Debug("Nothing Changed")
		return nil
	}

	creates.Each(func(c interface{}) bool {
		log.Debug("creates ", c)
		if a, _ := readAndFixAddressFile(c.(string)); a != nil {
			log.Debug("Get new address", a.Hex())
			kc.add(*a)
		}
		return false
	})
	deletes.Each(func(c interface{}) bool {
		log.Debug("delete ", c)
		kc.deleteByFile(c.(string))
		return false
	})
	updates.Each(func(c interface{}) bool {
		log.Debug("updates ", c)
		kc.deleteByFile(c.(string))
		if a, _ := readAndFixAddressFile(c.(string)); a != nil {
			log.Debug("update address", a.Hex())
			kc.add(*a)
		}
		return false
	})

	//kc.changed <- struct{}{}
	return nil
}
func (kc *keyCache) add(addr types.Address) {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()
	kc.cacheAddr.Add(addr)

}
func (kc *keyCache) deleteByFile(fullfilename string) {
	a, err := addressFromKeyPath(fullfilename)
	if err != nil {
		return
	}
	kc.mutex.Lock()
	defer kc.mutex.Unlock()
	kc.cacheAddr.Remove(a)
}

// This may not reloadAndFixAddressFile data from disk
func (kc *keyCache) maybeReload() {
	kc.mutex.Lock()
	if kc.kob.running {
		kc.mutex.Unlock()
		return // key observer is running means that cache is almost new
	}
	kc.kob.start()
	kc.mutex.Unlock()

	kc.refreshAndFixAddressFile()
}

func (kc *keyCache) close() {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	kc.kob.close()

	if kc.changed != nil {
		close(kc.changed)
		kc.changed = nil
	}

}
