package chain_state

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/fileutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

var journalLog = log15.New("module", "chain/state_bak/mv_db/undoLogger")

const (
	filenameSuffix       = ".log"
	filenameSuffixLength = len(filenameSuffix)
)

type undoLogger struct {
	pending     map[types.Hash][]byte
	dirFd       *os.File
	dirName     string
	maxFileSize int64

	logFileNum uint64

	fileId    uint64
	minFileId uint64

	fd       *os.File
	fileSize int64
}

func newUndoLogger(chainDir string) (*undoLogger, error) {
	var err error

	dirName := path.Join(chainDir, "state_logs")
	dirFd, err := fileutils.OpenOrCreateFd(dirName)
	if err != nil {
		return nil, err
	}

	logger := &undoLogger{
		dirName:     dirName,
		dirFd:       dirFd,
		logFileNum:  50,
		maxFileSize: 10 * 1024 * 1024,
	}

	if err = logger.loadFileId(); err != nil {
		return nil, err
	}

	if logger.fileId > 0 {
		logger.fd, err = logger.getFileFd(logger.fileId)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("logger.getFileFd failed, error is %s, logger.latestFileId is %d", err, logger.fileId))
		}
		logger.fileSize, err = fileutils.FileSize(logger.fd)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("fileSize failed, error is %sj, logger.latestFileId is %d", err, logger.fileId))
		}
	} else if err = logger.moveToNext(); err != nil {
		return nil, errors.New(fmt.Sprintf("logger.moveToNext failed, error is %s", err))
	}

	return logger, nil
}

func (logger *undoLogger) InsertBlock(blockHash *types.Hash, log []byte) {
	logger.pending[*blockHash] = log
}

func (logger *undoLogger) Flush(snapshotBlockHash *types.Hash, blockHashList []*types.Hash) (*chain_block.Location, error) {
	bufSize := 0
	for _, blockHash := range blockHashList {
		bufSize += len(logger.pending[*blockHash])
	}

	buf := make([]byte, 0, bufSize+types.HashSize+4)

	for _, blockHash := range blockHashList {
		buf = append(buf, logger.pending[*blockHash]...)
	}
	buf = append(buf, snapshotBlockHash.Bytes()...)
	buf = append(buf, chain_utils.Uint64ToFixedBytes(types.HashSize)...)

	freeSpaceLength := logger.maxFileSize - logger.fileSize

	if freeSpaceLength > 0 {
		if _, err := logger.fd.Write(buf[:freeSpaceLength]); err != nil {
			return nil, errors.New(fmt.Sprintf("logger.fd.Write failed, error is %s", err.Error()))
		}
	}

	if freeSpaceLength < int64(len(buf)) {
		logger.moveToNext()

		if _, err := logger.fd.Write(buf[freeSpaceLength:]); err != nil {
			return nil, errors.New(fmt.Sprintf("logger.fd.Write failed, error is %s", err.Error()))
		}
	}

	return chain_block.NewLocation(logger.fileId, logger.fileSize), nil
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

func (logger *undoLogger) LogFileIdList() ([]uint64, error) {
	allFilename, readErr := logger.dirFd.Readdirnames(0)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("logger.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	//filenameList := make([]string, 0, logger.logFileNum)
	fileIdList := make(Uint64Slice, 0, logger.logFileNum)
	for _, filename := range allFilename {
		if !logger.isLogFile(filename) {
			continue
		}
		fileId, err := logger.filenameToFileId(filename)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("logger.filenameToFileId failed, filename: %s. Error: %s",
				filename, readErr.Error()))
		}
		fileIdList = append(fileIdList, fileId)
	}

	sort.Sort(sort.Reverse(fileIdList))

	return fileIdList, nil
}

func (logger *undoLogger) ReadFile(fileId uint64) ([]byte, error) {
	fd, err := logger.getFd(fileId)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	readBuf := make([]byte, logger.maxFileSize)
	readN, err := fd.Read(readBuf)
	if err != nil {
		return nil, err
	}
	return readBuf[:readN], nil
}

