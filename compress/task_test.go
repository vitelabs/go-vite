package compress

import (
	"testing"
)

func testTaskInfoSplit(t *testing.T, task *taskInfo, gap int64) {
	//subTaskList1 := task.Split(gap)
	//for index, subTask := range subTaskList1 {
	//	beginHeight := &big.Int{}
	//	beginHeight.Set(task.beginHeight)
	//
	//	targetHeight := &big.Int{}
	//
	//	addBigInt := big.NewInt(gap * int64(index))
	//
	//	beginHeight.Add(beginHeight, addBigInt)
	//
	//	targetHeight.Add(beginHeight, big.NewInt(gap))
	//	targetHeight.Sub(targetHeight, big.NewInt(1))
	//
	//	if targetHeight.Cmp(task.targetHeight) > 0 {
	//		targetHeight.Set(task.targetHeight)
	//	}
	//
	//	if subTask.beginHeight.Cmp(beginHeight) != 0 {
	//		t.Error("Not Valid, index is "+strconv.Itoa(index)+", subTask.beginHeight is "+
	//			subTask.beginHeight.String(), " beginHeight is "+beginHeight.String())
	//	}
	//	if subTask.targetHeight.Cmp(targetHeight) != 0 {
	//		t.Error("Not Valid, index is "+strconv.Itoa(index)+", subTask.targetHeight is "+
	//			subTask.targetHeight.String(), " targetHeight is "+targetHeight.String())
	//	}
	//}
}

func TestTaskInfo_Split(t *testing.T) {
	//task := &taskInfo{
	//	beginHeight:  big.NewInt(12),
	//	targetHeight: big.NewInt(180),
	//}
	//
	//testTaskInfoSplit(t, task, 10)
	//testTaskInfoSplit(t, task, 5)
	//testTaskInfoSplit(t, task, 1000)
	//
	//task2 := &taskInfo{
	//	beginHeight:  big.NewInt(1),
	//	targetHeight: big.NewInt(5),
	//}
	//
	//testTaskInfoSplit(t, task2, 10)
	//testTaskInfoSplit(t, task2, 12)
	//testTaskInfoSplit(t, task2, 1)
	//
	//task3 := &taskInfo{
	//	beginHeight:  big.NewInt(10),
	//	targetHeight: big.NewInt(50),
	//}
	//
	//testTaskInfoSplit(t, task3, 5)
	//testTaskInfoSplit(t, task3, 8)
	//testTaskInfoSplit(t, task3, 21)
}
