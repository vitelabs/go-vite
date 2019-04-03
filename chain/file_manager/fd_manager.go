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

	File *os.File
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
	fdSet.maxFileId = location.FileId

	if fdSet.maxFileId <= 0 {
		fdSet.maxFileId = 1
	}

	if err = fdSet.resetWriteFd(); err != nil {
		return nil, errors.New(fmt.Sprintf("fdSet.resetWriteFd failed. Error %s", err))
	}

	return fdSet, nil
}

func (fdSet *fdManager) LatestLocation() *Location {
	writeFd := fdSet.GetWriteFd()
	return NewLocation(writeFd.cacheItem.FileId, writeFd.cacheItem.BufferLen)
}
func (fdSet *fdManager) GetFd(fileId uint64) (*fileDescription, error) {

	if fileId > fdSet.maxFileId {
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
	// remove files
	for i := fdSet.maxFileId; i > location.FileId; i-- {
		if err := os.Remove(fdSet.fileIdToAbsoluteFilename(i)); err != nil {
			return err
		}
		if fdSet.writeFd != nil {
			fdSet.writeFd.Close()
			fdSet.writeFd = nil
		}

		cacheItem := fdSet.fileCache.Back().Value.(*fileCacheItem)
		if cacheItem != nil {
			cacheItem.Mu.Lock()

			if err := cacheItem.File.Close(); err != nil {
				cacheItem.Mu.Unlock()
				return err
			}

			cacheItem.FileId = 0
			cacheItem.Buffer = nil
			cacheItem.BufferLen = 0
			cacheItem.Mu.Unlock()

			fdSet.fileCache.Remove(fdSet.fileCache.Back())
		}
		fdSet.maxFileId = i - 1
	}

	// recover write fd
	if err := fdSet.resetWriteFd(); err != nil {
		return err
	}

	// delete
	if err := fdSet.writeFd.DeleteTo(location.Offset); err != nil {
		return err
	}
	return nil
}

func (fdSet *fdManager) CreateNextFd() error {
	// set write fd
	fdSet.writeFd = nil

	// update maxFileId
	fdSet.maxFileId += 1

	// update fileReader cache
	if err := fdSet.resetWriteFd(); err != nil {
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
func (fdSet *fdManager) resetWriteFd() error {
	if fdSet.writeFd != nil {
		return nil
	}
	fileId := fdSet.maxFileId

	fd, err := fdSet.getFileFd(fileId)
	if err != nil {
		return err
	}
	if fd == nil {
		var err error
		fd, err = fdSet.createNewFile(fileId)
		if err != nil {
			return errors.New(fmt.Sprintf("fdSet.createNewFile failed, fileId is %d. Error: %s,", fileId, err))
		}
	}

	fileSize, err := fileutils.FileSize(fd)
	if err != nil {
		return err
	}

	var newItem *fileCacheItem
	if fdSet.fileCache.Len() >= fdSet.fileCacheLength {
		newItem = fdSet.fileCache.Front().Value.(*fileCacheItem)

		newItem.Mu.Lock()

		if err := newItem.File.Close(); err != nil {
			newItem.Mu.Unlock()
			return err
		}

		newItem.File = fd
		newItem.BufferLen = fileSize
		newItem.FlushPointer = fileSize
		newItem.FileId = fileId

		newItem.Mu.Unlock()

		fdSet.fileCache.MoveToBack(fdSet.fileCache.Front())
	} else {
		newItem = &fileCacheItem{
			Buffer:       make([]byte, fdSet.fileSize),
			BufferLen:    fileSize,
			FlushPointer: fileSize,
			FileId:       fileId,
			File:         fd,
		}
		fdSet.fileCache.PushBack(newItem)
	}

	if fileSize > 0 {
		if _, err := fd.Read(newItem.Buffer[:fileSize]); err != nil {
			return err
		}
	}

	// seek to end
	if _, err := fd.Seek(0, 2); err != nil {
		return err
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
