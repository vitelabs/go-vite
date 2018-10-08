package compress

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
)

var compressorTaskLog = log15.New("module", "compressorTask")

const writeMax = 1024 * 1024 * 200 // 200M

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
	splitSize      uint64
	tmpFile        string
	chain          Chain
	indexerHeight  uint64
	startHeightGap uint64
	taskGap        uint64
	log            log15.Logger
}

func NewCompressorTask(chain Chain, tmpFile string, indexerHeight uint64) *CompressorTask {
	compressorTask := &CompressorTask{
		splitSize: 10,
		chain:     chain,
		tmpFile:   tmpFile,
		log:       log15.New("module", "compressor/task"),

		indexerHeight:  indexerHeight,
		startHeightGap: 7200,
		taskGap:        3600,
	}

	return compressorTask
}

type TaskRunResult struct {
	Ti           *taskInfo
	IsSuccess    bool
	BlockNumbers uint64
}

func (task *CompressorTask) Run() *TaskRunResult {
	// Get task info
	var ti *taskInfo
	if ti = task.getTaskInfo(); ti == nil {
		return &TaskRunResult{
			Ti:           ti,
			IsSuccess:    false,
			BlockNumbers: 0,
		}
	}

	taskInfoList := ti.Split(task.splitSize)

	taskLen := len(taskInfoList)
	currentTaskIndex := 0

	tmpFileWriter := NewFileWriter(task.tmpFile)
	var blockNumbers = uint64(0)

	// Limit write length
	formatterErr := BlockFormatter(tmpFileWriter, func(hasWrite uint64, hasWriteBlocks uint64) ([]ledger.Block, error) {

		if currentTaskIndex >= taskLen ||
			hasWrite >= writeMax {
			return nil, io.EOF
		}

		blocks, err := task.getSubLedger(taskInfoList[currentTaskIndex])

		currentTaskIndex++
		blockNumbers += uint64(len(blocks))

		return blocks, err
	})

	tmpFileWriter.Close()

	if formatterErr != nil {
		task.log.Error("Block write failed, error is "+formatterErr.Error(), "method", "Run")
		return &TaskRunResult{
			Ti:           ti,
			IsSuccess:    false,
			BlockNumbers: 0,
		}
	}

	return &TaskRunResult{
		Ti:           ti,
		IsSuccess:    true,
		BlockNumbers: blockNumbers,
	}
}

func (task *CompressorTask) Clear() {
	os.Remove(task.tmpFile)
}

func (task *CompressorTask) getTaskInfo() *taskInfo {
	latestSnapshotBlock := task.chain.GetLatestSnapshotBlock()

	if latestSnapshotBlock.Height-task.indexerHeight > task.startHeightGap {
		ti := &taskInfo{
			beginHeight: task.indexerHeight + 1,
		}
		ti.targetHeight = ti.beginHeight + task.taskGap - 1
		return ti
	}
	return nil
}

func (task *CompressorTask) getSubLedger(ti *taskInfo) ([]ledger.Block, error) {
	snapshotBlocks, accountChainSubLedger, err := task.chain.GetConfirmSubLedger(ti.beginHeight, ti.targetHeight)
	if err != nil {
		return nil, err
	}

	blocks := make([]ledger.Block, 0)
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
