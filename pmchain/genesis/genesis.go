package chain_genesis

type Genesis struct {
	indexDB IndexDB
	blockDB BlockDB
}

func NewGenesis(indexDB IndexDB, blockDB BlockDB) *Genesis {
	return &Genesis{}
}

func (genesis *Genesis) InitLedger() bool {
	return false
}

func (genesis *Genesis) check() {

}

func (genesis *Genesis) destroyDB() {

}
