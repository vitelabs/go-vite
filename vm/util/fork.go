package util

// TODO
func IsForked(snapshotHeight uint64) bool {
	if snapshotHeight >= 100 {
		return true
	}
	return false
}
