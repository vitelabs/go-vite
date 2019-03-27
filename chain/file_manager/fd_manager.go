package chain_block

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"strconv"
)

type fdManager struct {
	dirName string
	dirFd   *os.File

	filenamePrefix     string
	filenamePrefixSize int
}

func (fdSet *fdManager) GetFd(location *Location) (*os.File, error) {

	return nil, nil
}

func (fdSet *fdManager) ReleaseFd(fd *os.File) {

}

func (fdSet *fdManager) RemoveFile(fileId uint64) error {
	return nil
}

func (fdSet *fdManager) NextFd(location *Location) (*os.File, *Location, error) {
	nextLocation := NewLocation(location.FileId+1, 0)
	var fd *os.File
	var err error
	fd, err = fdSet.GetFd(location)
	if err != nil {
		return nil, nil, err
	}
	if fd != nil {
		return fd, nextLocation, nil
	}

	fd, err = fdSet.createNewFile(nextLocation.FileId)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("fdSet.createNewFile failed, nextFileId is %d. Error: %s,", nextLocation.FileId, err))
	}

	fdSet.refFd(nextLocation.FileId, fd)
	return fd, nextLocation, nil
}

func (fdSet *fdManager) refFd(fileId uint64, file *os.File) {

}

func (fdSet *fdManager) createNewFile(fileId uint64) (*os.File, error) {
	absoluteFilename := fdSet.fileIdToAbsoluteFilename(fileId)

	file, cErr := os.Create(absoluteFilename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	return file, nil
}

func (fdSet *fdManager) fileIdToAbsoluteFilename(fileId uint64) string {
	return path.Join(fdSet.dirName, fdSet.filenamePrefix+strconv.FormatUint(fileId, 10))
}

func (fdSet *fdManager) filenameToFileId(filename string) (uint64, error) {
	fileIdStr := filename[fdSet.filenamePrefixSize:]
	return strconv.ParseUint(fileIdStr, 10, 64)

}
