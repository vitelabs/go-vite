package chain_index

type IndexDB struct {
	store Store
	memDb MemDB
}

func NewIndexDB(chainDir string) (*IndexDB, error) {
	store, err := NewStore(chainDir)
	if err != nil {
		return nil, err
	}
	return &IndexDB{
		store: store,
		memDb: newMemDb(),
	}, nil
}

func (iDB *IndexDB) CleanUnconfirmedIndex() {
	iDB.memDb.Clean()
}

func (iDB *IndexDB) Destroy() {}
