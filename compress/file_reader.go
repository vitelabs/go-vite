package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var fileReaderLog = log15.New("module", "file_reader")

type FileReader struct {
	file *os.File
}

func NewFileReader(filename string) io.Reader {
	file, err := os.Open(filename)

	if err != nil {
		fileReaderLog.Error("Open file failed, error is " + err.Error())
		return nil
	}

	return file
}

func (fw *FileReader) Read(p []byte) (int, error) {
	return fw.file.Read(p)
}

func (fw *FileReader) Close() {
	fw.file.Close()
	return
}
