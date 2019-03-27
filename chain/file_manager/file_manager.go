package chain_block

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
)

type fileManager struct {
	maxFileSize int64

	fdSet    *fdManager
	location *Location

	writeBuffer        []byte
	writeBufferPointer int
	writeBufferSize    int
}

func (fm *fileManager) Write(buf []byte) (*Location, error) {
	bufSize := len(buf)

	n := 0
	for n < bufSize {
		count, err := fm.write(buf[n:])
		if err != nil {
			return nil, err
		}
		n += count
	}
	return fm.location, nil
}

func (fm *fileManager) Read(location *Location) ([]byte, error) {
	fd, err := fm.fdSet.GetFd(location)
	if err != nil {
		return nil, err
	}
	defer fm.fdSet.ReleaseFd(fd)

	if _, err := fd.Seek(location.Offset, 0); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Seek failed, [Error] %s, location is %+v", err.Error(), location))
	}

	bufSizeBytes := make([]byte, 4)
	if _, err := fd.Read(bufSizeBytes); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}
	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize)
	if _, err := fd.Read(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}
	return buf, nil
}

func (fm *fileManager) ReadRange(startLocation *Location, endLocation *Location, parser DataParser) {

	realEndLocation := endLocation
	if realEndLocation == nil {
		realEndLocation = fm.location
	}

	currentLocation := startLocation
	for currentLocation.FileId <= realEndLocation.FileId {
		fd, err := fm.fdSet.GetFd(startLocation)
		if err != nil {
			parser.WriteError(errors.New(fmt.Sprintf("fm.fdSet.GetFd failed, fileId is %d. Error: %s. ", currentLocation.FileId, err.Error())))
			fm.fdSet.ReleaseFd(fd)
			return
		}

		buf, err := fm.readFile(fd, currentLocation)
		if err != nil {
			parser.WriteError(err)
			fm.fdSet.ReleaseFd(fd)
			return
		}

		parser.Write(buf, currentLocation)
	}

}

func (fm *fileManager) DeleteTo(location *Location) error {
	for i := fm.location.FileId; i > location.FileId; i-- {
		if err := fm.fdSet.RemoveFile(i); err != nil {
			return err
		}
	}

	// Truncate

	fd, err := fm.fdSet.GetFd(location)
	if err != nil {
		return err
	}
	defer fm.fdSet.ReleaseFd(fd)

	if err := fd.Truncate(location.Offset); err != nil {
		return err
	}

	fm.location = location

	return nil
}
func (fm *fileManager) DeleteAndReadTo(toLocation *Location, parser *DataParser) error {
	return nil
}

func (fm *fileManager) RemoveAllFile() error {
	return nil
}

func (fm *fileManager) Close() error {
	return nil
}

func (fm *fileManager) readFile(fd *os.File, location *Location) ([]byte, error) {
	startOffset := location.Offset

	if location.FileId == fm.location.FileId {
		// TODO
		return fm.writeBuffer[startOffset:fm.writeBufferPointer], nil
	}

	buf := make([]byte, fm.maxFileSize-startOffset)

	readN, rErr := fd.ReadAt(buf, startOffset)

	if rErr != nil && rErr != io.EOF {
		return nil, rErr
	}

	return buf[:readN], nil
}
func (fm *fileManager) write(buf []byte) (int, error) {
	bufLen := len(buf)
	freeSpaceLength := fm.writeBufferSize - fm.writeBufferPointer
	nextPointer := 0

	if freeSpaceLength < bufLen {
		nextPointer = freeSpaceLength + fm.writeBufferPointer
	} else {
		nextPointer = bufLen + fm.writeBufferPointer
	}

	copy(fm.writeBuffer[fm.writeBufferPointer:nextPointer], buf)

	count := nextPointer - fm.writeBufferPointer
	fm.writeBufferPointer = nextPointer

	if count < bufLen {
		fd, err := fm.fdSet.GetFd(fm.location)
		defer fm.fdSet.ReleaseFd(fd)
		if err != nil {
			return 0, err
		}

		if _, err := fd.Write(fm.writeBuffer); err != nil {
			return 0, errors.New(fmt.Sprintf("fm.Write failed, error is %s", err.Error()))
		}

		fm.writeBufferPointer = 0
		if _, nextLocation, err := fm.fdSet.NextFd(fm.location); err != nil {
			return 0, errors.New(fmt.Sprintf("fm.Write failed, error is %s", err.Error()))
		} else {
			fm.location = nextLocation
		}

	}

	return count, nil
}
