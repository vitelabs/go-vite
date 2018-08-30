package compress

import (
	"bytes"
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	"io"
)

type blockProcessor func([]byte)

type blockParserCache struct {
	currentBlockSize       uint32
	currentBlockSizeBuffer []byte
	currentBlockBuffer     []byte

	reader    io.Reader
	processor blockProcessor
}

var blockParserLog = log15.New("module", "compress", "block_parser")

func BlockParser(reader io.Reader, processor blockProcessor) {
	//r, w := io.Pipe()

	readNum := 1024 * 1024 * 10 // 10M

	blockParser := &blockParserCache{
		reader:    reader,
		processor: processor,
	}

	for {
		readBytes := make([]byte, readNum)
		_, rErr := reader.Read(readBytes)

		if rErr != nil && rErr != io.EOF {
			blockParserLog.Error("Read failed, error is " + rErr.Error())
			return
		}

		buffer := bytes.NewBuffer(readBytes)

		for buffer.Len() > 0 {
			if buffer.Len() != 0 &&
				blockParser.currentBlockSize == 0 {

				if len(blockParser.currentBlockSizeBuffer) == 0 {
					blockParser.currentBlockSizeBuffer = make([]byte, 0, 4)
				}

				readNum := 4 - len(blockParser.currentBlockSizeBuffer)

				sizeBytes := buffer.Next(readNum)
				blockParser.currentBlockSizeBuffer = append(blockParser.currentBlockSizeBuffer, sizeBytes...)

				if len(blockParser.currentBlockSizeBuffer) >= 4 {
					blockParser.currentBlockSize = binary.LittleEndian.Uint32(blockParser.currentBlockSizeBuffer)
				}
			}

			if buffer.Len() != 0 && blockParser.currentBlockSize != 0 {
				if len(blockParser.currentBlockBuffer) == 0 {
					blockParser.currentBlockBuffer = make([]byte, 0, blockParser.currentBlockSize)
				}
				readNum := blockParser.currentBlockSize - uint32(len(blockParser.currentBlockBuffer))

				blockBytes := buffer.Next(int(readNum))

				blockParser.currentBlockBuffer = append(blockParser.currentBlockBuffer, blockBytes...)

				if uint32(len(blockParser.currentBlockBuffer)) >= blockParser.currentBlockSize {
					processor(blockParser.currentBlockBuffer)
				}
			}
		}

		if rErr == io.EOF {
			return
		}
	}
}
