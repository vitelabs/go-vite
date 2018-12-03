package rocket

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/vm_context"
)

var blockId = uint64(0)

func (c *chain) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error {
	blockId += 1
	blockIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockIdBytes, blockId)

	for _, vmBlock := range vmAccountBlocks {
		blockBytes, err := vmBlock.AccountBlock.DbSerialize()
		if err != nil {
			return err
		}
		if err := c.chainDb.Db().Put(blockIdBytes, blockBytes, nil); err != nil {
			return err
		}
	}
	return nil
}
