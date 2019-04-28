package chain_file_manager

type DataParser interface {
	Write([]byte) error
	WriteError(err error)
	Close() error
}
