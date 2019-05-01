package ledger

type SnapshotChunk struct {
	SnapshotBlock *SnapshotBlock  `json:"snapshotBlock"`
	AccountBlocks []*AccountBlock `json:"accountBlocks"`
}
