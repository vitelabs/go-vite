package ledger

type SnapshotChunk struct {
	SnapshotBlock *SnapshotBlock
	AccountBlocks []*AccountBlock
}
