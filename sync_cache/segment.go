package sync_cache

const (
	separator = "_"
)

type Segment [2]uint64

func newSegment(from uint64, to uint64) *Segment {
	seg := Segment{from, to}
	return &seg
}
func newSegmentByFilename(filename string) *Segment {
	return nil
}

type SegmentList []*Segment

func (list SegmentList) Len() int { return len(list) }
func (list SegmentList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list SegmentList) Less(i, j int) bool {
	return list[i][0] < list[j][1]
}
