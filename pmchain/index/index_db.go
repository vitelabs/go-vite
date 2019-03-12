package chain_index

type IndexDB struct {
	db DB
}

func NewIndexDB() *IndexDB {
	return &IndexDB{}
}

func (iDB *IndexDB) Destroy() {}
