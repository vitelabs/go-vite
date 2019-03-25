package chain_state_bak

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

var journalLog = log15.New("module", "chain/state_bak/mv_db/journal")

const (
	filenameSuffix       = ".log"
	filenameSuffixLength = len(filenameSuffix)
)

type journal struct {
	dirFd       *os.File
	dirName     string
	maxFileSize int64

	logFileNum uint64

	fileId    uint64
	minFileId uint64

	fd       *os.File
	fileSize int64
}

func newJournal(chainDir string) (*journal, error) {
	var err error

	dirName := path.Join(chainDir, "state_logs")
	dirFd, err := fileutils.OpenOrCreateFd(dirName)
	if err != nil {
		return nil, err
	}

	j := &journal{
		dirName:     dirName,
		dirFd:       dirFd,
		logFileNum:  5,
		maxFileSize: 6 * 1024 * 1024,
	}

	if err = j.loadFileId(); err != nil {
		return nil, err
	}

	if j.fileId > 0 {
		j.fd, err = j.getFileFd(j.fileId)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("j.getFileFd failed, error is %s, j.latestFileId is %d", err, j.fileId))
		}
		j.fileSize, err = fileutils.FileSize(j.fd)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("fileSize failed, error is %sj, j.latestFileId is %d", err, j.fileId))
		}
	} else if err = j.moveToNext(); err != nil {
		return nil, errors.New(fmt.Sprintf("j.moveToNext failed, error is %s", err))
	}

	return j, nil
}

func (j *journal) Write(blockHashList []*types.Hash) error {
	buf := make([]byte, 0, len(blockHashList))
	for _, blockHash := range blockHashList {
		buf = append(buf, blockHash.Bytes()...)
	}

	freeSpaceLength := j.maxFileSize - j.fileSize
	if freeSpaceLength > 0 {
		if _, err := j.fd.Write(buf[:freeSpaceLength]); err != nil {
			return errors.New(fmt.Sprintf("j.fd.Write failed, error is %s", err.Error()))
		}
	}

	if freeSpaceLength < int64(len(buf)) {
		j.moveToNext()

		if _, err := j.fd.Write(buf[freeSpaceLength:]); err != nil {
			return errors.New(fmt.Sprintf("j.fd.Write failed, error is %s", err.Error()))
		}
	}

	return nil
}

type Uint64Slice []uint64

func (c Uint64Slice) Len() int {
	return len(c)
}
func (c Uint64Slice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Uint64Slice) Less(i, j int) bool {
	return c[i] < c[j]
}

func (j *journal) LogFileIdList() ([]uint64, error) {
	allFilename, readErr := j.dirFd.Readdirnames(0)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("j.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	//filenameList := make([]string, 0, j.logFileNum)
	fileIdList := make(Uint64Slice, 0, j.logFileNum)
	for _, filename := range allFilename {
		if !j.isLogFile(filename) {
			continue
		}
		fileId, err := j.filenameToFileId(filename)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("j.filenameToFileId failed, filename: %s. Error: %s",
				filename, readErr.Error()))
		}
		fileIdList = append(fileIdList, fileId)
	}

	sort.Sort(sort.Reverse(fileIdList))

	return fileIdList, nil
}

func (j *journal) ReadFile(fileId uint64) ([]byte, error) {
	fd, err := j.getFd(fileId)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	readBuf := make([]byte, j.maxFileSize)
	readN, err := fd.Read(readBuf)
	if err != nil {
		return nil, err
	}
	return readBuf[:readN], nil
}

func (j *journal) loadFileId() error {
	allFilename, readErr := j.dirFd.Readdirnames(0)
	if readErr != nil {
		return errors.New(fmt.Sprintf("j.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	minFileId := uint64(1)

	maxFileId := uint64(0)
	for _, filename := range allFilename {
		if !j.isLogFile(filename) {
			continue
		}

		fileId, err := j.filenameToFileId(filename)
		if err != nil {
			return errors.New(fmt.Sprintf("strconv.ParseUint failed, error is %s, fileName is %s", err.Error(), filename))
		}

		if fileId > maxFileId {
			maxFileId = fileId
		}
		if fileId < minFileId {
			minFileId = fileId
		}
	}
	j.fileId = maxFileId
	j.minFileId = minFileId
	return nil
}

func (j *journal) removeOutDateLog() {
	if j.fileId-j.minFileId <= j.logFileNum {
		return
	}

	filename := j.fileIdToAbsoluteFilename(j.minFileId)

	if err := os.Remove(filename); err != nil {
		if err == os.ErrNotExist {
			j.minFileId += 1
			return
		}
		journalLog.Error(fmt.Sprintf("os.Remove failed, filename is %s. Error: %s, ", filename, err.Error()))
		return
	}
	j.minFileId += 1
}

func (j *journal) moveToNext() error {
	nextFileId := j.fileId + 1

	fd, err := j.createNewFile(nextFileId)
	if err != nil {
		return errors.New(fmt.Sprintf("moveToNextFd failed, error is %s, nextFileId is %d", err, nextFileId))
	}

	if err := j.fd.Close(); err != nil {
		errors.New(fmt.Sprintf("j.fd.Close() failed, error is %s, latestFileId is %d", err, j.fileId))
	}

	j.fileId = nextFileId
	j.fd = fd
	j.fileSize = 0

	j.removeOutDateLog()
	return nil
}

func (j *journal) getFileFd(fileId uint64) (*os.File, error) {
	absoluteFilename := j.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}

func (j *journal) createNewFile(fileId uint64) (*os.File, error) {
	absoluteFilename := j.fileIdToAbsoluteFilename(fileId)

	file, cErr := os.Create(absoluteFilename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	return file, nil
}
func (j *journal) fileIdToAbsoluteFilename(fileId uint64) string {
	return path.Join(j.dirName, strconv.FormatUint(fileId, 10)+filenameSuffix)
}

func (j *journal) filenameToFileId(filename string) (uint64, error) {
	fileIdStr := filename[:len(filename)-filenameSuffixLength]
	return strconv.ParseUint(fileIdStr, 10, 64)

}

func (j *journal) isLogFile(filename string) bool {
	return strings.HasSuffix(filename, ".log")
}

func (j *journal) getFd(fileId uint64) (*os.File, error) {
	absoluteFilename := j.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}
