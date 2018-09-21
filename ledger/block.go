package ledger

type Block interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}
