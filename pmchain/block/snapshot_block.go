package chain_block

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
)

func (bDB *BlockDB) GetSnapshotBlock(location *Location) (*ledger.SnapshotBlock, error) {
	fd, err := bDB.fm.GetFd(location)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("bDB.fm.GetFd failed, [Error] %s, location is %+v", err.Error(), location))
	}
	if fd == nil {
		return nil, errors.New(fmt.Sprintf("fd is nil, location is %+v", location))
	}
	defer fd.Close()

	if _, err := fd.Seek(int64(location.Offset()-1), 0); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Seek failed, [Error] %s, location is %+v", err.Error(), location))
	}

	bufSizeBytes := make([]byte, 4)
	if _, err := fd.Read(bufSizeBytes); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}
	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize)
	if _, err := fd.Read(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("fd.Read failed, [Error] %s", err.Error()))
	}

	sb := &ledger.SnapshotBlock{}
	if err := sb.Deserialize(buf); err != nil {
		return nil, errors.New(fmt.Sprintf("sb.Deserialize failed, [Error] %s", err.Error()))
	}

	return sb, nil
}

// TODO optimize
func (bDB *BlockDB) GetSnapshotHeader(location *Location) (*ledger.SnapshotBlock, error) {
	sb, err := bDB.GetSnapshotBlock(location)
	if err != nil {
		return nil, err
	}
	sb.SnapshotContent = nil
	return sb, nil
}
