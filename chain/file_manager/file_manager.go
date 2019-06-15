package chain_file_manager

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"sync"
)

type FileManager struct {
	fileSize int64

	fdSet                  *fdManager
	nextFlushStartLocation *Location
	prevFlushLocation      *Location

	fSyncWg sync.WaitGroup
	log     log15.Logger
}

func NewFileManager(dirName string, fileSize int64, cacheCount int) (*FileManager, error) {
	fm := &FileManager{
		fileSize: fileSize,
		log:      log15.New("module", "fileManager"),
	}

	fdSet, err := newFdManager(fm, dirName, int(fileSize), cacheCount)
	if err != nil {
		return nil, err
	}

	fm.fdSet = fdSet
	fm.nextFlushStartLocation = fm.fdSet.LatestLocation()
	fm.prevFlushLocation = fm.nextFlushStartLocation

	return fm, nil
}

func (fm *FileManager) NextFlushStartLocation() *Location {
	if fm.nextFlushStartLocation == nil {
		return nil
	}
	return NewLocation(fm.nextFlushStartLocation.FileId, fm.nextFlushStartLocation.Offset)
}

func (fm *FileManager) SetNextFlushStartLocation(location *Location) {
	fm.nextFlushStartLocation = NewLocation(location.FileId, location.Offset)
}

func (fm *FileManager) LatestLocation() *Location {
	return fm.fdSet.LatestLocation()
}

func (fm *FileManager) Write(buf []byte) (*Location, error) {

	bufSize := len(buf)

	location := fm.fdSet.LatestLocation()
	// FOR DEBUG
	//fm.log.Info(fmt.Sprintf("file manager write %d bytes, location is %+v", bufSize, location), "method", "Write")

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

func (fm *FileManager) DeleteTo(location *Location) error {

	if location.Compare(fm.LatestLocation()) >= 0 {
		return nil
	}
	if err := fm.fdSet.DeleteTo(location); err != nil {
		return err
	}

	if location.Compare(fm.nextFlushStartLocation) < 0 {
		fm.nextFlushStartLocation = location
	}

	// FOR DEBUG
	//fm.log.Info(fmt.Sprintf("file manager delete to %+v, fm.nextFlushStartLocation is %+v, latest location is %+v", location, fm.nextFlushStartLocation, fm.LatestLocation()), "method", "DeleteTo")
	return nil
}

func (fm *FileManager) Flush(startLocation *Location, targetLocation *Location, buf []byte) error {
	// flush
	flushLocation := NewLocation(startLocation.FileId, startLocation.Offset)

	bufStart := int64(0)

	for flushLocation.Compare(targetLocation) < 0 {
		fd, err := fm.fdSet.GetFd(flushLocation.FileId)
		if err != nil {
			return errors.New(fmt.Sprintf("fm.fdSet.GetFd failed, fileId is %d. Error: %s. ", flushLocation.FileId, err.Error()))
		}
		if fd == nil {
			fd, err = fm.fdSet.GetTmpFlushFd(flushLocation.FileId)
			if err != nil {
				return errors.New(fmt.Sprintf("fm.fdSet.GetTmpFlushFd failed, fileId is %d. Error: %s. ", flushLocation.FileId, err.Error()))
			}

			if fd == nil {
				return errors.New(fmt.Sprintf("fd is nil, fileId is %d. Error: %s", flushLocation.FileId, err))
			}
		}

		targetOffset := fm.fileSize
		if flushLocation.FileId == targetLocation.FileId {
			targetOffset = targetLocation.Offset
		}

		bufEnd := bufStart + targetOffset - flushLocation.Offset

		n, err := fd.Flush(flushLocation.Offset, buf[bufStart:bufEnd])
		if err != nil {
			return errors.New(fmt.Sprintf("fd flush failed, fileId is %d", flushLocation.FileId))
		}

		flushOffset := flushLocation.Offset + int64(n)
		bufStart += int64(n)

		if flushOffset >= fm.fileSize {
			flushLocation.FileId += 1
			flushLocation.Offset = 0
		} else {
			flushLocation.Offset = flushOffset
		}
	}

	// FOR DEBUG
	//fm.log.Info(fmt.Sprintf("file manager flush, start location is %+v, target location is %+v, buf size is %d", startLocation, targetLocation, len(buf)), "method", "Flush")

	if fm.prevFlushLocation.Compare(targetLocation) > 0 {
		// Disk delete
		if err := fm.fdSet.DiskDelete(fm.prevFlushLocation, targetLocation); err != nil {
			return err
		}
		// FOR DEBUG
		//fm.log.Info(fmt.Sprintf("file manager disk delete, prevFlushLocation is %+v, targetLocation is %+v", fm.prevFlushLocation, targetLocation), "method", "Flush")
	}

	fm.prevFlushLocation = targetLocation

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

func (fm *FileManager) SetLog(h log15.Handler) {
	fm.log.SetHandler(h)
}

func (fm *FileManager) Close() error {
	if err := fm.fdSet.Close(); err != nil {
		return nil
	}

	return nil
}

func (fm *FileManager) GetCacheStatusList() []interfaces.DBStatus {
	return []interfaces.DBStatus{{
		Name:   "blockDB.fm.cache",
		Count:  uint64(len(fm.fdSet.fileFdCache)),
		Size:   uint64(int64(len(fm.fdSet.fileFdCache)) * fm.fileSize),
		Status: "",
	}}
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
