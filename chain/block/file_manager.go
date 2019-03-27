package chain_block

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fileutils"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	filenamePrefix       = "data"
	filenamePrefixLength = len(filenamePrefix)
)

type fileManager struct {
	maxFileSize int64

	dirName string
	dirFd   *os.File

	latestFileFd   *os.File
	latestFileId   uint64
	latestFileSize int64
}

func newFileManager(dirName string) (*fileManager, error) {
	var err error

	fm := &fileManager{
		dirName:     dirName,
		maxFileSize: 10 * 1024 * 1024,
	}

	fm.dirFd, err = fileutils.OpenOrCreateFd(dirName)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("fileutils.OpenOrCreateFd failed, error is %s, dirName is %s", err, dirName))
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
		fm.latestFileSize, err = fileutils.FileSize(fm.latestFileFd)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("fm.fileSize failed, error is %s, fm.latestFileId is %d", err, fm.latestFileId))
		}
	} else if err = fm.moveToNext(); err != nil {
		return nil, errors.New(fmt.Sprintf("fm.moveToNext failed, error is %s", err))
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

	if _, err := fd.Seek(location.Offset, 0); err != nil {
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
		fm.moveToNext()
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

	startFileId := startLocation.FileId

	var endFileId uint64
	var endLocationOffset int64

	if endLocation != nil {
		endFileId = endLocation.FileId
		endLocationOffset = endLocation.Offset
	} else {
		endFileId = fm.latestFileId
		endLocationOffset = fm.latestFileSize
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
			startOffset = startLocation.Offset
		}
		if i == endFileId {
			endOffset = endLocationOffset
			size, err := fm.readDataSize(fd, endOffset)
			if err != nil {
				bfp.WriteErr(errors.New(fmt.Sprintf("fm.readDataSize failed, Error: %s", err)))

				fd.Close()
				return
			}
			endOffset += size
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
	// remove
	for i := fm.latestFileId; i > location.FileId+1; i-- {
		if err := os.Remove(fm.fileIdToAbsoluteFilename(i)); err != nil {
			return err
		}

	}
	// Truncate
	if fm.latestFileId > location.FileId {
		fm.latestFileFd.Close()

		fm.latestFileId = location.FileId
		var err error
		fm.latestFileFd, err = fm.getFileFd(fm.latestFileId)
		if err != nil {
			return err
		}
	}
	fm.latestFileSize = location.Offset

	if err := fm.latestFileFd.Truncate(fm.latestFileSize); err != nil {
		return err
	}

	return nil
}

func (fm *fileManager) DeleteAndReadTo(location *Location, bfp *blockFileParser) {
	defer bfp.Close()

	fdList := make([]*os.File, 0, fm.latestFileId-location.FileId+1)

	startOffset := int64(location.Offset)

	for i := location.FileId; i <= fm.latestFileId; i++ {
		var fd *os.File
		if i == fm.latestFileId {

			fd = fm.latestFileFd

		} else {
			var err error
			fd, err = fm.getFileFd(i)
			if err != nil {
				bfp.WriteErr(err)
				return
			}
		}

		if i == location.FileId {
			size, err := fm.readDataSize(fd, startOffset)
			if err != nil {
				bfp.WriteErr(err)
				return
			}
			startOffset += size
		}

		fdList = append(fdList, fd)

		buf, err := fm.readFile(fd, i, startOffset, 0)
		if err != nil {
			bfp.WriteErr(err)
			return
		}

		bfp.Write(i, buf)
		startOffset = 0
	}

	for i := len(fdList) - 1; i > 1; i-- {
		// delete
		fd := fdList[i]
		filename := fd.Name()
		fd.Close()

		if err := os.Remove(filename); err != nil {
			bfp.WriteErr(err)
			return
		}
	}

	// truncate
	fm.latestFileId = location.FileId
	fm.latestFileFd = fdList[0]
	fm.latestFileSize = location.Offset

	if err := fm.latestFileFd.Truncate(fm.latestFileSize); err != nil {
		bfp.WriteErr(err)
		return
	}

	return
}

func (fm *fileManager) GetFd(location *Location) (*os.File, error) {
	return fm.getFileFd(location.FileId)
}

func (fm *fileManager) readDataSize(fd *os.File, offset int64) (int64, error) {
	dataSize := make([]byte, 4)
	readN, rErr := fd.ReadAt(dataSize, offset)

	if rErr != nil && rErr != io.EOF {
		return 0, rErr
	}
	if readN < 4 {
		return 0, nil
	}

	return int64(binary.BigEndian.Uint32(dataSize)), nil

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

func (fm *fileManager) moveToNext() error {
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
