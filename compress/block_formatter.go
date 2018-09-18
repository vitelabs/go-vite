package compress

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	"io"
)

type block interface {
	FileSerialize() ([]byte, error)
	FileDeSerialize([]byte) (block, error)
}

type blocksGetter func() ([]block, error)

var blockFormatterLog = log15.New("module", "compress", "block_formatter")

func BlockFormatter(writer io.Writer, getter blocksGetter) error {
	for {
		blocks, gErr := getter()
		if gErr != nil && gErr != io.EOF {
			blockFormatterLog.Error("Read failed, error is " + gErr.Error())
			return gErr
		}

		for _, block := range blocks {
			blockBytes, sErr := block.FileSerialize()
			if sErr != nil {
				blockFormatterLog.Error("Serialize failed, error is " + sErr.Error())
				return sErr
			}

			size := uint32(len(blockBytes))
			if size == 0 {
				continue
			}
			sizeBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(sizeBytes, size)

			needWrite := make([]byte, 4+size)
			needWrite = append(needWrite, sizeBytes...)
			needWrite = append(needWrite, blockBytes...)

			writtenSize := 0
			for writtenSize < len(needWrite) {
				n, wErr := writer.Write(needWrite)
				if wErr != nil && wErr != io.ErrShortWrite {
					blockFormatterLog.Error("Write block bytes failed, error is " + wErr.Error())
					return wErr
				}
				writtenSize += n
			}

		}

		if gErr == io.EOF {
			return nil
		}
	}
}
