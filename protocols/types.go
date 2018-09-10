package protocols

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

type Msg struct {
	Code    uint64
	Id      uint64
	Payload Serializable
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type Transport interface {
	MsgReadWriter
	Close()
}
