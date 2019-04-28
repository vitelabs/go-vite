package message

type Exception byte

const (
	Missing             Exception = iota // I don`t have the resource you requested
	Canceled                             // the request have been canceled
	Unsolicited                          // the request must have pre-checked
	RepetitiveHandshake                  // handshake should happen only once, as the first msg
	DifferentGenesis
	Unauthorized
)

var exception = map[Exception]string{
	Missing:             "missing resource",
	Canceled:            "request canceled",
	Unsolicited:         "unsolicited request",
	RepetitiveHandshake: "repetitive handshake",
	DifferentGenesis:    "different genesis block",
	Unauthorized:        "unauthorized",
}

func (exp Exception) String() string {
	str, ok := exception[exp]
	if ok {
		return str
	}

	return "unknown exception"
}

func (exp Exception) Error() string {
	return exp.String()
}

func (exp Exception) Serialize() ([]byte, error) {
	return []byte{byte(exp)}, nil
}

func (exp *Exception) Deserialize(buf []byte) error {
	*exp = Exception(buf[0])
	return nil
}
