package chain_file_manager

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
)

type FileManager struct {
	fileSize int64

	fdSet    *fdManager
	location *Location
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

	return fm, nil
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

func (fm *FileManager) Read(location *Location) ([]byte, []byte, error) {
	bufSizeBytes := make([]byte, 4)
	nextLocation, _, err := fm.ReadRaw(location, bufSizeBytes)
	if err != nil {
		if err == io.EOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize)
	if _, _, err := fm.ReadRaw(nextLocation, buf); err != nil && err != io.EOF {
		return nil, nil, err
	}

	return buf, bufSizeBytes, nil
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

		fd, err := fm.fdSet.GetFd(currentLocation)
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
		fd, err := fm.fdSet.GetFd(currentLocation)
		if err != nil {
			parser.WriteError(errors.New(fmt.Sprintf("fm.fdSet.GetFd failed, fileId is %d. Error: %s. ", currentLocation.FileId, err.Error())))
			return
		}
		if fd == nil {
			parser.WriteError(errors.New(fmt.Sprintf("fd is nil, location is %+v\n", currentLocation)))
			return
		}

		var toLocation *Location
		if currentLocation.FileId == realEndLocation.FileId {
			toLocation = realEndLocation
		}

		buf, err := fm.readFile(fd, currentLocation, toLocation)
		fd.Close()

		if err != nil {
			parser.WriteError(err)
			return
		}

		if err := parser.Write(buf); err != nil {
			return
		}

		currentLocation = NewLocation(currentLocation.FileId+1, 0)
	}
}

func (fm *FileManager) DeleteTo(location *Location) error {
	return fm.fdSet.DeleteTo(location)
}

func (fm *FileManager) Close() error {
	writeFd := fm.fdSet.GetWriteFd()
	if writeFd != nil {
		if err := writeFd.Flush(); err != nil {
			return err
		}
	}

	if err := fm.fdSet.Close(); err != nil {
		return nil
	}

	return nil
}

func (fm *FileManager) readFile(fd *fileDescription, fromLocation *Location, toLocation *Location) ([]byte, error) {
	startOffset := fromLocation.Offset
	var buf []byte
	if toLocation == nil {
		buf = make([]byte, fm.fileSize-startOffset)
	} else {
		buf = make([]byte, toLocation.Offset-fromLocation.Offset)
	}

	readN, rErr := fd.ReadAt(buf, startOffset)

	if rErr != nil && rErr != io.EOF {
		return nil, rErr
	}
	return buf[:readN], nil

}

func (fm *FileManager) write(buf []byte) (int, error) {
	bufLen := len(buf)

	fd := fm.fdSet.GetWriteFd()
	count, err := fd.Write(buf)

	if err != nil {
		return 0, err
	}

	if count < bufLen {
		if _, err := fm.fdSet.CreateNextFd(); err != nil {
			return count, errors.New(fmt.Sprintf("fm.Write failed, error is %s", err.Error()))
		}
	}

	return count, nil
}
