package core

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}
