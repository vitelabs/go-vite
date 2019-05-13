package sync_cache

import "testing"

func TestNewSegmentByFilename(t *testing.T) {
	name := "f_100_0000000000000000000000000000000000000000000000000000000000000001_200_0000000000000000000000000000000000000000000000000000000000000002"
	seg, err := newSegmentByFilename(name)
	if err != nil {
		t.Error(err)
	}
	if seg.Bound != [2]uint64{100, 200} {
		t.Errorf("different bound: %v", seg.Bound)
	}

	name = "f_100_0000000000000000000000000000000000000000000000000000000000000001_200_0000000000000000000000000000000000000000000000000000000000000002.v"
	seg, err = newSegmentByFilename(name)
	if err != nil {
		t.Error(err)
	}
}
