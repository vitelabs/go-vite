package chain_file_manager

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"sync"
)

type FileManager struct {
	fileSize int64

	fdSet                  *fdManager
	nextFlushStartLocation *Location
	prevFlushLocation      *Location

	lockMu sync.RWMutex

	fSyncWg sync.WaitGroup
}

func NewFileManager(dirName string, fileSize int64, cacheCount int) (*FileManager, error) {
	fm := &FileManager{
		fileSize: fileSize,
	}

	fdSet, err := newFdManager(dirName, int(fileSize), cacheCount)
	if err != nil {
		return nil, err
	}

	fm.fdSet = fdSet
	fm.nextFlushStartLocation = fm.fdSet.LatestLocation()
	fm.prevFlushLocation = fm.nextFlushStartLocation

	return fm, nil
}

func (fm *FileManager) NextFlushStartLocation() *Location {
	return fm.nextFlushStartLocation
}

func (fm *FileManager) LatestLocation() *Location {
	return fm.fdSet.LatestLocation()
}

func (fm *FileManager) Write(buf []byte) (*Location, error) {
	bufSize := len(buf)

	location := fm.fdSet.LatestLocation()
	n := 0
	for n < bufSize {
		count, err := fm.write(buf[n:])
		if err != nil {
			return nil, err
		}
		n += count
	}
	return location, nil
}
func (fm *FileManager) LockDelete() {
	fm.lockMu.RLock()
}

func (fm *FileManager) UnlockDelete() {
	fm.lockMu.RUnlock()
}

func (fm *FileManager) DeleteTo(location *Location) error {
	fm.lockMu.Lock()
	defer fm.lockMu.Unlock()

	if location.Compare(fm.LatestLocation()) >= 0 {
		return nil
	}
	if err := fm.fdSet.DeleteTo(location); err != nil {
		return err
	}

	if location.Compare(fm.nextFlushStartLocation) < 0 {
		fm.nextFlushStartLocation = location
	}
	return nil
}

func (fm *FileManager) Flush(startLocation *Location, targetLocation *Location) error {
	var fdList []fileDescription

	// flush
	flushStartLocation := startLocation

	for flushStartLocation.Compare(targetLocation) < 0 {
		fd, err := fm.fdSet.GetFd(flushStartLocation.FileId)
		if err != nil {
			return errors.New(fmt.Sprintf("fm.fdSet.GetFd failed, fileId is %d. Error: %s. ", flushStartLocation.FileId, err.Error()))
		}
		if fd == nil {
			return errors.New(fmt.Sprintf("fd is nil, fileId is %+v\n", flushStartLocation.FileId))
		}

		targetOffset := fm.fileSize
		if flushStartLocation.FileId == targetLocation.FileId {
			targetOffset = targetLocation.Offset
		}

		offset, err := fd.Flush(targetOffset)
		if err != nil {
			return errors.New(fmt.Sprintf("fd flush failed, fileId is %d", flushStartLocation.FileId))
		}
		fdList = append(fdList, *fd)

		if offset >= fm.fileSize {
			flushStartLocation.FileId += 1
			flushStartLocation.Offset = 0

		} else {
			flushStartLocation.Offset = offset
		}
	}

	fm.nextFlushStartLocation = flushStartLocation

	if fm.prevFlushLocation.Compare(fm.nextFlushStartLocation) > 0 {
		// Disk delete
		if err := fm.fdSet.DiskDelete(fm.prevFlushLocation, fm.nextFlushStartLocation); err != nil {
			return err
		}
	}

	fm.prevFlushLocation = fm.nextFlushStartLocation

	// sync
	fdListLen := len(fdList)
	if fdListLen <= 0 {
		return nil
	} else if fdListLen <= 1 {
		fdList[0].Sync()
	} else {
		fm.fSyncWg.Add(len(fdList))
		for _, fd := range fdList {
			tmpFd := fd
			go func() {
				defer fm.fSyncWg.Done()
				tmpFd.Sync()
			}()
		}
		fm.fSyncWg.Wait()
	}

	return nil
}
func (fm *FileManager) GetNextLocation(location *Location) (*Location, error) {
	bufSizeBytes := make([]byte, 4)
	_, _, err := fm.ReadRaw(location, bufSizeBytes)
	if err != nil {
		return nil, err
	}
	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	offset := location.Offset + int64(bufSize) + 4
	return NewLocation(location.FileId+uint64(offset/fm.fileSize), offset%fm.fileSize), nil
}

