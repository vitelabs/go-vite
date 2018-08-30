package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var fileWriterLog = log15.New("module", "file_writer")

type FileWriter struct {
	file *os.File
}

func NewFileWriter(filename string) io.Writer {
	file, err := os.Create(filename)

	if err != nil {
		fileWriterLog.Error("Create file failed, error is " + err.Error())
		return nil
	}

	return &FileWriter{
		file: file,
	}
}

func (fw *FileWriter) Write(data []byte) (int, error) {
	return fw.file.Write(data)
}

func (fw *FileWriter) Close() {
	fw.file.Close()
	return
}
