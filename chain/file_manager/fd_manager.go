package chain_file_manager

import (
	"container/list"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fileutils"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

type fileCacheItem struct {
	Buffer []byte

	BufferLen    int64
	FlushPointer int64

	FileId uint64

	Mu sync.RWMutex

	FileWriter *os.File
}

type fdManager struct {
	dirName string
	dirFd   *os.File

	filenamePrefix     string
	filenamePrefixSize int

	fileCache       *list.List
	fileCacheLength int

	fileSize int64
	writeFd  *fileDescription
}

func newFdManager(dirName string, fileSize int, cacheLength int) (*fdManager, error) {
	if cacheLength <= 0 {
		cacheLength = 1
	}
	fdSet := &fdManager{
		dirName:            dirName,
		filenamePrefix:     "f",
		filenamePrefixSize: 1,

		fileCache:       list.New(),
		fileCacheLength: cacheLength,
		fileSize:        int64(fileSize),
	}

	var err error
	fdSet.dirFd, err = fileutils.OpenOrCreateFd(dirName)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("fileutils.OpenOrCreateFd failed, error is %s, dirName is %s", err, dirName))
	}

	location, err := fdSet.loadLatestLocation()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("fdSet.loadLatestFileId failed. Error: %s", err))
	}

	if location == nil {
		location = NewLocation(1, 0)
	}
	if err = fdSet.resetWriteFd(location); err != nil {
		return nil, errors.New(fmt.Sprintf("fdSet.resetWriteFd failed. Error %s", err))
	}

	return fdSet, nil
}

func (fdSet *fdManager) LatestLocation() *Location {
	writeFd := fdSet.GetWriteFd()
	return NewLocation(writeFd.cacheItem.FileId, writeFd.cacheItem.BufferLen)
}

func (fdSet *fdManager) LatestFileId() uint64 {
	return fdSet.writeFd.cacheItem.FileId
}

func (fdSet *fdManager) GetFd(fileId uint64) (*fileDescription, error) {

	if fileId > fdSet.LatestFileId() {
		return nil, nil
	}

	fileCacheItem := fdSet.getCacheItem(fileId)
	if fileCacheItem != nil {
		return NewFdByBuffer(fdSet, fileCacheItem), nil
	}

	fd, err := fdSet.getFileFd(fileId)
	if err != nil {
		return nil, err
	}

	return NewFdByFile(fd), nil
}

func (fdSet *fdManager) GetWriteFd() *fileDescription {
	return fdSet.writeFd
}

func (fdSet *fdManager) DeleteTo(location *Location) error {
	for i := fdSet.LatestFileId(); i > location.FileId; i-- {
		if fdSet.writeFd != nil {
			fdSet.writeFd.cacheItem.FileWriter.Close()
			fdSet.writeFd.Close()

			fdSet.writeFd = nil
		}

		cacheItem := fdSet.fileCache.Back().Value.(*fileCacheItem)
		if cacheItem != nil {
			cacheItem.Mu.Lock()

			cacheItem.FileWriter.Close()
			cacheItem.FileId = 0
			cacheItem.Buffer = nil
			cacheItem.BufferLen = 0
			cacheItem.Mu.Unlock()

			fdSet.fileCache.Remove(fdSet.fileCache.Back())
		}
	}

	// recover write fd
	if err := fdSet.resetWriteFd(location); err != nil {
		return err
	}

	return nil
}

