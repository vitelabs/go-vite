package chain_block

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	filenamePrefix       = "data"
	filenamePrefixLength = 4
)

type fileManager struct {
	maxFileSize  int64
	bufSizeBytes []byte

	dirName string
	dirFd   *os.File

	prevFiledId uint64
	prevOffset  uint32

	latestFileFd   *os.File
	latestFileId   uint64
	latestFileSize int64
}

func newFileManager(dirName string) (*fileManager, error) {
	var err error

	fm := &fileManager{
		dirName:      dirName,
		maxFileSize:  10 * 1024 * 1024,
		bufSizeBytes: make([]byte, 4),
	}

	fm.dirFd, err = fm.newDirFd(dirName)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("fm.newDirFd failed, error is %s, dirName is %s", err, dirName))
	}

	fm.latestFileId, err = fm.loadLatestFileId()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("fm.loadLatestFileId failed, error is %s", err))
	}

	if fm.latestFileId > 0 {
		fm.latestFileFd, err = fm.getFileFd(fm.latestFileId)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("fm.getFileFd failed, error is %s, fm.latestFileId is %d", err, fm.latestFileId))
		}
		fm.latestFileSize, err = fm.fileSize(fm.latestFileFd)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("fm.fileSize failed, error is %s, fm.latestFileId is %d", err, fm.latestFileId))
		}
	} else if err = fm.moveOneForward(); err != nil {
		return nil, errors.New(fmt.Sprintf("fm.moveOneForward failed, error is %s", err))
	}
	return fm, nil
}

func (fm *fileManager) LatestLocation() *Location {
	return NewLocation(fm.latestFileId, fm.latestFileSize)
}

func (fm *fileManager) RemoveAllFile() error {
	if fm.latestFileFd != nil {
		if err := fm.latestFileFd.Close(); err != nil {
			return err
		}
		fm.latestFileFd = nil
	}
	if fm.dirFd != nil {
		if err := fm.dirFd.Close(); err != nil {
			return err
		}
		fm.dirFd = nil
	}

	if err := os.RemoveAll(fm.dirName); err != nil {
		return err
	}
	fm.latestFileId = 0
	fm.latestFileSize = 0
	return nil
}

func (fm *fileManager) Close() error {
	fm.dirName = ""
	if fm.dirFd != nil {
		if err := fm.dirFd.Close(); err != nil {
			return err
		}
		fm.dirFd = nil
	}

	fm.latestFileId = 0
	fm.latestFileSize = 0
	if fm.latestFileFd != nil {
		if err := fm.latestFileFd.Close(); err != nil {
			return err
		}
		fm.latestFileFd = nil
	}
	return nil
}

func (fm *fileManager) Read(location *Location) ([]byte, error) {
	fd, err := fm.GetFd(location)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("bDB.fm.GetFd failed, [Error] %s, location is %+v", err.Error(), location))
	}
	if fd == nil {
		return nil, errors.New(fmt.Sprintf("fd is nil, location is %+v", location))
	}
	defer fd.Close()

	if _, err := fd.Seek(int64(location.Offset()), 0); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Seek failed, [Error] %s, location is %+v", err.Error(), location))
	}

	bufSizeBytes := make([]byte, 4)
	if _, err := fd.Read(bufSizeBytes); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}
	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize-4)
	if _, err := fd.Read(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}
	return buf, nil
}

func (fm *fileManager) Write(buf []byte) (*Location, error) {
	bufSize := int64(len(buf))

	if fm.latestFileSize+bufSize > fm.maxFileSize {
		fm.moveOneForward()
	}

	if _, err := fm.latestFileFd.Write(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("fm.latestFileFd.Write failed, error is %s", err.Error()))
	}

	location := NewLocation(fm.latestFileId, fm.latestFileSize)

	fm.latestFileSize += bufSize

	return location, nil
}

