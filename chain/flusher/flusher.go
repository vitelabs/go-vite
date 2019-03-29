package chain_flusher

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sync"
	"time"
)

type Storage interface {
	Prepare()

	CancelPrepare()

	RedoLog() []byte

	Commit() error

	PatchRedoLog([]byte) error
}

type Flusher struct {
	dirName   string
	storeList []Storage
	log       log15.Logger
	fd        *os.File

	mu      *sync.RWMutex
	wg      sync.WaitGroup
	stopped chan struct{}

	flushInterval time.Duration
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

	flusher := &Flusher{
		dirName:       path.Join(chainDir, "flusher"),
		storeList:     storeList,
		log:           log15.New("module", "flusher"),
		fd:            fd,
		mu:            mu,
		stopped:       make(chan struct{}),
		flushInterval: time.Second,
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

		binary.BigEndian.PutUint32(redoLogLengthBytes, redoLogLength)

		if _, err := flusher.fd.Write(redoLogLengthBytes); err != nil {
			flusher.log.Error(fmt.Sprintf("write failed, error is %s", err.Error()), "method", "Flush")
			return
		}

		if _, err := flusher.fd.Write(redoLog); err != nil {
			flusher.log.Error(fmt.Sprintf("write failed, error is %s", err.Error()), "method", "Flush")
			return
		}
	}

	// sync redo log
	if err := flusher.fd.Sync(); err != nil {
		for _, store := range flusher.storeList {
			// cancel prepare
			store.CancelPrepare()
		}

		// clean redo log failed
		if err := flusher.fd.Truncate(0); err != nil {
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

	redoLogList, err := flusher.loadRedoLog()
	if err != nil {
		panic(err)
	}

	return flusher.redo(redoLogList)

}

func (flusher *Flusher) Recover() error {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	redoLogList, err := flusher.loadRedoLog()
	if err != nil || len(redoLogList) <= 0 {
		flusher.cleanRedoLog()
		return nil
	}
	return flusher.redo(redoLogList)

}

func (flusher *Flusher) Start() {
	flusher.mu.Lock()
	defer flusher.mu.Unlock()

	flusher.wg.Add(1)
	go func() {
		defer flusher.wg.Done()
		for {
			select {
			case <-flusher.stopped:
				flusher.Flush()
				return
			default:
				flusher.Flush()
				time.Sleep(flusher.flushInterval)
			}
		}

	}()
}

func (flusher *Flusher) Stop() {

	close(flusher.stopped)
	flusher.wg.Wait()
}

func (flusher *Flusher) loadRedoLog() ([][]byte, error) {
	fileSize, err := fileutils.FileSize(flusher.fd)
	if err != nil {
		return nil, err
	}
	if fileSize <= 0 {
		return nil, nil
	}

	buf := make([]byte, fileSize)
	if _, err := flusher.fd.Read(buf); err != nil {
		return nil, err
	}

	redoLogList := make([][]byte, 0, len(flusher.storeList))
	currentPointer := uint32(0)
	nextPointer := uint32(0)
	endPointer := uint32(fileSize)

	for nextPointer < endPointer {
		nextPointer = currentPointer + 4
		buffer := buf[currentPointer:nextPointer]
		if len(buffer) < 4 {
			return nil, errors.New("read redo log failed")
		}
		size := binary.BigEndian.Uint32(buffer)

		currentPointer = nextPointer
		nextPointer = currentPointer + size

		buffer = buf[currentPointer:nextPointer]
		if uint32(len(buffer)) < size {
			return nil, errors.New("read redo log failed")
		}

		redoLogList = append(redoLogList, buffer)

		currentPointer = nextPointer
	}
	return redoLogList, nil
}

func (flusher *Flusher) cleanRedoLog() error {
	err := flusher.fd.Truncate(0)
	if err != nil {
		flusher.log.Error(fmt.Sprintf("clean flusher file failed. Error: %s", err.Error()), "method", "Flush")
	}
	return err
}

func (flusher *Flusher) redo(redoLogList [][]byte) error {
	for index, redoLog := range redoLogList {
		if err := flusher.storeList[index].PatchRedoLog(redoLog); err != nil {
			return err
		}
	}

	// clean flush log
	flusher.cleanRedoLog()
	return nil
}
