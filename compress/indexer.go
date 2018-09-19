package compress

import (
	"bufio"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const INDEX_SEP = ",,,"

type indexItem struct {
	startHeight uint64
	endHeight   uint64

	filename string
	fileSize int64

	blockNumbers uint64
}

type Indexer struct {
	file      *os.File
	indexList []*indexItem
	log       log15.Logger

	dir string
}

func NewIndexer(dir string) *Indexer {
	indexFileName := path.Join(dir, "index")
	var file *os.File
	var oErr error

	indexer := &Indexer{
		log: log15.New("module", "compressor/index"),
		dir: dir,
	}

	file, oErr = os.OpenFile(indexFileName, os.O_RDWR, 0666)
	if !os.IsExist(oErr) {
		var cErr error
		file, cErr = os.Create(indexFileName)

		if cErr != nil {
			indexer.log.Crit(cErr.Error(), "method", "NewIndexer")
		}
	}

	indexer.file = file

	if parsedSize, err := indexer.loadFromFile(); err != nil {
		indexer.log.Error("loadFromFile failed, error is "+err.Error(), "method", "NewIndexer")
		if tErr := indexer.truncate(parsedSize); tErr != nil {
			indexer.log.Crit("truncate failed, error is "+tErr.Error(), "method", "NewIndexer")
		}
	}

	return indexer
}

func (indexer *Indexer) getFileSize(filename string) (int64, error) {
	file, openErr := os.Open(filename)
	if openErr != nil {
		indexer.log.Error("Open new file error, error is "+openErr.Error(), "method", "Add")
		return 0, openErr
	}

	// Get stat
	fileInfo, statErr := file.Stat()
	if statErr != nil {
		indexer.log.Error("File stat error, error is "+statErr.Error(), "method", "Add")
		return 0, statErr
	}

	fileSize := fileInfo.Size()

	// Close
	if closeErr := file.Close(); closeErr != nil {
		indexer.log.Error("Close failed, error is "+closeErr.Error(), "method", "Add")
		return fileSize, closeErr
	}
	return fileSize, nil
}

func (indexer *Indexer) newFileName(startHeight uint64, endHeight uint64) string {
	return filepath.Join(indexer.dir, "subgraph_"+strconv.FormatUint(startHeight, 10)+"_"+strconv.FormatUint(endHeight, 10))
}

func (indexer *Indexer) check(item *indexItem) (bool, error) {
	// Rename err
	fileSize, fileSizeErr := indexer.getFileSize(item.filename)

	if fileSizeErr != nil {
		indexer.log.Error("getFileSize failed, error is "+fileSizeErr.Error(), "method", "Add")
		return false, fileSizeErr
	}

	if fileSize != item.fileSize {
		return false, nil
	}
	return true, nil
}

func (indexer *Indexer) checkDataFile() (bool, error) {
	// Rename err
	//fileSize, fileSizeErr := indexer.getFileSize(item.filename)
	//
	//if fileSizeErr != nil {
	//	indexer.log.Error("getFileSize failed, error is "+fileSizeErr.Error(), "method", "Add")
	//	return false, fileSizeErr
	//}
	//
	//if fileSize != item.fileSize {
	//	return false, nil
	//}
	//return true, nil
}

func (indexer *Indexer) delete(filename string) {
	os.Remove(filename)
}

func (indexer *Indexer) getAllFileNames() []string {
	dir, openErr := os.Open(indexer.dir)
	if openErr != nil {
		indexer.log.Error("Open failed, error is "+openErr.Error(), "method", "getAllFileNames")
		return nil
	}
	defer dir.Close()

	allFileName, readErr := dir.Readdirnames(0)
	if readErr != nil {
		indexer.log.Error("Readdirnames failed, error is "+readErr.Error(), "method", "getAllFileNames")
		return nil
	}

	return allFileName
}

func (indexer *Indexer) truncate(n int64) error {
	return indexer.file.Truncate(n)
}

func (indexer *Indexer) loadFromFile() (int64, error) {
	indexer.indexList = make([]*indexItem, 0)
	//indexer.file.Seek(0, io.SeekStart)
	reader := bufio.NewReader(indexer.file)

	var hasParsedSize = int64(0)
	for {
		var line []byte

		for {
			rLine, isPrefix, err := reader.ReadLine()

			if err != nil {
				if err != io.EOF {
					indexer.log.Error("ReadLine failed, error is "+err.Error(), "method", "loadFromFile")
					return hasParsedSize, err
				}
				break
			}

			line = append(line, rLine...)

			if !isPrefix {
				break
			}
		}

		if len(line) >= 0 {
			indexer.log.Info(string(line))
		}

		item, parseErr := indexer.parseLine(line)
		if parseErr != nil {
			indexer.log.Error("ParseLine failed, error is "+parseErr.Error(), "method", "loadFromFile")
			return hasParsedSize, parseErr
		}
		if isCorrect, err := indexer.check(item); !isCorrect {
			if err != nil {
				indexer.log.Error("Check failed, error is "+err.Error(), "method", "loadFromFile")
			}

			return hasParsedSize, err
		}

		hasParsedSize += int64(len(line))

		indexer.indexList = append(indexer.indexList, item)
	}

	return hasParsedSize, nil
}

func (indexer *Indexer) parseLine(line []byte) (*indexItem, error) {
	segs := strings.Split(string(line), INDEX_SEP)

	startHeight, err1 := strconv.ParseUint(segs[0], 10, 64)
	if err1 != nil {
		return nil, err1
	}

	endHeight, err2 := strconv.ParseUint(segs[1], 10, 64)
	if err1 != nil {
		return nil, err2
	}

	filename := segs[2]

	fileSize, err3 := strconv.ParseInt(segs[3], 10, 64)
	if err3 != nil {
		return nil, err3
	}

	blockNumbers, err4 := strconv.ParseUint(segs[4], 10, 64)
	if err4 != nil {
		return nil, err4
	}

	item := &indexItem{
		startHeight:  startHeight,
		endHeight:    endHeight,
		filename:     filename,
		fileSize:     fileSize,
		blockNumbers: blockNumbers,
	}
	return item, nil
}

func (indexer *Indexer) formatToLine(item *indexItem) string {
	lineString := ""
	lineString += strconv.FormatUint(item.startHeight, 10) + INDEX_SEP
	lineString += strconv.FormatUint(item.endHeight, 10) + INDEX_SEP
	lineString += item.filename + INDEX_SEP
	lineString += strconv.FormatInt(item.fileSize, 10) + INDEX_SEP
	lineString += strconv.FormatUint(item.blockNumbers, 10)
	return lineString
}

func (indexer *Indexer) LatestHeight() uint64 {
	if lastItem := indexer.indexList[len(indexer.indexList)-1]; lastItem != nil {
		return lastItem.endHeight
	}
	return 0
}

func (indexer *Indexer) Add(ti *taskInfo, tmpFile string, blockNumbers uint64) error {
	if lastItem := indexer.indexList[len(indexer.indexList)-1]; lastItem != nil {
		if ti.beginHeight != lastItem.endHeight+1 {
			indexer.log.Error("ti.beginHeight != lastItem.endHeight + 1", "method", "Add")
			return nil
		}
	}

	newFileName := indexer.newFileName(ti.beginHeight, ti.targetHeight)

	// Rename err
	fileSize, fileSizeErr := indexer.getFileSize(newFileName)

	// Close
	if fileSizeErr != nil {
		indexer.log.Error("getFileSize failed, error is "+fileSizeErr.Error(), "method", "Add")
		return fileSizeErr
	}

	newItem := &indexItem{
		startHeight: ti.beginHeight,
		endHeight:   ti.targetHeight,

		filename: newFileName,

		fileSize:     fileSize,
		blockNumbers: blockNumbers,
	}

	_, writeErr := indexer.file.WriteString(indexer.formatToLine(newItem) + "\n")

	if writeErr != nil {
		return writeErr
	}

	indexer.indexList = append(indexer.indexList, newItem)
	return nil
}