func (fm *fileManager) ReadRange(startLocation *Location, endLocation *Location, bfp *blockFileParser) {
	defer bfp.Close()

	startFileId := startLocation.FileId()

	var endLocationOffset int64
	var endFileId uint64
	if endLocation != nil {
		endFileId = endLocation.FileId()
		endLocationOffset = int64(endLocation.Offset())
	} else {
		endLocationOffset = fm.latestFileSize
		endFileId = fm.latestFileId
	}

	for i := startFileId; i <= endFileId; i++ {

		fd, err := fm.getFileFd(i)
		if err != nil {
			bfp.WriteErr(errors.New(fmt.Sprintf("bDB.fm.GetFd failed, [Error] %s. fileId is %d", err.Error(), i)))
			fd.Close()
			return
		}
		if fd == nil {

			bfp.WriteErr(errors.New(fmt.Sprintf("fd is nil, fileId is %d", i)))

			fd.Close()
			return
		}

		startOffset := int64(0)
		endOffset := int64(0)
		if i == startFileId {
			startOffset = int64(startLocation.Offset())
		}
		if i == endFileId {
			endOffset = endLocationOffset
			size, err := fm.readDataSize(fd, endOffset)
			if err != nil {
				bfp.WriteErr(errors.New(fmt.Sprintf("fm.readDataSize failed, Error: %s", err)))

				fd.Close()
				return
			}
			endOffset += int64(size)
		}

		buf, err := fm.readFile(fd, i, startOffset, endOffset)
		if err != nil {
			bfp.WriteErr(err)
			fd.Close()
			return
		}

		bfp.Write(i, buf)
		fd.Close()
	}
	return
}

func (fm *fileManager) DeleteTo(location *Location) error {
	startOffset := int64(location.Offset())

	for i := location.FileId(); i <= fm.latestFileId; i++ {
		if err := fm.deleteFile(i, startOffset); err != nil {
			return err
		}
		startOffset = 0
	}
	if fm.latestFileId < location.fileId {
		fm.latestFileId = location.fileId
		fm.latestFileFd.Close()

		var err error
		if fm.latestFileFd, err = fm.getFileFd(fm.latestFileId); err != nil {
			return err
		}
	}
	fm.latestFileSize = startOffset

	return nil
}

func (fm *fileManager) DeleteAndReadTo(location *Location, bfp *blockFileParser) {
	defer bfp.Close()

	startOffset := int64(location.Offset())

	for i := location.FileId(); i <= fm.latestFileId; i++ {
		buf, err := fm.deleteAndReadFile(i, startOffset)
		if err != nil {
			bfp.WriteErr(err)
			return
		}

		bfp.Write(i, buf)
		startOffset = 0
	}

	if fm.latestFileId < location.fileId {
		fm.latestFileId = location.fileId
		fm.latestFileFd.Close()

		var err error
		if fm.latestFileFd, err = fm.getFileFd(fm.latestFileId); err != nil {
			bfp.WriteErr(err)
			return
		}
	}
	fm.latestFileSize = startOffset

	return
}

func (fm *fileManager) GetFd(location *Location) (*os.File, error) {
	return fm.getFileFd(location.FileId())
}

func (fm *fileManager) readDataSize(fd *os.File, offset int64) (uint32, error) {
	dataSize := make([]byte, 4)
	readN, rErr := fd.ReadAt(dataSize, offset)

	if rErr != nil && rErr != io.EOF {
		return 0, rErr
	}
	if readN < 4 {
		return 0, nil
	}

	return binary.BigEndian.Uint32(dataSize), nil

}
func (fm *fileManager) readFile(fd *os.File, fileId uint64, startOffset, endOffset int64) ([]byte, error) {

	if endOffset == 0 {
		if fileId == fm.latestFileId {
			endOffset = fm.latestFileSize
		} else {
			endOffset = fm.maxFileSize
		}
	}

	buf := make([]byte, endOffset-startOffset)

	readN, rErr := fd.ReadAt(buf, startOffset)

	if rErr != nil && rErr != io.EOF {
		return nil, rErr
	}

	return buf[:readN], nil
}

func (fm *fileManager) deleteFile(fileId uint64, startOffset int64) error {
	fd, err := fm.getFileFd(fileId)
	if err != nil {
		return err
	}
	defer fd.Close()

	if startOffset <= 0 {
		if err := os.Remove(fd.Name()); err != nil {
			return err
		}
	} else {
		fd.Truncate(startOffset)
	}

	return nil
}

