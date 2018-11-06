package compress

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"io"
)

type blockProcessor func(block ledger.Block, err error)

type blockParserCache struct {
	currentBlockSize       uint32
	currentBlockSizeBuffer []byte
	currentBlockType       byte
	currentBlockBuffer     []byte

	reader        io.Reader
	hasReadBlocks uint64
	processor     blockProcessor
}

func (blockParser *blockParserCache) RefreshCache() {
	blockParser.currentBlockSize = 0
	blockParser.currentBlockSizeBuffer = make([]byte, 0, 4)
	blockParser.currentBlockType = 0
	blockParser.currentBlockBuffer = make([]byte, 0, blockParser.currentBlockSize)
}

var blockParserLog = log15.New("module", "compress/block_parser")

var readNum = 1024 * 1024 * 10 // 10M

// If blockNum is zero, finish when stream encounter io.EOF
func BlockParser(reader io.Reader, blockNum uint64, processor blockProcessor) {
	blockParser := &blockParserCache{
		reader:        reader,
		processor:     processor,
		hasReadBlocks: 0,
	}

	blockParser.RefreshCache()

	readBytes := make([]byte, readNum)

	for {
		readN, rErr := reader.Read(readBytes)

		if rErr != nil && rErr != io.EOF {
			blockParserLog.Error("Read failed, error is " + rErr.Error())
			return
		}

		buffer := bytes.NewBuffer(readBytes[:readN])

		for buffer.Len() > 0 {
			if blockParser.currentBlockSize == 0 {

				readNum := 4 - len(blockParser.currentBlockSizeBuffer)

				sizeBytes := buffer.Next(readNum)
				blockParser.currentBlockSizeBuffer = append(blockParser.currentBlockSizeBuffer, sizeBytes...)

				if len(blockParser.currentBlockSizeBuffer) >= 4 {
					blockParser.currentBlockSize = binary.BigEndian.Uint32(blockParser.currentBlockSizeBuffer)
				}
			} else if blockParser.currentBlockSize != 0 && blockParser.currentBlockType == 0 {
				blockParser.currentBlockType = buffer.Next(1)[0]

			} else {
				readNum := blockParser.currentBlockSize - uint32(len(blockParser.currentBlockBuffer))

				blockBytes := buffer.Next(int(readNum))

				blockParser.currentBlockBuffer = append(blockParser.currentBlockBuffer, blockBytes...)

				if uint32(len(blockParser.currentBlockBuffer)) >= blockParser.currentBlockSize {
					var block ledger.Block
					switch blockParser.currentBlockType {
					case BlockTypeAccountBlock:
						block = &ledger.AccountBlock{}
					case BlockTypeSnapshotBlock:
						block = &ledger.SnapshotBlock{}

					}
					if block != nil {
						blockParser.hasReadBlocks++
						processor(block, block.Deserialize(blockParser.currentBlockBuffer))
					} else {
						processor(nil, errors.New("Unknown block type"))
					}

					blockParser.RefreshCache()
				}
			}
		}

		if rErr == io.EOF || (blockNum > 0 && blockParser.hasReadBlocks >= blockNum) {
			return
		}
	}
}
