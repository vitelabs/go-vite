package chain_flusher

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sync"
	"time"
)

type Storage interface {
	Id() types.Hash

	Prepare()

	CancelPrepare()

	RedoLog() ([]byte, error)

	Commit() error

	AfterCommit()

	BeforeRecover([]byte)

	AfterRecover()

	PatchRedoLog([]byte) error
}

type Flusher struct {
	dirName   string
	storeList []Storage
	idMap     map[types.Hash]Storage

	log log15.Logger
	fd  *os.File

	mu *sync.RWMutex

	syncFlush sync.WaitGroup
	wg        sync.WaitGroup

	flushInterval time.Duration
	flushingMu    sync.Mutex

	startCommitFlag types.Hash
	commitWg        sync.WaitGroup

	terminal chan struct{}
}

func NewFlusher(storeList []Storage, flushMu *sync.RWMutex, chainDir string) (*Flusher, error) {
	fileName := path.Join(chainDir, "flush.redo.log")
	fd, oErr := os.OpenFile(fileName, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			var err error
			fd, err = os.Create(fileName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New(fmt.Sprintf("open file %s failed. Error: %s", fileName, oErr.Error()))
		}
	}

	idMap := make(map[types.Hash]Storage, len(storeList))
	for _, store := range storeList {
		idMap[store.Id()] = store
	}

	startCommitFlag, _ := types.BytesToHash(crypto.Hash256([]byte("start commit")))

	flusher := &Flusher{
		dirName:   path.Join(chainDir, "flusher"),
		storeList: storeList,
		mu:        flushMu,
		idMap:     idMap,

		log: log15.New("module", "flusher"),
		fd:  fd,

		flushInterval: 900 * time.Millisecond,

		startCommitFlag: startCommitFlag,
	}

	return flusher, nil
}
func (flusher *Flusher) Close() error {
	if err := flusher.fd.Close(); err != nil {
		return err
	}
	return nil
}

func (flusher *Flusher) ReplaceStore(id types.Hash, store Storage) {

	flusher.flushingMu.Lock()
	defer flusher.flushingMu.Unlock()
	flusher.idMap[id] = store
	for index, istore := range flusher.storeList {
		if istore.Id() == id {
			flusher.storeList[index] = store
			break
		}
	}
}

func (flusher *Flusher) Start() {
	flusher.terminal = make(chan struct{})
	flusher.loopFlush()
}

func (flusher *Flusher) Stop() {
	close(flusher.terminal)
	flusher.wg.Wait()
}

// force to flush synchronously
func (flusher *Flusher) Flush() {
	flusher.flush()
}

func (flusher *Flusher) Recover() error {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	redoLogList, stores, err := flusher.loadRedo()
	if err != nil || len(redoLogList) <= 0 {
		flusher.cleanRedoLog()
		return nil
	}

	flusher.beforeRecover(stores, redoLogList)
	if err := flusher.redo(stores, redoLogList); err != nil {
		return err
	}
	flusher.afterRecover()
	return nil
}

func (flusher *Flusher) loopFlush() {
	flusher.wg.Add(1)
	go func() {
		defer flusher.wg.Done()

		for {
			select {
			case <-flusher.terminal:
				flusher.flush()
				return

			default:
				flusher.flush()
				time.Sleep(flusher.flushInterval)

			}
		}
	}()
}

func (flusher *Flusher) flush() {
	flusher.flushingMu.Lock()
	defer flusher.flushingMu.Unlock()

	// prepare, lock write
	flusher.log.Info("start prepare")
	flusher.prepare()
	flusher.log.Info("prepare finish")

	// write redo log
	flusher.log.Info("start write redo log")
	if err := flusher.writeRedoLog(); err != nil {
		return
	}
	flusher.log.Info("finish writing redo log")

	// sync redo log
	// if sync redo log failed, stop flushing and cancel prepare
	// lock write when cancel prepare

	flusher.log.Info("sync write redo log")
	if !flusher.syncRedoLog() {
		return
	}
	flusher.log.Info("finish sync writing redo log")

	// commit
	flusher.log.Info("start commit")
	err := flusher.commit()
	flusher.log.Info("finish committing")

	// after commit, lock write
	flusher.log.Info("after commit")
	flusher.afterCommit()
	flusher.log.Info("finish after committing")

	// redo
	if err != nil {
		flusher.log.Info("commit redo")
		if err := flusher.commitRedo(); err != nil {
			panic(err)
		}
		flusher.log.Info("finish committing redo")
	}

	// clean redo log
	flusher.cleanRedoLog()
}

func (flusher *Flusher) commitRedo() error {

	redoLogList, stores, err := flusher.loadRedo()
	if err != nil {
		panic(err)
	}

	return flusher.redo(stores, redoLogList)

}

