package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var fileWriterLog = log15.New("module", "file_writer")

func NewFileWriter(filename string) io.WriteCloser {
	file, err := os.Create(filename)

	if err != nil {
		fileWriterLog.Error("Create file failed, error is " + err.Error())
		return nil
	}

	return file
}
