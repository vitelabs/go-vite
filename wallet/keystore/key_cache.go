package keystore

import (
	"bufio"
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"gopkg.in/fatih/set.v0"
	"os"
	"sync"
	"time"
)

type keyCache struct {
	keydir   string
	kob      *keystoreObserver
	mutex    sync.Mutex
	throttle *time.Timer
	notify   chan struct{}
	fileC    fileCache
}

func newKeyCache(keydir string) (*keyCache, chan struct{}) {
	kc := &keyCache{
		keydir: keydir,
		notify: make(chan struct{}, 1),
		fileC:  fileCache{all: set.NewNonTS()},
	}
	kc.kob = newObserver(kc)
	return kc, kc.notify
}

func (kc *keyCache) refresh() error {
	creates, deletes, updates, err := kc.fileC.scan(kc.keydir)
	if err != nil {
		log.Info("Failed scan keydir", "err", err)
		return err
	}

	if creates.Size() == 0 && deletes.Size() == 0 && updates.Size() == 0 {
		log.Debug("Nothing Changed")
		return nil
	}

	buf := new(bufio.Reader)
	key := encryptedKeyJSON{}

	readAddressFunc := func(path string) *types.Address {
		fd, err := os.Open(path)
		if err != nil {
			log.Trace("can not to open ", "path", path, "err", err)
			return nil
		}
		defer fd.Close()
		buf.Reset(fd)
		key.HexAddress = ""
		err = json.NewDecoder(buf).Decode(&key)
		if err != nil {
			log.Trace("decode keystore file failed ", "path", path, "err", err)
			return nil
		}
		addr, err := types.HexToAddress(key.HexAddress)
		if err != nil {
			log.Trace("Address is invalid ", "path", path, "err", err)
			return nil
		}
		return &addr

	}

	for _, p := range creates.List() {
		if a := readAddressFunc(p.(string)); a != nil {
			kc.add(*a)
		}
	}
	for _, p := range deletes.List() {
		kc.deleteByFile(p.(string))
	}
	for _, p := range updates.List() {
		path := p.(string)
		kc.deleteByFile(path)
		if a := readAddressFunc(path); a != nil {
			kc.add(*a)
		}
	}

	return nil
}
func (cache *keyCache) add(addresses types.Address) {

}
func (cache *keyCache) deleteByFile(s string) {

}
