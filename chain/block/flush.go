package chain_block

import (
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

func (bDB *BlockDB) Id() types.Hash {
	return bDB.id
}

func (bDB *BlockDB) Prepare() {
	bDB.flushStartLocation = bDB.fm.NextFlushStartLocation()
	bDB.flushTargetLocation = bDB.fm.LatestLocation()
	bDB.fm.LockDelete()

}

func (bDB *BlockDB) CancelPrepare() {
	bDB.flushStartLocation = nil
	bDB.flushTargetLocation = nil
	bDB.fm.UnlockDelete()
}

type RedoLogWriter struct {
	RedoLog []byte
	Err     error
}

func (w *RedoLogWriter) Write(buf []byte) error {
	w.RedoLog = append(w.RedoLog, buf...)
	return nil
}
func (w *RedoLogWriter) WriteError(err error) {
	w.Err = err
}
func (*RedoLogWriter) Close() error {
	return nil
}

func (bDB *BlockDB) RedoLog() ([]byte, error) {
	var redoLog []byte
	redoLog = make([]byte, 0, 24+bDB.flushStartLocation.Distance(bDB.fileSize, bDB.flushTargetLocation))

	redoLog = append(redoLog, chain_utils.SerializeLocation(bDB.flushStartLocation)...)
	redoLog = append(redoLog, chain_utils.SerializeLocation(bDB.flushTargetLocation)...)

	writer := &RedoLogWriter{
		RedoLog: redoLog,
	}

	bDB.fm.ReadRange(bDB.flushStartLocation, bDB.flushTargetLocation, writer)

	if writer.Err != nil {
		return nil, writer.Err
	}

	return writer.RedoLog, nil

}

func (bDB *BlockDB) Commit() error {
	defer func() {
		bDB.flushStartLocation = nil
		bDB.flushTargetLocation = nil
		bDB.fm.UnlockDelete()
	}()

	return bDB.fm.Flush(bDB.flushStartLocation, bDB.flushTargetLocation)

}

func (bDB *BlockDB) PatchRedoLog(redoLog []byte) error {
	flushStartLocation := chain_utils.DeserializeLocation(redoLog[:12])
	flushTargetLocation := chain_utils.DeserializeLocation(redoLog[12:24])

	if err := bDB.fm.DeleteTo(flushStartLocation); err != nil {
		return err
	}

	if _, err := bDB.fm.Write(redoLog[24:]); err != nil {
		return err
	}

	return bDB.fm.Flush(flushStartLocation, flushTargetLocation)
}
