package chain_index

type IndexDB struct {
	db    DB
	memDb MemDB
}

func NewIndexDB() *IndexDB {
	return &IndexDB{
		memDb: newMemDb(),
	}
}

func (iDB *IndexDB) Destroy() {}
