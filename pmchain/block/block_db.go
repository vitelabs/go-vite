package chain_block

type BlockDB struct {
}

func NewBlockDB() *BlockDB {
	return &BlockDB{}
}

func (bDB *BlockDB) Destroy() {}
