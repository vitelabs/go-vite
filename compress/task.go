package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"math/big"
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

type CompressorTask struct {
	splitSize int64
	tmpFile   string
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

	taskLen := len(taskInfoList)
	currentTaskIndex := 0

	formatterErr := BlockFormatter(NewFileWriter(task.tmpFile), func() ([]block, error) {
		if currentTaskIndex > taskLen {
			return nil, io.EOF
		}

		return task.getSubLedger(taskInfoList[currentTaskIndex])
	})

	if formatterErr != nil {
		// todo: Clear tmp file
		return
	}

	// todo: Write db index

}

func (task *CompressorTask) getTaskInfo() *taskInfo {
	return &taskInfo{}
}

func (task *CompressorTask) getSubLedger(ti *taskInfo) ([]block, error) {
	return nil, nil
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
