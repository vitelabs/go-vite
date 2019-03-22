package chain_block

import (
	"encoding/binary"
	"github.com/golang/snappy"
)

func makeWriteBytes(buf []byte, dataType byte, data []byte) []byte {
	buf[4] = dataType
	sBuf := snappy.Encode(buf[5:], data)

	bufSize := 5 + len(sBuf)
	binary.BigEndian.PutUint32(buf, uint32(bufSize))

	return buf[:bufSize]
}
