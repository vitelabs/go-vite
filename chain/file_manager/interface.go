package chain_block

type DataParser interface {
	Write([]byte, *Location)
	WriteError(err error)
}
