package chain_file_manager

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fileutils"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"container/list"
)

type fileCacheItem struct {
	Buffer    []byte
	BufferLen int64
	Mu        sync.RWMutex
	FileId    uint64
	IsDelete  bool
}

type fdManager struct {
	dirName string
	dirFd   *os.File

	filenamePrefix     string
	filenamePrefixSize int

	fileCache       *list.List
	fileCacheLength int

	fileSize int64

	maxFileId uint64

	writeFd *fileDescription
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

	if location.FileId <= 0 {
		if location, err = fdSet.CreateNextFd(); err != nil {
			return nil, errors.New(fmt.Sprintf("fdSet.NextFd failed. Error %s", err))
		}
	}

	fdSet.maxFileId = location.FileId

	return fdSet, nil
}

func (fdSet *fdManager) LatestLocation() *Location {
	return NewLocation(fdSet.maxFileId, fdSet.writeFd.writePointer)
}
func (fdSet *fdManager) GetFd(location *Location) (*fileDescription, error) {
	fileId := location.FileId
	if fileId > fdSet.maxFileId {
		return nil, nil
	}

	fileCacheItem := fdSet.getCacheItem(location.FileId)
	if fileCacheItem != nil {
		return NewFdByBuffer(fdSet, fileCacheItem), nil
	}

	fd, err := fdSet.getFileFd(location.FileId)
	if err != nil {
		return nil, err
	}

	return NewFdByFile(fd), nil
}

func (fdSet *fdManager) GetWriteFd() *fileDescription {
	return fdSet.writeFd
}

func (fdSet *fdManager) DeleteTo(location *Location) error {
	latestLocation := fdSet.LatestLocation()
	// remove files
	for i := latestLocation.FileId; i > location.FileId; i-- {
		if err := os.Remove(fdSet.fileIdToAbsoluteFilename(i)); err != nil {
			return err
		}

		fdSet.writeFd = nil
		fileCacheItem := fdSet.getCacheItem(location.FileId)
		if fileCacheItem != nil {
			fileCacheItem.Mu.Lock()
			fileCacheItem.FileId = 0
			fileCacheItem.Buffer = nil
			fileCacheItem.BufferLen = 0
			fileCacheItem.IsDelete = true
			fileCacheItem.Mu.Unlock()

			fdSet.fileCache.Remove(fdSet.fileCache.Back())
		}
		fdSet.maxFileId = i - 1
	}

	// recover write fd
	if fdSet.writeFd == nil {
		fd, err := fdSet.getFileFd(fdSet.maxFileId)
		if err != nil {
			return err
		}
		if fd == nil {
			if _, err = fdSet.CreateNextFd(); err != nil {
				return errors.New(fmt.Sprintf("fdSet.NextFd failed. Error %s", err))
			}
		} else {
			newItem := &fileCacheItem{
				Buffer:   make([]byte, fdSet.fileSize),
				FileId:   fdSet.maxFileId,
				IsDelete: false,
			}
			fileSize, err := fileutils.FileSize(fd)
			if err != nil {
				return err
			}
			newItem.BufferLen = fileSize

			fdSet.writeFd = NewWriteFd(fd, newItem)
		}
	}

	// truncate
	if err := fdSet.writeFd.Truncate(location.Offset); err != nil {
		return err
	}

	return nil
}

func (fdSet *fdManager) CreateNextFd() (*Location, error) {
	nextLocation := NewLocation(fdSet.maxFileId+1, 0)

	fd, err := fdSet.createNewFile(nextLocation.FileId)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("fdSet.createNewFile failed, nextFileId is %d. Error: %s,", nextLocation.FileId, err))
	}

	// write file
	if fdSet.writeFd != nil {
		if err := fdSet.writeFd.Flush(); err != nil {
			return nil, errors.New(fmt.Sprintf("fm.latestFileFd.Write failed, error is %s", err.Error()))
		}
	}

	// update maxFileId
	fdSet.maxFileId = nextLocation.FileId

	// update cache
	var newItem *fileCacheItem
	if fdSet.fileCache.Len() >= fdSet.fileCacheLength {
		newItem = fdSet.fileCache.Front().Value.(*fileCacheItem)

		newItem.Mu.Lock()
		newItem.BufferLen = 0
		newItem.FileId = nextLocation.FileId
		newItem.Mu.Unlock()

		fdSet.fileCache.MoveToBack(fdSet.fileCache.Front())

	} else {
		newItem = &fileCacheItem{
			Buffer:   make([]byte, fdSet.fileSize),
			FileId:   nextLocation.FileId,
			IsDelete: false,
		}
		fdSet.fileCache.PushBack(newItem)
	}

	// set write fd
	fdSet.writeFd = NewWriteFd(fd, newItem)
	return nextLocation, nil

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

	return NewLocation(maxFileId, 0), nil
}

func (fdSet *fdManager) reset() {
	fdSet.fileCache = list.New()
	fdSet.maxFileId = 0
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
		return nil, errors.New("Create file failed, error is " + cErr.Error())
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
