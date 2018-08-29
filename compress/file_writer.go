package compress

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var fileWriterLog = log15.New("module", "file_writer")

type Block interface {
	FileSerialize() ([]byte, error)
	FileDeSerialize([]byte) (Block, error)
}

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

func (fw *FileWriter) WriteBlock(block Block) error {
	buffer, err := block.FileSerialize()
	if err != nil {
		return err
	}

	size := uint32(len(buffer))
	sizeBytes := make([]byte, 4)

	binary.LittleEndian.PutUint32(sizeBytes, size)

	result := make([]byte, 4+size)
	result = append(result, sizeBytes...)
	result = append(result, buffer...)

	fw.file.Write(result)
	return nil
}

func (fw *FileWriter) Close() {
	fw.file.Close()
	return
}
