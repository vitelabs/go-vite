package chain_file_manager

type DataParser interface {
	Write([]byte, *Location) error
	WriteError(err error)
	Close() error
}
