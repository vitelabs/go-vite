package sync_cache

import (
	"os"
	"testing"
)

func TestRead3(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect sync cache.")

	filename := "xxxxx/f_57990301_57991300_1617778532"
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}
	reader := &Reader{
		cache:      nil,
		file:       file,
		readBuffer: make([]byte, 8*1024),
		item:       nil,
	}
	for {

		ab, sb, err := reader.Read()
		if err != nil {
			break
		} else if ab != nil {
			t.Logf("[ab] height:%d, hash:%s, address:%s\n", ab.Height, ab.Hash, ab.AccountAddress)
		} else if sb != nil {
			t.Logf("[sb] height:%d, hash:%s\n", sb.Height, sb.Hash)
		}
	}
}