func (fdSet *fdManager) DiskDelete(topLocation *Location, lowLocation *Location) error {
	for i := topLocation.FileId; i > lowLocation.FileId; i-- {
		if err := os.Remove(fdSet.fileIdToAbsoluteFilename(i)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	fd, err := fdSet.getFileFd(lowLocation.FileId)
	defer fd.Close()
	if err != nil {
		return err
	}

	if fd == nil {
		return nil
	}
	if err := fd.Truncate(lowLocation.Offset); err != nil {
		return err
	}
	return nil
}

func (fdSet *fdManager) CreateNextFd() error {
	// new location
	newLocation := NewLocation(fdSet.LatestFileId()+1, 0)

	// set write fd
	fdSet.writeFd = nil

	// update fileReader cache
	if err := fdSet.resetWriteFd(newLocation); err != nil {
		return err
	}

	// set write fd
	return nil

}

func (fdSet *fdManager) RemoveAllFiles() error {
	fdSet.reset()
	if fdSet.writeFd != nil {
		fdSet.writeFd.Close()
		fdSet.writeFd = nil
	}

	if err := os.RemoveAll(fdSet.dirName); err != nil {
		return err
	}

	return nil
}
func (fdSet *fdManager) Close() error {
	fdSet.reset()
	if fdSet.writeFd != nil {
		fdSet.writeFd.Close()
		fdSet.writeFd = nil
	}

	if fdSet.dirFd != nil {
		if err := fdSet.dirFd.Close(); err != nil {
			return err
		}
		fdSet.dirFd = nil
	}

	return nil
}

// tools
func (fdSet *fdManager) resetWriteFd(location *Location) error {
	if fdSet.writeFd != nil {
		return nil
	}
	fileId := location.FileId

	var fd *os.File
	var err error
	if location.Offset > 0 {
		fd, err = fdSet.getFileFd(fileId)
		if err != nil {
			return err
		}

		if fd == nil {
			return errors.New(fmt.Sprintf("fd is nil, fileId is %d, location is %+v\n", fileId, location))

		}
	}

	var newItem *fileCacheItem
	if fdSet.fileCache.Len() >= fdSet.fileCacheLength {
		newItem = fdSet.fileCache.Front().Value.(*fileCacheItem)
		if newItem.FlushPointer >= fdSet.fileSize {
			fdSet.fileCache.MoveToBack(fdSet.fileCache.Front())
		} else {
			newItem = nil
		}
	}

	if newItem == nil {
		newItem = &fileCacheItem{
			Buffer: make([]byte, fdSet.fileSize),
		}
		fdSet.fileCache.PushBack(newItem)
	}

	newItem.Mu.Lock()

	if newItem.FileWriter != nil {
		newItem.FileWriter.Close()
	}

	bufferLen := location.Offset

	newItem.FileWriter = fd
	newItem.FileId = fileId
	newItem.BufferLen = bufferLen
	newItem.FlushPointer = bufferLen

	newItem.Mu.Unlock()
	if bufferLen > 0 && fd != nil {
		if _, err := fd.Read(newItem.Buffer[:bufferLen]); err != nil {
			return err
		}
	}

	fdSet.writeFd = NewWriteFd(fdSet, newItem, fdSet.fileSize)

	return nil
}

func (fdSet *fdManager) loadLatestLocation() (*Location, error) {
	allFilename, readErr := fdSet.dirFd.Readdirnames(0)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("fm.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	maxFileId := uint64(0)
	for _, filename := range allFilename {
		if !fdSet.isCorrectFile(filename) {
			continue
		}

		fileId, err := fdSet.filenameToFileId(filename)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("strconv.ParseUint failed, error is %s, fileName is %s", err.Error(), filename))
		}

		if fileId > maxFileId {
			maxFileId = fileId
		}
	}

	fd, err := fdSet.getFileFd(maxFileId)
	if err != nil {
		return nil, err
	}
	if fd == nil {
		return nil, nil
	}

	fileSize, err := fileutils.FileSize(fd)
	if err != nil {
		return nil, err
	}
	return NewLocation(maxFileId, fileSize), nil
}

func (fdSet *fdManager) reset() {
	if fdSet.writeFd != nil {
		fdSet.writeFd.Close()
	}

	back := fdSet.fileCache.Back()
	for back != nil {
		backValue := back.Value.(*fileCacheItem)

		backValue.Mu.Lock()
		backValue.FileWriter.Close()
		backValue.Mu.Unlock()
		back = back.Prev()
	}
}

func (fdSet *fdManager) getCacheItem(fileId uint64) *fileCacheItem {
	fileCache := fdSet.fileCache
	if fileCache.Len() <= 0 {
		return nil
	}
	front := fileCache.Front()

	if front.Value.(*fileCacheItem).FileId > fileId {
		return nil
	}

	back := fileCache.Back()
	if back.Value.(*fileCacheItem).FileId < fileId {
		return nil
	}
	current := back
	for current != nil {
		cacheItem := current.Value.(*fileCacheItem)
		if cacheItem.FileId == fileId {
			return cacheItem
		}
		current = current.Prev()
	}
	return nil
}

func (fdSet *fdManager) getFileFd(fileId uint64) (*os.File, error) {
	absoluteFilename := fdSet.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			return nil, nil
		}
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}

func (fdSet *fdManager) createNewFile(fileId uint64) (*os.File, error) {
	absoluteFilename := fdSet.fileIdToAbsoluteFilename(fileId)

	file, cErr := os.Create(absoluteFilename)

	if cErr != nil {
		return nil, errors.New("Create fileReader failed, error is " + cErr.Error())
	}

	return file, nil
}

func (fdSet *fdManager) isCorrectFile(filename string) bool {
	return strings.HasPrefix(filename, fdSet.filenamePrefix)
}

func (fdSet *fdManager) fileIdToAbsoluteFilename(fileId uint64) string {
	return path.Join(fdSet.dirName, fdSet.filenamePrefix+strconv.FormatUint(fileId, 10))
}

func (fdSet *fdManager) filenameToFileId(filename string) (uint64, error) {
	fileIdStr := filename[fdSet.filenamePrefixSize:]
	return strconv.ParseUint(fileIdStr, 10, 64)

}
