package compress

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"os"
)

var compressorTaskLog = log15.New("module", "compressorTask")

type taskInfo struct {
	beginHeight  *big.Int
	targetHeight *big.Int
}

func (ti *taskInfo) Split(gap int64) []*taskInfo {
	gapBigInt := big.NewInt(gap)
	oneBigInt := big.NewInt(1)

	tiList := []*taskInfo{}

	segStartHeight := &big.Int{}
	segStartHeight.Set(ti.beginHeight)

	for segStartHeight.Cmp(ti.targetHeight) <= 0 {
		segEndHeight := &big.Int{}
		segEndHeight.Add(segStartHeight, gapBigInt)
		segEndHeight.Sub(segEndHeight, oneBigInt)
		if segEndHeight.Cmp(ti.targetHeight) > 0 {
			segEndHeight.Set(ti.targetHeight)
		}
		tiList = append(tiList, &taskInfo{
			beginHeight:  segStartHeight,
			targetHeight: segEndHeight,
		})

		segStartHeight = &big.Int{}
		segStartHeight.Add(segEndHeight, oneBigInt)
	}

	return tiList
}

type subLedger struct {
	sbList []*ledger.SnapshotBlock
	abList []*ledger.AccountBlock
}

func (sl *subLedger) Write(file *os.File) {

}

type CompressorTask struct {
	splitSize int64
	tmpFile   *os.File
}

func NewCompressorTask() *CompressorTask {
	compressorTask := &CompressorTask{
		splitSize: 10,
	}

	return compressorTask
}

func (task *CompressorTask) Run() {
	// Get task info
	var ti *taskInfo
	if ti = task.getTaskInfo(); ti == nil {
		return
	}

	taskInfoList := ti.Split(task.splitSize)
	for _, taskInfo := range taskInfoList {
		subLedger, getSubLedgerErr := task.getSubLedger(taskInfo)
		if getSubLedgerErr != nil {
			compressorTaskLog.Error(getSubLedgerErr.Error())
			// todo: Clear tmp file
			break
		}

		subLedger.Write(task.tmpFile)
	}

	task.writeIndex()
}

func (task *CompressorTask) getTaskInfo() *taskInfo {
	return &taskInfo{}
}

func (task *CompressorTask) getSubLedger(ti *taskInfo) (*subLedger, error) {
	return &subLedger{
		sbList: []*ledger.SnapshotBlock{},
		abList: []*ledger.AccountBlock{},
	}, nil
}

func (task *CompressorTask) writeIndex() {

}

type fileIndex struct {
	startHeight *big.Int
	endHeight   *big.Int

	filename string
	filesize uint64

	blockNumbers uint64
}
