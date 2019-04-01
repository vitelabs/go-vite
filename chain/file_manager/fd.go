package chain_file_manager

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
)

var offsetTooBigErr = errors.New("offset > len(fd.buffer)")

type fileDescription struct {
	writePerm bool

	fdSet      *fdManager
	fileReader *os.File
	cacheItem  *fileCacheItem

	fileId uint64

	readPointer int64

	writeMaxSize int64

	closed bool
}

func NewFdByFile(file *os.File) *fileDescription {
	return &fileDescription{
		fileReader: file,
		writePerm:  false,
	}
}

func NewFdByBuffer(fdSet *fdManager, cacheItem *fileCacheItem) *fileDescription {
	return &fileDescription{
		fdSet:     fdSet,
		cacheItem: cacheItem,

		fileId: cacheItem.FileId,

		writePerm: false,
	}
}

func NewWriteFd(fdSet *fdManager, cacheItem *fileCacheItem, writeMaxSize int64) *fileDescription {
	return &fileDescription{
		fdSet:     fdSet,
		cacheItem: cacheItem,

		fileId: cacheItem.FileId,

		writeMaxSize: writeMaxSize,
		writePerm:    true,
	}
}

func (fd *fileDescription) Read(b []byte) (int, error) {
	if fd.fileReader != nil {
		return fd.fileReader.Read(b)
	}

	readN, err := fd.readAt(b, fd.readPointer)
	if err != nil {
		return readN, err
	}

	fd.readPointer += int64(readN)

	return readN, nil
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

	if !fd.writePerm {
		return 0, errors.New(fmt.Sprintf("can't write, writePerm is %+v\n", fd.writePerm))
	}

	if fd.fileId != cacheItem.FileId {
		return 0, errors.New(fmt.Sprintf("fd.fileId is %d, cacheItem.FileId is %d", fd.fileId, cacheItem.FileId))
	}

	if cacheItem.BufferLen >= fd.writeMaxSize {
		return 0, errors.New("can't write, fileReader is full")
	}

	if cacheItem.File == nil {
		return 0, errors.New("cacheItem.File == nil")
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

func (fd *fileDescription) Flush(targetOffset int64) (int64, error) {
	if fd.closed {
		return 0, nil
	}

	cacheItem := fd.cacheItem

	if cacheItem == nil {
		return 0, errors.New("cacheItem is nil")
	}

	cacheItem.Mu.Lock()
	defer cacheItem.Mu.Unlock()

	if fd.fileId != cacheItem.FileId {
		return 0, errors.New(fmt.Sprintf("fd.fileId is %d, cacheItem.FileId is %d", fd.fileId, cacheItem.FileId))
	}

	if cacheItem.File == nil {
		return 0, errors.New("cacheItem.File == nil")
	}

	if cacheItem.FlushPointer >= cacheItem.BufferLen ||
		cacheItem.FlushPointer >= targetOffset {
		return cacheItem.FlushPointer, nil
	}

	n, err := cacheItem.File.Write(cacheItem.Buffer[cacheItem.FlushPointer:targetOffset])
	cacheItem.FlushPointer += int64(n)

	if err != nil {
		return cacheItem.FlushPointer, err
	}
	if err := cacheItem.File.Sync(); err != nil {
		return cacheItem.FlushPointer, err
	}

	return cacheItem.FlushPointer, nil
}

func (fd *fileDescription) DeleteTo(size int64) error {
	cacheItem := fd.cacheItem

	if cacheItem == nil {
		return errors.New("cacheItem is nil")
	}

	cacheItem.Mu.Lock()
	defer cacheItem.Mu.Unlock()

	if err := cacheItem.File.Truncate(size); err != nil {
		return err
	}

	if _, err := cacheItem.File.Seek(size, 0); err != nil {
		return err
	}

	cacheItem.BufferLen = size
	cacheItem.FlushPointer = size
	fd.readPointer = size

	return nil
}

func (fd *fileDescription) Close() {
	if fd.closed {
		return
	}
	if fd.fileReader != nil {
		fd.fileReader.Close()
	}
	fd.cacheItem = nil
}

func (fd *fileDescription) readAt(b []byte, offset int64) (int, error) {
	cacheItem := fd.cacheItem

	cacheItem.Mu.RLock()
	defer cacheItem.Mu.RUnlock()

	if cacheItem.FileId != fd.fileId {
		if len(cacheItem.Buffer) <= 0 {
			return 0, nil
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
		return 0, offsetTooBigErr
	}

	readN := len(b)
	offsetInt := int(offset)
	restLen := int(cacheItem.BufferLen) - offsetInt

	if readN > restLen {
		readN = restLen
	}

	copy(b, cacheItem.Buffer[offsetInt:offsetInt+readN])
	return readN, nil
}
