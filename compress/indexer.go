package compress

import (
	"bufio"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"math/big"
	"os"
	"path"
)

type indexItem struct {
	startHeight *big.Int
	endHeight   *big.Int

	filename string
	filesize uint64

	blockNumbers uint64
}

type Indexer struct {
	file      *os.File
	indexList []*indexItem
}

var indexerLog = log15.New("module", "indexer")

func NewIndexer(dir string) *Indexer {
	indexFileName := path.Join(dir, "index")
	var file *os.File
	var oErr error

	file, oErr = os.OpenFile(indexFileName, os.O_RDWR, 0666)
	if !os.IsExist(oErr) {
		var cErr error
		file, cErr = os.Create(indexFileName)

		if cErr != nil {
			indexerLog.Crit(cErr.Error())
		}
	}

	indexer := &Indexer{
		file: file,
	}
	return indexer
}

func (indexer *Indexer) readFromFile() {
	indexer.indexList = make([]*indexItem, 0)
	indexer.file.Seek(0, io.SeekStart)

	reader := bufio.NewReader(indexer.file)
	for {
		var line []byte

		for {
			rLine, isPrefix, err := reader.ReadLine()
			line = append(line, rLine...)
			if !isPrefix {
				break
			}
		}
		//for ; ; {
		//	//
		//}

	}
}
