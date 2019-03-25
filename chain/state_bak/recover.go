package chain_state_bak

//import (
//	"github.com/vitelabs/go-vite/chain/block"
//	"github.com/vitelabs/go-vite/common/types"
//)
//
//func (sDB *StateDB) CheckAndDelete(location *chain_block.Location) error {
//	logFileIdList, err := sDB.flushLog.LogFileIdList()
//	if err != nil {
//		return err
//	}
//
//	var undoBlockHashList []*types.Hash
//LOOP:
//	for _, logFileId := range logFileIdList {
//		buf, err := sDB.flushLog.ReadFile(logFileId)
//
//		if err != nil {
//			return err
//		}
//		currentPointer := len(buf)
//
//		for currentPointer >= types.HashSize {
//			hashBytes := buf[currentPointer-types.HashSize : currentPointer]
//			hash, err := types.BytesToHash(hashBytes)
//			if err != nil {
//				return err
//			}
//
//			ok, err := sDB.chain.IsAccountBlockExisted(&hash)
//			if err != nil {
//				return err
//			}
//			if !ok {
//				undoBlockHashList = append(undoBlockHashList, &hash)
//			} else {
//				break LOOP
//			}
//		}
//	}
//	if len(undoBlockHashList) > 0 {
//		if err := sDB.mvDB.Undo(undoBlockHashList, location); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
