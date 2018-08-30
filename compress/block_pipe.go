package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
)

type BlockProcessor interface {
	Process(block Block) bool
}

type BlockParser struct {
	buffer []byte
}

var blockPipeLog = log15.New("module", "file_reader")

func NewBlockParser(reader io.Reader, processor BlockProcessor) {
	//r, w := io.Pipe()

	readLimit := 1024 * 1024
	readBuffer := make([]byte, readLimit, readLimit)

	for {
		_, err := reader.Read(readBuffer)
		if err != nil {
			if err != io.EOF {

			}
		}
	}

}

type BlockPipeReader struct {
}

func NewBlockPipeReader() io.Reader {
	return &BlockPipeReader{}
}

func (bpr *BlockPipeReader) Read(b []byte) (int, error) {

}

func (bpr *BlockPipeReader) ReadBlock(block Block) {

}

type BlockPipeWriter struct {
	buffer []byte
	blocks []Block
}

func NewBlockPipeWriter() *BlockPipeWriter {
	bufferLimit := 1024 * 1024
	buffer := make([]byte, bufferLimit, bufferLimit)
	return &BlockPipeWriter{
		buffer: buffer,
	}
}

func (bpw *BlockPipeWriter) WriteBlock() {

}

func (bpw *BlockPipeWriter) Write(b []byte) (int, error) {
	bpw.buffer = append(bpw.buffer, b)
}
