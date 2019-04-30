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

	BufferLen int64

	FileId uint64

	Mu sync.RWMutex

	FileWriter *os.File
}

type fdManager struct {
	dirName string
	dirFd   *os.File

	filenamePrefix     string
	filenamePrefixSize int

	fileCache   *list.List
	fileFdCache map[uint64]*fileDescription

	fileCacheLength int

	fileSize int64
	writeFd  *fileDescription

	changeFdMu sync.RWMutex

	fileManager *FileManager
}

func newFdManager(fileManager *FileManager, dirName string, fileSize int, cacheLength int) (*fdManager, error) {
	if cacheLength <= 0 {
		cacheLength = 1
	}
	fdSet := &fdManager{
		dirName:            dirName,
		fileManager:        fileManager,
		filenamePrefix:     "f",
		filenamePrefixSize: 1,

		fileCache:       list.New(),
		fileCacheLength: cacheLength,

		fileFdCache: make(map[uint64]*fileDescription, cacheLength),
		fileSize:    int64(fileSize),
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

func (fdSet *fdManager) GetFd(fileId uint64) (*fileDescription, error) {
	fdSet.changeFdMu.RLock()
	defer fdSet.changeFdMu.RUnlock()

	if fileId > fdSet.latestFileId() {
		return nil, nil
	}

	// get from cache
	if fd, ok := fdSet.fileFdCache[fileId]; ok {
		return fd, nil
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
func (fdSet *fdManager) GetTmpFlushFd(fileId uint64) (*fileDescription, error) {
	file, err := fdSet.getFileFd(fileId)
	if err != nil {
		return nil, err
	}

	if file == nil {
		file, err = fdSet.createNewFile(fileId)
		if err != nil {
			return nil, err
		}

		if file == nil {
			return nil, nil
		}
	}

	return NewFdByBuffer(fdSet, &fileCacheItem{
		FileId: fileId,

		FileWriter: file,
	}), err
}

func (fdSet *fdManager) GetWriteFd() *fileDescription {
	fdSet.changeFdMu.RLock()
	defer fdSet.changeFdMu.RUnlock()

	return fdSet.writeFd
}

func (fdSet *fdManager) DeleteTo(location *Location) error {
	fdSet.changeFdMu.Lock()
	defer fdSet.changeFdMu.Unlock()

	for i := fdSet.latestFileId(); i > location.FileId; i-- {
		if fdSet.writeFd != nil {
			fdSet.writeFd.cacheItem.FileWriter.Close()
			fdSet.writeFd.Close()

			fdSet.writeFd = nil
		}

		cacheItem := fdSet.fileCache.Back().Value.(*fileCacheItem)
		if cacheItem != nil {

			delete(fdSet.fileFdCache, cacheItem.FileId)

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

func (fdSet *fdManager) DiskDelete(highLocation *Location, lowLocation *Location) error {
	for i := highLocation.FileId; i > lowLocation.FileId; i-- {
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
	fdSet.changeFdMu.Lock()
	defer fdSet.changeFdMu.Unlock()

	// new location
	newLocation := NewLocation(fdSet.latestFileId()+1, 0)

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
	fdSet.changeFdMu.Lock()
	defer fdSet.changeFdMu.Unlock()

	fdSet.reset()

	if err := os.RemoveAll(fdSet.dirName); err != nil {
		return err
	}

	return nil
}
func (fdSet *fdManager) Close() error {
	fdSet.changeFdMu.Lock()
	defer fdSet.changeFdMu.Unlock()

	fdSet.reset()

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
		cacheItem := fdSet.writeFd.cacheItem

		cacheItem.Mu.Lock()
		defer cacheItem.Mu.Unlock()

		if cacheItem.BufferLen > location.Offset {
			cacheItem.BufferLen = location.Offset
		}
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

	nextFlushStartLocation := fdSet.fileManager.NextFlushStartLocation()

	var newItem *fileCacheItem

	if nextFlushStartLocation != nil {
		nextFlushStartFileId := nextFlushStartLocation.FileId

		// remove stale cache
		for fdSet.fileCache.Len() > fdSet.fileCacheLength {
			item := fdSet.fileCache.Front().Value.(*fileCacheItem)
			if item.FileId >= nextFlushStartFileId {
				break
			}

			fdSet.fileCache.Remove(fdSet.fileCache.Front())
		}

		// ring buffer. reuse cache
		if fdSet.fileCache.Len() >= fdSet.fileCacheLength {
			newItem = fdSet.fileCache.Front().Value.(*fileCacheItem)
			if newItem.FileId < nextFlushStartFileId {
				fdSet.fileCache.MoveToBack(fdSet.fileCache.Front())

				delete(fdSet.fileFdCache, newItem.FileId)
			} else {
				newItem = nil
			}
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

	newItem.Mu.Unlock()

	if bufferLen > 0 && fd != nil {
		if _, err := fd.Read(newItem.Buffer[:bufferLen]); err != nil {
			return err
		}
	}

	fdSet.writeFd = NewFdByBuffer(fdSet, newItem)

	fdSet.fileFdCache[newItem.FileId] = fdSet.writeFd

	return nil
}

func (fdSet *fdManager) latestFileId() uint64 {
	return fdSet.writeFd.cacheItem.FileId
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
	for _, fileFd := range fdSet.fileFdCache {
		// close file reader
		fileFd.Close()
		if fileFd.cacheItem != nil && fileFd.cacheItem.FileWriter != nil {
			// close file writer
			fileFd.cacheItem.FileWriter.Close()
		}
	}

	fdSet.fileCache = nil
	fdSet.writeFd = nil
	fdSet.fileFdCache = nil
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
