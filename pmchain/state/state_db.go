package chain_state

type StateDB struct{}

func NewStateDB() *StateDB {
	return &StateDB{}
}

func (sDB *StateDB) Destroy() {}