func (fm *fileManager) deleteAndReadFile(fileId uint64, toOffset int64) ([]byte, error) {
	fd, err := fm.getFileFd(fileId)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var buf []byte
	if fileId == fm.latestFileId {
		buf = make([]byte, fm.latestFileSize-toOffset)
	} else {
		buf = make([]byte, fm.maxFileSize-toOffset)
	}

	readN, rErr := fd.ReadAt(buf, toOffset)

	if rErr != nil && rErr != io.EOF {
		return nil, rErr
	}

	if toOffset <= 0 {
		if err := os.Remove(fd.Name()); err != nil {
			return nil, err
		}
	} else {
		fd.Truncate(toOffset)
	}

	return buf[:readN], nil
}

func (fm *fileManager) newDirFd(dirName string) (*os.File, error) {
	var dirFd *os.File
	for dirFd == nil {
		var openErr error
		dirFd, openErr = os.Open(dirName)
		if openErr != nil {
			if os.IsNotExist(openErr) {
				var cErr error
				cErr = os.Mkdir(dirName, 0744)

				if cErr != nil {
					return nil, errors.New(fmt.Sprintf("Create %s failed, error is %s", dirName, cErr.Error()))
				}
			} else {
				return nil, errors.New(fmt.Sprintf("os.Open %s failed, error is %s", dirName, openErr.Error()))
			}
		}
	}

	return dirFd, nil
}

func (fm *fileManager) loadLatestFileId() (uint64, error) {
	allFilename, readErr := fm.dirFd.Readdirnames(0)
	if readErr != nil {
		return 0, errors.New(fmt.Sprintf("fm.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	latestFileId := uint64(0)
	for _, filename := range allFilename {
		if !fm.isDataFile(filename) {
			continue
		}

		fileId, err := fm.filenameToFileId(filename)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("strconv.ParseUint failed, error is %s, fileName is %s", err.Error(), filename))
		}

		if fileId > latestFileId {
			latestFileId = fileId
		}
	}

	return latestFileId, nil
}

func (fm *fileManager) isDataFile(filename string) bool {

	return strings.HasPrefix(filename, filenamePrefix)
}

func (fm *fileManager) fileIdToAbsoluteFilename(fileId uint64) string {
	return path.Join(fm.dirName, filenamePrefix+strconv.FormatUint(fileId, 10))
}

func (fm *fileManager) filenameToFileId(filename string) (uint64, error) {
	fileIdStr := filename[filenamePrefixLength:]
	return strconv.ParseUint(fileIdStr, 10, 64)

}

func (fm *fileManager) getFileFd(fileId uint64) (*os.File, error) {
	absoluteFilename := fm.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}

func (fm *fileManager) fileSize(fd *os.File) (int64, error) {
	fileInfo, err := fd.Stat()
	if err != nil {
		return 0, errors.New(fmt.Sprintf("fd.Stat() failed, error is %s", err.Error()))
	}

	return fileInfo.Size(), nil
}

func (fm *fileManager) moveOneForward() error {
	nextLatestFileId := fm.latestFileId + 1

	fd, err := fm.createNewFile(nextLatestFileId)
	if err != nil {
		return errors.New(fmt.Sprintf("moveToNextFd failed, error is %s, nextLatestFileId is %d", err, nextLatestFileId))
	}

	if err := fm.latestFileFd.Close(); err != nil {
		errors.New(fmt.Sprintf("fm.latestFileFd.Close() failed, error is %s, latestFileId is %d", err, fm.latestFileId))
	}

	fm.latestFileId = nextLatestFileId
	fm.latestFileFd = fd
	fm.latestFileSize = 0

	return nil
}

func (fm *fileManager) createNewFile(fileId uint64) (*os.File, error) {
	absoluteFilename := fm.fileIdToAbsoluteFilename(fileId)

	file, cErr := os.Create(absoluteFilename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	return file, nil
}
