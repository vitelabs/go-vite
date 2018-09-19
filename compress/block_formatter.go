package compress

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	"io"
)

type block interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

type blocksGetter func(uint64, uint64) ([]block, error)

var blockFormatterLog = log15.New("module", "compress", "block_formatter")

func BlockFormatter(writer io.Writer, getter blocksGetter) error {
	hasWrite := uint64(0)
	hasWriteBlocks := uint64(0)
	for {
		blocks, gErr := getter(hasWrite, hasWriteBlocks)
		if gErr != nil && gErr != io.EOF {
			blockFormatterLog.Error("Read failed, error is " + gErr.Error())
			return gErr
		}

		for _, block := range blocks {
			blockBytes, sErr := block.Serialize()
			if sErr != nil {
				blockFormatterLog.Error("Serialize failed, error is " + sErr.Error())
				return sErr
			}

			size := uint32(len(blockBytes))
			if size == 0 {
				continue
			}
			sizeBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(sizeBytes, size)

			needWrite := make([]byte, 4+size)
			needWrite = append(needWrite, sizeBytes...)
			needWrite = append(needWrite, blockBytes...)

			writtenSize := 0
			needWriteLen := len(needWrite)
			for writtenSize < needWriteLen {
				n, wErr := writer.Write(needWrite[writtenSize:needWriteLen])
				if wErr != nil && wErr != io.ErrShortWrite {
					blockFormatterLog.Error("Write block bytes failed, error is " + wErr.Error())
					return wErr
				}
				writtenSize += n
			}

			hasWrite += uint64(needWriteLen)
			hasWriteBlocks++
		}

		if gErr == io.EOF {
			return nil
		}
	}
}
