package compress

import (
	"testing"
)

func testTaskInfoSplit(t *testing.T, task *taskInfo, gap uint64) {
	subTaskList1 := task.Split(gap)
	for index, subTask := range subTaskList1 {
		beginHeight := task.beginHeight + gap*uint64(index)

		targetHeight := beginHeight + gap - 1

		if targetHeight > task.targetHeight {
			targetHeight = task.targetHeight
		}

		if subTask.beginHeight != beginHeight {
			t.Errorf("Not Valid, index is %d, subTask.beginHeight is %d, beginHeight is %d", index, subTask.beginHeight, beginHeight)
		}
		if subTask.targetHeight != targetHeight {
			t.Errorf("Not Valid, index is %d, subTask.beginHeight is %d, beginHeight is %d", index, subTask.targetHeight, targetHeight)
		}
	}
}

func TestTaskInfo_Split(t *testing.T) {

	task := &taskInfo{
		beginHeight:  12,
		targetHeight: 180,
	}
	//
	testTaskInfoSplit(t, task, 10)
	testTaskInfoSplit(t, task, 5)
	testTaskInfoSplit(t, task, 1000)

	task2 := &taskInfo{
		beginHeight:  1,
		targetHeight: 5,
	}

	testTaskInfoSplit(t, task2, 10)
	testTaskInfoSplit(t, task2, 12)
	testTaskInfoSplit(t, task2, 1)

	task3 := &taskInfo{
		beginHeight:  10,
		targetHeight: 50,
	}

	testTaskInfoSplit(t, task3, 5)
	testTaskInfoSplit(t, task3, 8)
	testTaskInfoSplit(t, task3, 21)
}
