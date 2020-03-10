package ledger

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}
