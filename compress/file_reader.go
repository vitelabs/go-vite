package compress

import (
	"io"
	"os"

	"github.com/vitelabs/go-vite/log15"
)

var fileReaderLog = log15.New("module", "file_reader")

func NewFileReader(filename string) (io.ReadCloser, error) {
	file, err := os.Open(filename)

	if err != nil {
		fileReaderLog.Error("Open file failed, error is " + err.Error())
		return nil, err
	}

	return file, nil
}
