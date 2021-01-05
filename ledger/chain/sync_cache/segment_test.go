package sync_cache

import "testing"

func TestNewSegmentByFilename(t *testing.T) {
	name := "f_100_0000000000000000000000000000000000000000000000000000000000000001_200_0000000000000000000000000000000000000000000000000000000000000002"
	seg, err := newSegmentByFilename(name)
	if err != nil {
		t.Error(err)
	}
	if seg.From != 100 || seg.To != 200 {
		t.Errorf("different bound: %d %d", seg.From, seg.To)
	}

	name = "f_100_0000000000000000000000000000000000000000000000000000000000000001_200_0000000000000000000000000000000000000000000000000000000000000002.v"
	seg, err = newSegmentByFilename(name)
	if err != nil {
		t.Error(err)
	}
}
