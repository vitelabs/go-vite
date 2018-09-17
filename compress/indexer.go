package compress

import (
	"bufio"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const INDEX_SEP = ",,,"

type indexItem struct {
	startHeight uint64
	endHeight   uint64

	filename string
	fileSize uint64

	blockNumbers uint64
}

type Indexer struct {
	file      *os.File
	indexList []*indexItem
}

var indexerLog = log15.New("module", "compressor")

func NewIndexer(dir string) *Indexer {
	indexFileName := path.Join(dir, "index")
	var file *os.File
	var oErr error

	file, oErr = os.OpenFile(indexFileName, os.O_RDWR, 0666)
	if !os.IsExist(oErr) {
		var cErr error
		file, cErr = os.Create(indexFileName)

		if cErr != nil {
			indexerLog.Crit(cErr.Error(), "method", "NewIndexer")
		}
	}

	indexer := &Indexer{
		file: file,
	}

	if err := indexer.loadFromFile(); err != nil {
		indexerLog.Crit(err.Error(), "method", "NewIndexer")
	}

	return indexer
}

func (indexer *Indexer) Add(item *indexItem) error {

	return nil
}

func (indexer *Indexer) Delete() {

}

func (indexer *Indexer) loadFromFile() error {
	indexer.indexList = make([]*indexItem, 0)
	//indexer.file.Seek(0, io.SeekStart)
	reader := bufio.NewReader(indexer.file)

	for {
		var line []byte

		for {
			rLine, isPrefix, err := reader.ReadLine()

			if err != nil {
				if err != io.EOF {
					return err
				}
				break
			}

			line = append(line, rLine...)
			if !isPrefix {
				break
			}
		}

		if len(line) >= 0 {
			indexerLog.Info(string(line))
		}

		item, parseErr := indexer.parseLine(line)
		if parseErr != nil {
			return parseErr
		}

		indexer.indexList = append(indexer.indexList, item)
	}

	return nil
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

	fileSize, err3 := strconv.ParseUint(segs[3], 10, 64)
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

func (indexer *Indexer) formatToLine(item *indexItem) []byte {
	lineString := ""
	lineString += strconv.FormatUint(item.startHeight, 10) + INDEX_SEP
	lineString += strconv.FormatUint(item.endHeight, 10) + INDEX_SEP
	lineString += item.filename + INDEX_SEP
	lineString += strconv.FormatUint(item.fileSize, 10) + INDEX_SEP
	lineString += strconv.FormatUint(item.blockNumbers, 10)
	return []byte(lineString)
}
