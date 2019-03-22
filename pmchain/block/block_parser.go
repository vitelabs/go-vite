package chain_block

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

const (
	BlockTypeUnknown       = byte(0)
	BlockTypeAccountBlock  = byte(1)
	BlockTypeSnapshotBlock = byte(2)
)

var ClosedErr = errors.New("blockFileParser is closed")

type byteBuffer struct {
	BlockType byte
	Buffer    []byte
}

type blockFileParser struct {
	blockSize              int
	blockSizeBuffer        []byte
	blockSizeBufferPointer int

	blockType byte

	blockBufferPointer int
	blockBuffer        []byte

	bytesBuffer chan *byteBuffer

	closed bool
	err    error
}

func newBlockFileParser() *blockFileParser {
	bp := &blockFileParser{
		blockSizeBuffer: make([]byte, 4),
		bytesBuffer:     make(chan *byteBuffer, 1000),
		closed:          false,
	}

	return bp
}

func (bfp *blockFileParser) Close() error {
	if bfp.closed {
		return ClosedErr
	}

	bfp.closed = true
	close(bfp.bytesBuffer)
	return nil
}

func (bfp *blockFileParser) WriteErr(err error) {
	bfp.err = err
}

func (bfp *blockFileParser) Write(buf []byte) (int, error) {
	if bfp.closed {
		return 0, ClosedErr
	}

	readPointer := 0
	bufLen := len(buf)

	for readPointer < bufLen {
		restLen := bufLen - readPointer

		if bfp.blockSizeBufferPointer < 4 {
			readNumbers := 4 - bfp.blockSizeBufferPointer

			if readNumbers > restLen {
				readNumbers = restLen
			}

			nextPointer := readPointer + readNumbers

			copy(bfp.blockSizeBuffer[bfp.blockSizeBufferPointer:], buf[readPointer:nextPointer])

			readPointer = nextPointer
			bfp.blockSizeBufferPointer += readNumbers

			if bfp.blockSizeBufferPointer >= 4 {
				bfp.blockSize = int(binary.BigEndian.Uint32(bfp.blockSizeBuffer) - 5)
			}
		} else if bfp.blockType == BlockTypeUnknown {

			bfp.blockType = buf[readPointer]
			readPointer += 1

		} else {
			readNumbers := bfp.blockSize - bfp.blockBufferPointer

			if readNumbers > restLen {
				if len(bfp.blockBuffer) <= 0 {
					bfp.blockBuffer = make([]byte, 0, bfp.blockSize)
				}
				bfp.blockBuffer = append(bfp.blockBuffer, buf[readPointer:]...)

				readPointer = bufLen
				bfp.blockBufferPointer += restLen
			} else {
				nextPointer := readPointer + readNumbers
				if len(bfp.blockBuffer) <= 0 {
					bfp.bytesBuffer <- &byteBuffer{
						BlockType: bfp.blockType,
						Buffer:    buf[readPointer:nextPointer],
					}
				} else {
					bfp.bytesBuffer <- &byteBuffer{
						BlockType: bfp.blockType,
						Buffer:    append(bfp.blockBuffer, buf[readPointer:nextPointer]...),
					}
				}

				readPointer = nextPointer

				bfp.blockSizeBufferPointer = 0
				bfp.blockType = BlockTypeUnknown

				bfp.blockBufferPointer = 0
				bfp.blockBuffer = nil
			}
		}
	}
	return bufLen, nil
}
func (bfp *blockFileParser) Iterator() <-chan *byteBuffer {
	return bfp.bytesBuffer
}

func (bfp *blockFileParser) Error() error {
	return bfp.err
}
