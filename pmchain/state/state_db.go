package chain_state

type StateDB struct {
	mvDB *multiVersionDB
}

func NewStateDB() *StateDB {
	return &StateDB{}
}

func (sDB *StateDB) Destroy() {}
