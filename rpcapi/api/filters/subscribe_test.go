package filters

import (
	"testing"
)

type HeightResult struct {
	offset uint64
	count  uint64
	finish bool
}

func TestGetHeightPage(t *testing.T) {
	testCases := []struct {
		startHeight uint64
		endHeight   uint64
		accHeight   uint64
		resultList  []HeightResult
	}{
		{0, 50, 150, []HeightResult{
			{50, 50, true}},
		},
		{0, 0, 150, []HeightResult{
			{100, 100, false},
			{150, 50, true},
		}},
		{0, 500, 150, []HeightResult{
			{100, 100, false},
			{150, 50, true},
		}},
		{0, 500, 700, []HeightResult{
			{100, 100, false},
			{200, 100, false},
			{300, 100, false},
			{400, 100, false},
			{500, 100, true},
		}},
		{0, 501, 700, []HeightResult{
			{100, 100, false},
			{200, 100, false},
			{300, 100, false},
			{400, 100, false},
			{500, 100, false},
			{501, 1, true},
		}},
	}
	for i, testCase := range testCases {
		startHeight := testCase.startHeight
		endHeight := testCase.endHeight
		accHeight := testCase.accHeight
		if startHeight == 0 {
			startHeight = 1
		}
		if endHeight == 0 || endHeight > accHeight {
			endHeight = accHeight
		}
		index := 0
		for {
			if index > len(testCase.resultList)-1 {
				t.Fatalf("%vth testcase, current index %v, expected finish", i, index)
			}
			offset, count, finish := GetHeightPage(startHeight, endHeight, getAccountBlocksCount)
			startHeight = offset + 1
			if result := testCase.resultList[index]; offset != result.offset || count != result.count || finish != result.finish {
				t.Fatalf("%vth testcase, index %v, expected [%v,%v,%v], got [%v,%v,%v]", i, index, result.offset, result.count, result.finish, offset, count, finish)
			}
			index = index + 1
			if count == 0 {
				break
			}
			if finish {
				break
			}
		}
		if index != len(testCase.resultList) {
			t.Fatalf("%vth testcase, expected result list len %v, got %v", i, len(testCase.resultList), index)
		}
	}

}
