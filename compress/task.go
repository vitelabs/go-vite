package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var compressorTaskLog = log15.New("module", "compressorTask")

type taskInfo struct {
	beginHeight  uint64
	targetHeight uint64
}

func (ti *taskInfo) Split(gap uint64) []*taskInfo {
	tiList := make([]*taskInfo, 0)

	segStartHeight := ti.beginHeight

	for segStartHeight <= ti.targetHeight {
		segEndHeight := segStartHeight + gap - 1
		if segEndHeight > ti.targetHeight {
			segEndHeight = ti.targetHeight
		}

		tiList = append(tiList, &taskInfo{
			beginHeight:  segStartHeight,
			targetHeight: segEndHeight,
		})

		segStartHeight = segEndHeight + 1
	}

	return tiList
}

type CompressorTask struct {
	splitSize uint64
	tmpFile   string
	chain     Chain
	log       log15.Logger
}

func NewCompressorTask(chain Chain, tmpFile string) *CompressorTask {
	compressorTask := &CompressorTask{
		splitSize: 10,
		chain:     chain,
		tmpFile:   tmpFile,
		log:       log15.New("module", "compressor/task"),
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

	tmpFileWriter := NewFileWriter(task.tmpFile)
	formatterErr := BlockFormatter(tmpFileWriter, func() ([]block, error) {
		if currentTaskIndex > taskLen {
			return nil, io.EOF
		}

		blocks, err := task.getSubLedger(taskInfoList[currentTaskIndex])
		currentTaskIndex++
		return blocks, err
	})
	tmpFileWriter.Close()

	if formatterErr != nil {
		os.Remove(task.tmpFile)
		task.log.Error("Block write failed, error is "+formatterErr.Error(), "method", "Run")
		return
	}

	// todo: Write index

}

func (task *CompressorTask) getTaskInfo() *taskInfo {
	return &taskInfo{}
}

func (task *CompressorTask) getSubLedger(ti *taskInfo) ([]block, error) {
	snapshotBlocks, accountChainSubLedger, err := task.chain.GetConfirmSubLedger(ti.beginHeight, ti.targetHeight)
	if err != nil {
		return nil, err
	}

	blocks := make([]block, 0)
	for _, snapshotBlock := range snapshotBlocks {
		blocks = append(blocks, snapshotBlock)
	}

	for _, accountChain := range accountChainSubLedger {
		for _, accountBlock := range accountChain {
			blocks = append(blocks, accountBlock)
		}
	}
	return blocks, nil
}

func (task *CompressorTask) writeIndex() {

}