func (logger *undoLogger) CompareLocation(location *chain_block.Location) int {
	return chain_block.NewLocation(logger.fileId, logger.fileSize).Compare(location)
}

func (logger *undoLogger) DeleteTo(location *chain_block.Location) error {
	// remove
	for i := logger.fileId; i > location.FileId()+1; i-- {
		if err := os.Remove(logger.fileIdToAbsoluteFilename(i)); err != nil {
			return err
		}

	}
	// Truncate
	if logger.fileId > location.FileId() {
		logger.fd.Close()

		logger.fileId = location.FileId()
		var err error
		logger.fd, err = logger.getFileFd(logger.fileId)
		if err != nil {
			return err
		}
	}
	logger.fileSize = location.Offset()

	if err := logger.fd.Truncate(logger.fileSize); err != nil {
		return err
	}

	return nil
}

func (logger *undoLogger) loadFileId() error {
	allFilename, readErr := logger.dirFd.Readdirnames(0)
	if readErr != nil {
		return errors.New(fmt.Sprintf("logger.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	minFileId := uint64(1)

	maxFileId := uint64(0)
	for _, filename := range allFilename {
		if !logger.isLogFile(filename) {
			continue
		}

		fileId, err := logger.filenameToFileId(filename)
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
	logger.fileId = maxFileId
	logger.minFileId = minFileId
	return nil
}

func (logger *undoLogger) removeOutDateLog() {
	if logger.fileId-logger.minFileId <= logger.logFileNum {
		return
	}

	filename := logger.fileIdToAbsoluteFilename(logger.minFileId)

	if err := os.Remove(filename); err != nil {
		if err == os.ErrNotExist {
			logger.minFileId += 1
			return
		}
		journalLog.Error(fmt.Sprintf("os.Remove failed, filename is %s. Error: %s, ", filename, err.Error()))
		return
	}
	logger.minFileId += 1
}

func (logger *undoLogger) moveToNext() error {
	nextFileId := logger.fileId + 1

	fd, err := logger.createNewFile(nextFileId)
	if err != nil {
		return errors.New(fmt.Sprintf("moveToNextFd failed, error is %s, nextFileId is %d", err, nextFileId))
	}

	if err := logger.fd.Close(); err != nil {
		errors.New(fmt.Sprintf("logger.fd.Close() failed, error is %s, latestFileId is %d", err, logger.fileId))
	}

	logger.fileId = nextFileId
	logger.fd = fd
	logger.fileSize = 0

	logger.removeOutDateLog()
	return nil
}

func (logger *undoLogger) getFileFd(fileId uint64) (*os.File, error) {
	absoluteFilename := logger.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}

func (logger *undoLogger) createNewFile(fileId uint64) (*os.File, error) {
	absoluteFilename := logger.fileIdToAbsoluteFilename(fileId)

	file, cErr := os.Create(absoluteFilename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	return file, nil
}
func (logger *undoLogger) fileIdToAbsoluteFilename(fileId uint64) string {
	return path.Join(logger.dirName, strconv.FormatUint(fileId, 10)+filenameSuffix)
}

func (logger *undoLogger) filenameToFileId(filename string) (uint64, error) {
	fileIdStr := filename[:len(filename)-filenameSuffixLength]
	return strconv.ParseUint(fileIdStr, 10, 64)

}

func (logger *undoLogger) isLogFile(filename string) bool {
	return strings.HasSuffix(filename, ".log")
}

func (logger *undoLogger) getFd(fileId uint64) (*os.File, error) {
	absoluteFilename := logger.fileIdToAbsoluteFilename(fileId)

	file, oErr := os.OpenFile(absoluteFilename, os.O_RDWR, 0666)
	if oErr != nil {
		return nil, errors.New(fmt.Sprintf("error is %s, fileId is %d, absoluteFilename is %s",
			oErr.Error(), fileId, absoluteFilename))
	}
	return file, oErr
}
