package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var fileReaderLog = log15.New("module", "file_reader")

func NewFileReader(filename string) io.ReadCloser {
	file, err := os.Open(filename)

	if err != nil {
		fileReaderLog.Error("Open file failed, error is " + err.Error())
		return nil
	}

	return file
}
