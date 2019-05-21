package chain_file_manager

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
)

type fileDescription struct {
	fdSet      *fdManager
	fileReader *os.File

	cacheItem *fileCacheItem

	fileId uint64

	writeMaxSize int64
}

func NewFdByFile(file *os.File) *fileDescription {
	return &fileDescription{
		fileReader: file,
	}
}

func NewFdByBuffer(fdSet *fdManager, cacheItem *fileCacheItem) *fileDescription {
	return &fileDescription{
		fdSet:     fdSet,
		cacheItem: cacheItem,

		fileId:       cacheItem.FileId,
		writeMaxSize: int64(len(cacheItem.Buffer)),
	}
}

func (fd *fileDescription) ReadAt(b []byte, offset int64) (int, error) {

	if fd.fileReader != nil {
		return fd.fileReader.ReadAt(b, offset)
	}

	readN, err := fd.readAt(b, offset)

	if err != nil {
		return readN, err
	}
	return readN, nil
}

func (fd *fileDescription) Write(buf []byte) (int, error) {

	cacheItem := fd.cacheItem

	cacheItem.Mu.Lock()
	defer cacheItem.Mu.Unlock()

	if fd.fileId != cacheItem.FileId {
		return 0, errors.New(fmt.Sprintf("fd.fileId is %d, cacheItem.FileId is %d", fd.fileId, cacheItem.FileId))
	}

	if cacheItem.BufferLen >= fd.writeMaxSize {
		return 0, nil
	}

	bufLen := len(buf)
	freeSpaceLength := int(fd.writeMaxSize - cacheItem.BufferLen)

	count := 0
	if freeSpaceLength < bufLen {
		count = freeSpaceLength
	} else {
		count = bufLen
	}

	nextPointer := cacheItem.BufferLen + int64(count)
	copy(cacheItem.Buffer[cacheItem.BufferLen:nextPointer], buf[:count])

	cacheItem.BufferLen = nextPointer

	return count, nil
}

func (fd *fileDescription) Flush(startOffset int64, buf []byte) (int, error) {

	cacheItem := fd.cacheItem

	if cacheItem == nil {
		return 0, errors.New("cacheItem is nil")
	}

	cacheItem.Mu.Lock()
	defer cacheItem.Mu.Unlock()

	if fd.fileId != cacheItem.FileId {
		return 0, errors.New(fmt.Sprintf("fd.fileId is %d, cacheItem.FileId is %d", fd.fileId, cacheItem.FileId))
	}

	if cacheItem.FileWriter == nil {
		fd, err := fd.fdSet.createNewFile(cacheItem.FileId)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("fdSet.createNewFile failed, fileId is %d. Error: %s,", cacheItem.FileId, err))
		}
		if fd == nil {
			return 0, errors.New("fd is nil")
		}
		cacheItem.FileWriter = fd
	}

	file := cacheItem.FileWriter

	if _, err := file.Seek(startOffset, 0); err != nil {
		return 0, err
	}

	n, err := file.Write(buf)
	if err != nil {
		return n, err
	}

	if err := file.Sync(); err != nil {
		return n, err
	}
	return n, nil
}

func (fd *fileDescription) Close() {
	if fd.fileReader != nil {
		fd.fileReader.Close()
	}
}

func (fd *fileDescription) readAt(b []byte, offset int64) (int, error) {
	cacheItem := fd.cacheItem

	cacheItem.Mu.RLock()
	defer cacheItem.Mu.RUnlock()

	if cacheItem.FileId != fd.fileId {
		// is delete
		if len(cacheItem.Buffer) <= 0 {
			return 0, io.EOF
		}
		var err error
		fd.fileReader, err = fd.fdSet.getFileFd(fd.fileId)
		if err != nil {
			return 0, err
		}
		if fd.fileReader == nil {
			return 0, errors.New(fmt.Sprintf("can't open fileReader, fileReader id is %d", fd.fileId))
		}

		return fd.fileReader.ReadAt(b, offset)
	}

	if offset > cacheItem.BufferLen {
		return 0, io.EOF
	}

	readN := len(b)
	offsetInt := int(offset)
	restLen := int(cacheItem.BufferLen) - offsetInt

	if readN > restLen {
		readN = restLen
	}

	copy(b, cacheItem.Buffer[offsetInt:offsetInt+readN])
	if readN < len(b) {
		return readN, io.EOF
	}
	return readN, nil
}