func (flusher *Flusher) loadRedo() ([][]byte, []Storage, error) {
	fileSize, err := fileutils.FileSize(flusher.fd)
	if err != nil {
		return nil, nil, err
	}
	if fileSize <= 0 {
		return nil, nil, nil
	}

	buf := make([]byte, fileSize)
	if _, err := flusher.fd.ReadAt(buf, 0); err != nil {
		return nil, nil, err
	}

	currentPointer := 0
	nextPointer := 0
	endPointer := int(fileSize)

	status := 0
	size := types.HashSize

	redoLogList := make([][]byte, 0, len(flusher.storeList))
	stores := make([]Storage, 0, len(flusher.storeList))

	hasStartCommitFlag := false
	for {
		nextPointer = currentPointer + size
		if nextPointer > endPointer {
			break
		}
		buffer := buf[currentPointer:nextPointer]
		if len(buffer) < size {
			return nil, nil, errors.New("read redo log failed")
		}

		switch status {
		case 0:
			id, err := types.BytesToHash(buffer)
			if err != nil {
				return nil, nil, err
			}

			if id == flusher.startCommitFlag {
				hasStartCommitFlag = true
				break
			}

			store, ok := flusher.idMap[id]
			if !ok {
				return nil, nil, errors.New(fmt.Sprintf("id is not existed, id: %s", id))
			}

			stores = append(stores, store)
			status = 1
			size = 4

		case 1:
			status = 2
			size = int(binary.BigEndian.Uint32(buffer))

		case 2:
			redoLogList = append(redoLogList, buffer)
			status = 0
			size = types.HashSize
		}

		currentPointer = nextPointer
	}

	if !hasStartCommitFlag {
		return nil, nil, nil
	}

	return redoLogList, stores, nil
}

func (flusher *Flusher) cleanRedoLog() error {
	err := flusher.fd.Truncate(0)
	if err != nil {
		flusher.log.Error(fmt.Sprintf("truncate file failed. Error: %s", err.Error()), "method", "cleanRedoLog")
	}

	_, err = flusher.fd.Seek(0, 0)
	if err != nil {
		flusher.log.Error(fmt.Sprintf("seek file failed. Error: %s", err.Error()), "method", "cleanRedoLog")
	}

	return err
}

func (flusher *Flusher) prepare() {
	// prepare, lock write
	flusher.mu.Lock()

	for _, store := range flusher.storeList {
		store.Prepare()
	}

	flusher.mu.Unlock()
}

func (flusher *Flusher) writeRedoLog() error {
	// write redo (sum)
	if err := flusher.cleanRedoLog(); err != nil {
		return err
	}

	redoLogLengthBytes := make([]byte, 4)

	for _, store := range flusher.storeList {

		redoLog, err := store.RedoLog()
		if err != nil {
			fErr := errors.New(fmt.Sprintf("store.Buf failed. Error: %s", err.Error()))
			flusher.log.Error(fErr.Error(), "method", "Flush")
			return fErr
		}

		redoLogLength := uint32(len(redoLog))

		if redoLogLength <= 0 {
			continue
		}

		id := store.Id()
		if _, err := flusher.fd.Write(id.Bytes()); err != nil {
			fErr := errors.New(fmt.Sprintf("write failed. Error: %s", err.Error()))

			flusher.log.Error(fErr.Error(), "method", "Flush")
			return fErr
		}

		binary.BigEndian.PutUint32(redoLogLengthBytes, redoLogLength)
		if _, err := flusher.fd.Write(redoLogLengthBytes); err != nil {
			fErr := errors.New(fmt.Sprintf("write failed. Error: %s", err.Error()))

			flusher.log.Error(fErr.Error(), "method", "Flush")
			return fErr
		}

		if _, err := flusher.fd.Write(redoLog); err != nil {
			fErr := errors.New(fmt.Sprintf("write failed. Error: %s", err.Error()))

			flusher.log.Error(fErr.Error(), "method", "Flush")
			return fErr
		}
	}

	if _, err := flusher.fd.Write(flusher.startCommitFlag.Bytes()); err != nil {
		fErr := errors.New(fmt.Sprintf("write failed. Error: %s", err.Error()))
		flusher.log.Error(fErr.Error(), "method", "Flush")
		return fErr
	}
	return nil

}

func (flusher *Flusher) syncRedoLog() bool {
	err := flusher.fd.Sync()

	if err == nil {
		return true
	}

	flusher.log.Error(fmt.Sprintf("sync failed. Error: %s", err.Error()), "method", "Flush")

	// cancel prepare, lock write
	flusher.mu.Lock()

	for _, store := range flusher.storeList {
		// cancel prepare
		store.CancelPrepare()
	}

	flusher.mu.Unlock()

	// clean redo log failed
	if err := flusher.cleanRedoLog(); err != nil {
		flusher.log.Error(err.Error(), "method", "Flush")
	}

	return false
}

func (flusher *Flusher) commit() error {
	var commitErr error
	flusher.commitWg.Add(len(flusher.storeList))
	for _, store := range flusher.storeList {
		commitStore := store
		go func() {
			defer flusher.commitWg.Done()
			if err := commitStore.Commit(); err != nil {
				commitErr = err
				flusher.log.Error(fmt.Sprintf("%s commit failed. Error: %s", commitStore.Id(), err.Error()), "method", "Flush")
				return
			}
		}()
	}

	flusher.commitWg.Wait()

	return commitErr
}

func (flusher *Flusher) afterCommit() {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	for _, store := range flusher.storeList {
		store.AfterCommit()
	}

}

func (flusher *Flusher) beforeRecover(stores []Storage, redoLogList [][]byte) {
	for index, redoLog := range redoLogList {
		stores[index].BeforeRecover(redoLog)
	}
}

func (flusher *Flusher) afterRecover() {
	for _, store := range flusher.storeList {
		store.AfterRecover()
	}
}

func (flusher *Flusher) redo(stores []Storage, redoLogList [][]byte) error {
	for index, redoLog := range redoLogList {
		if err := stores[index].PatchRedoLog(redoLog); err != nil {
			return err
		}
	}

	// clean flush log
	flusher.cleanRedoLog()
	return nil
}