func (fm *FileManager) Read(location *Location) ([]byte, *Location, error) {
	bufSizeBytes := make([]byte, 4)
	nextLocation, _, err := fm.ReadRaw(location, bufSizeBytes)
	if err != nil {
		return nil, nextLocation, err
	}

	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize)

	nextLocation, _, err = fm.ReadRaw(nextLocation, buf)

	return buf, nextLocation, err
}

func (fm *FileManager) ReadRaw(startLocation *Location, buf []byte) (*Location, int, error) {
	fileSize := fm.fileSize
	readLen := len(buf)

	i := 0
	currentLocation := startLocation
	for i < readLen {
		readSize := readLen - i
		freeSize := int(fileSize - currentLocation.Offset)
		if readSize > freeSize {
			readSize = freeSize
		}

		fd, err := fm.fdSet.GetFd(currentLocation.FileId)
		if err != nil {
			return currentLocation, 0, err
		}
		if fd == nil {
			return currentLocation, i, io.EOF
		}

		readN, rErr := fd.ReadAt(buf[i:i+readSize], currentLocation.Offset)
		fd.Close()

		i += readN

		nextOffset := currentLocation.Offset + int64(readN)

		if nextOffset >= fileSize {
			currentLocation = NewLocation(currentLocation.FileId+1, 0)
		} else {
			currentLocation = NewLocation(currentLocation.FileId, nextOffset)
		}

		if rErr != nil {
			return currentLocation, i, rErr
		}

	}
	return currentLocation, i, nil
}

func (fm *FileManager) ReadRange(startLocation *Location, endLocation *Location, parser DataParser) {
	realEndLocation := endLocation
	if realEndLocation == nil {
		realEndLocation = fm.LatestLocation()
	}

	currentLocation := startLocation
	for currentLocation.FileId <= realEndLocation.FileId {
		fd, err := fm.fdSet.GetFd(currentLocation.FileId)
		if err != nil {
			parser.WriteError(errors.New(fmt.Sprintf("fm.fdSet.GetFd failed, fileId is %d. Error: %s. ", currentLocation.FileId, err.Error())))
			return
		}
		if fd == nil {
			parser.WriteError(errors.New(fmt.Sprintf("fd is nil, location is %+v\n", currentLocation)))
			return
		}

		toLocation := NewLocation(currentLocation.FileId, fm.fileSize)
		if currentLocation.FileId == realEndLocation.FileId {
			toLocation = realEndLocation
		}

		buf, err := fm.readFile(fd, currentLocation, toLocation)
		fd.Close()

		if err := parser.Write(buf); err != nil {
			return
		}

		if err != nil {
			parser.WriteError(err)
			return
		}

		currentLocation = NewLocation(currentLocation.FileId+1, 0)
	}
}

func (fm *FileManager) Close() error {
	if err := fm.fdSet.Close(); err != nil {
		return nil
	}

	return nil
}

func (fm *FileManager) readFile(fd *fileDescription, fromLocation *Location, toLocation *Location) ([]byte, error) {
	buf := make([]byte, toLocation.Offset-fromLocation.Offset)

	readN, rErr := fd.ReadAt(buf, fromLocation.Offset)

	return buf[:readN], rErr

}

func (fm *FileManager) write(buf []byte) (int, error) {
	bufLen := len(buf)

	fd := fm.fdSet.GetWriteFd()
	count, err := fd.Write(buf)

	if err != nil {
		return 0, err
	}

	if count < bufLen {
		if err := fm.fdSet.CreateNextFd(); err != nil {
			return count, errors.New(fmt.Sprintf("fm.Write failed, error is %s", err.Error()))
		}
	}

	return count, nil
}
