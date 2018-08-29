package compress

import (
	"encoding/binary"
	"errors"
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

func (fw *FileReader) ReadBlock(block Block) error {
	sizeBytes := make([]byte, 4)
	sizeN, sizeErr := fw.file.Read(sizeBytes)

	if sizeErr != nil {
		return sizeErr
	}

	if sizeN < 4 {
		err := errors.New("read block failed, file format is not correct. sizeN is not correct")
		fileReaderLog.Error(err.Error())
		return nil
	}

	size := binary.LittleEndian.Uint32(sizeBytes)

	buffer := make([]byte, size)
	bufferN, bufferErr := fw.file.Read(sizeBytes)

	if bufferErr != nil {
		return bufferErr
	}

	if uint32(bufferN) < size {
		err := errors.New("read block failed, file format is not correct. bufferN is not correct")
		fileReaderLog.Error(err.Error())
	}

	block.FileDeSerialize(buffer)

	return nil
}

func (fw *FileReader) Close() {
	fw.file.Close()
	return
}
