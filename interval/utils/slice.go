package utils

import "github.com/vitelabs/go-vite/interval/common"

func ReverseAccountBlocks(s []*common.AccountStateBlock) []*common.AccountStateBlock {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func ReverseSnapshotBlocks(s []*common.SnapshotBlock) []*common.SnapshotBlock {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
