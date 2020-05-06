package p2p

import (
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type handShaker struct {
	p2p *p2p
	biz HandShaker
}

func (self *handShaker) handshake(conn *websocket.Conn) (*peer, error) {
	var err error
	s, e := self.biz.GetState()
	if e != nil {
		return nil, e
	}

	conn.WriteJSON(handshakeMsg{Id: self.p2p.id, NetId: self.p2p.netId, Addr: self.p2p.addr, S: self.biz.EncodeState(s)})
	req := handshakeMsg{}
	err = conn.ReadJSON(&req)
	if err != nil {
		return nil, err
	}

	if req.NetId != self.p2p.netId {
		return nil, errors.New("NetId diff, self[" + strconv.Itoa(self.p2p.netId) + "], peer[" + strconv.Itoa(req.NetId) + "]")
	}

	err = self.biz.Handshake(req.Id, req.S)
	if err != nil {
		return nil, err
	}

	state := self.biz.DecodeState(req.S)
	return newPeer(req.Id, self.p2p.id, req.Addr, conn, state), nil
}

type handshakeMsg struct {
	NetId int
	Id    string
	Addr  string
	S     []byte
}
