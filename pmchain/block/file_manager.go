package chain_block

import (
	"fmt"
	"github.com/pkg/errors"
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
	maxFileSize int64

	dirName string
	dirFd   *os.File

	latestFileId   uint64
	latestFileFd   *os.File
	latestFileSize int64
}

func newFileManager(dirName string) (*fileManager, error) {
	var err error

	fm := &fileManager{
		dirName:     dirName,
		maxFileSize: 10 * 1024 * 1024,
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

func (fm *fileManager) Close() error {
	fm.dirName = ""
	if fm.dirFd != nil {
		if err := fm.dirFd.Close(); err != nil {
			return err
		}
	}

	fm.latestFileId = 0
	fm.latestFileSize = 0
	if fm.latestFileFd != nil {
		if err := fm.latestFileFd.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (fm *fileManager) Write(buf []byte) (*Location, error) {
	bufSize := int64(len(buf))
	if fm.latestFileSize+bufSize > fm.maxFileSize {
		fm.moveOneForward()
	}

	if _, err := fm.latestFileFd.Write(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("fm.latestFileFd.Write failed, error is %s", err.Error()))
	}

	location := newLocation(fm.latestFileId, uint32(fm.latestFileSize)+1)
	fm.latestFileSize += bufSize

	return location, nil
}

func (fm *fileManager) DeleteTo(location *Location) (*Location, error) {
	return nil, nil
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
