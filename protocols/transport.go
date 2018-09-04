package protocols

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net"
)

// @section PBTS
const headerLength = 20
const maxPayloadSize = ^uint32(0)

type PBTS struct {
	conn net.Conn
}

// transport use protobuf to serialize/deserialize message.
func NewPBTS(conn net.Conn) Transport {
	return &PBTS{
		conn: conn,
	}
}

func (pt *PBTS) ReadMsg() (m Msg, err error) {
	header := make([]byte, headerLength)

	err = readFullBytes(pt.conn, header)
	if err != nil {
		return m, fmt.Errorf("read msg header error: %v\n", err)
	}

	// extract msg.Code
	m.Code = MsgCode(binary.BigEndian.Uint64(header[:8]))

	// extract length of payload
	size := binary.BigEndian.Uint32(header[8:12])

	if size > maxPayloadSize {
		return m, fmt.Errorf("msg %d payload too large: %d / %d\n", m.Code, size, maxPayloadSize)
	}

	// read payload according to size
	if size > 0 {
		payload := make([]byte, size)

		err = readFullBytes(pt.conn, payload)
		if err != nil {
			return m, fmt.Errorf("read msg %d payload (%d bytes) error: %v\n", m.Code, size, err)
		}

		m.Payload = payload
	}

	return m, nil
}

func readFullBytes(conn net.Conn, data []byte) error {
	length := cap(data)
	index := 0
	for {
		n, err := conn.Read(data[index:])
		if err != nil {
			return err
		}

		index += n
		if index == length {
			break
		}
	}
	return nil
}

func (pt *PBTS) String() string {
	return pt.conn.RemoteAddr().String()
}

func (pt *PBTS) WriteMsg(m Msg) error {
	data, err := pack(m)

	if err != nil {
		return fmt.Errorf("pack msg %s to %s error: %v\n", m.Code, pt, err)
	}

	n, err := pt.conn.Write(data)
	if err != nil {
		return fmt.Errorf("write msg %d (%d bytes) to %s error: %v\n", m.Code, n, pt, err)
	}

	return nil
}

func (pt *PBTS) Handshake(our *HandShakeMsg) (*HandShakeMsg, error) {
	data, err := our.Serialize()
	if err != nil {
		return nil, fmt.Errorf("serialize our handshake error: %v", err)
	}
	sendErr := make(chan error, 1)
	go func() {
		sendErr <- pt.WriteMsg(Msg{
			Code:    HandShakeCode,
			Payload: data,
		})
	}()

	msg, err := pt.ReadMsg()
	if err != nil {
		<-sendErr
		return nil, fmt.Errorf("read handshake from %s error: %v\n", pt, err)
	}

	if msg.Code != HandShakeCode {
		return nil, fmt.Errorf("need handshake from %s, got %d\n", pt, msg.Code)
	}

	hs := new(HandShakeMsg)
	err = hs.Deserialize(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("handshake from %s deserialize error: %v\n", pt, err)
	}

	if hs.Version != our.Version {
		return nil, fmt.Errorf("unmatched version\n")
	}

	if hs.NetID != our.NetID {
		return nil, fmt.Errorf("unmatched network id: %d / %d from %s\n", hs.NetID, our.NetID, pt)
	}

	if hs.GenesisBlock != our.GenesisBlock {
		return nil, errors.New("different genesisblock")
	}

	if err := <-sendErr; err != nil {
		return nil, fmt.Errorf("send handshake to %s error: %v\n", pt, err)
	}

	return hs, nil
}

func (pt *PBTS) Close(exp ExceptionMsg) {
	defer pt.conn.Close()

	data, err := exp.Serialize()
	if err != nil {
		log.Printf("serialize msg %s error: %v\n", exp, err)
		return
	}
	err = pt.WriteMsg(Msg{
		Code:    ExceptionCode,
		Payload: data,
	})

	if err != nil {
		log.Printf("send msg %s to %s error: %v\n", exp, pt, err)
	}
}

func pack(m Msg) (data []byte, err error) {
	if uint32(len(m.Payload)) > maxPayloadSize {
		return data, fmt.Errorf("msg %d payload too large: %d / %d\n", m.Code, len(m.Payload), maxPayloadSize)
	}

	header := make([]byte, headerLength)

	// add code to header
	binary.BigEndian.PutUint64(header[:8], uint64(m.Code))

	// sign payload length to header
	size := uint32(len(m.Payload))
	binary.BigEndian.PutUint32(header[8:12], size)

	// concat header and payload
	if size == 0 {
		return header, nil
	}

	data = append(header, m.Payload...)
	return data, nil
}
