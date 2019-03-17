package chain_state

type StateDB struct {
	mvDB *multiVersionDB
}

func NewStateDB(chainDir string) (*StateDB, error) {
	mvDB, err := newMultiVersionDB(chainDir)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		mvDB: mvDB,
	}, nil
}

func (sDB *StateDB) Destroy() error {
	if err := sDB.mvDB.Destroy(); err != nil {
		return err
	}

	sDB.mvDB = nil
	return nil
}
