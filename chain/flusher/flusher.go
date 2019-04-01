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
)

type Storage interface {
	Id() types.Hash
	Prepare()

	CancelPrepare()

	RedoLog() []byte

	Commit() error

	PatchRedoLog([]byte) error
}

type Flusher struct {
	dirName   string
	storeList []Storage
	idMap     map[types.Hash]Storage

	log log15.Logger
	fd  *os.File

	mu *sync.RWMutex
	//wg sync.WaitGroup
	//stopped chan struct{}

	//flushInterval   time.Duration
	startCommitFlag types.Hash
}

func NewFlusher(mu *sync.RWMutex, storeList []Storage, chainDir string) (*Flusher, error) {
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
		idMap:     idMap,

		log: log15.New("module", "flusher"),
		fd:  fd,
		mu:  mu,
		//stopped:         make(chan struct{}),
		//flushInterval:   time.Second,
		startCommitFlag: startCommitFlag,
	}

	return flusher, nil
}

func (flusher *Flusher) Flush() {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	// prepare
	for _, store := range flusher.storeList {
		store.Prepare()
	}

	// write redo (sum)
	if err := flusher.cleanRedoLog(); err != nil {
		return
	}

	redoLogLengthBytes := make([]byte, 4)

	for _, store := range flusher.storeList {

		redoLog := store.RedoLog()
		redoLogLength := uint32(len(redoLog))

		if redoLogLength <= 0 {
			continue
		}

		id := store.Id()
		if _, err := flusher.fd.Write(id.Bytes()); err != nil {
			flusher.log.Error(fmt.Sprintf("write failed. Error: %s", err.Error()), "method", "Flush")
			return
		}

		binary.BigEndian.PutUint32(redoLogLengthBytes, redoLogLength)
		if _, err := flusher.fd.Write(redoLogLengthBytes); err != nil {
			flusher.log.Error(fmt.Sprintf("write failed. Error: %s", err.Error()), "method", "Flush")
			return
		}

		if _, err := flusher.fd.Write(redoLog); err != nil {
			flusher.log.Error(fmt.Sprintf("write failed. Error: %s", err.Error()), "method", "Flush")
			return
		}
	}

	if _, err := flusher.fd.Write(flusher.startCommitFlag.Bytes()); err != nil {
		flusher.log.Error(fmt.Sprintf("write failed. Error: %s", err.Error()), "method", "Flush")
		return
	}

	// sync redo log
	if err := flusher.fd.Sync(); err != nil {
		for _, store := range flusher.storeList {
			// cancel prepare
			store.CancelPrepare()
		}

		// clean redo log failed
		if err := flusher.cleanRedoLog(); err != nil {
			flusher.log.Crit(err.Error(), "method", "Flush")
		}

		flusher.log.Error(fmt.Sprintf("sync failed. Error: %s", err.Error()), "method", "Flush")
		return
	}

	// commit
	for _, store := range flusher.storeList {
		if err := store.Commit(); err != nil {

			if err := flusher.Redo(); err != nil {
				panic(err)
			}

			flusher.log.Error(fmt.Sprintf("commit failed. Error: %s", err.Error()), "method", "Flush")
		}

	}

	// clean flush log
	flusher.cleanRedoLog()
}

func (flusher *Flusher) Redo() error {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	redoLogList, stores, err := flusher.loadRedo()
	if err != nil {
		panic(err)
	}

	return flusher.redo(stores, redoLogList)

}

func (flusher *Flusher) Recover() error {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	redoLogList, stores, err := flusher.loadRedo()
	if err != nil || len(redoLogList) <= 0 {
		flusher.cleanRedoLog()
		return nil
	}
	return flusher.redo(stores, redoLogList)

}

//func (flusher *Flusher) Start() {
//	flusher.mu.Lock()
//	defer flusher.mu.Unlock()
//
//	flusher.wg.Add(1)
//	go func() {
//		defer flusher.wg.Done()
//		for {
//			select {
//			case <-flusher.stopped:
//				flusher.Flush()
//				return
//			default:
//				time.Sleep(flusher.flushInterval)
//				flusher.Flush()
//			}
//		}
//
//	}()
//}
//
//func (flusher *Flusher) Stop() {
//
//	close(flusher.stopped)
//	flusher.wg.Wait()
//}

func (flusher *Flusher) loadRedo() ([][]byte, []Storage, error) {
	fileSize, err := fileutils.FileSize(flusher.fd)
	if err != nil {
		return nil, nil, err
	}
	if fileSize <= 0 {
		return nil, nil, nil
	}

	buf := make([]byte, fileSize)
	if _, err := flusher.fd.Read(buf); err != nil {
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
			buffer = buf[currentPointer:nextPointer]
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
