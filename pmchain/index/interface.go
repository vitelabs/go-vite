package chain_index

type DB interface {
	NewBatch() Batch
	Write(Batch) error
}

type Batch interface {
	Put(key, value []byte)
}
