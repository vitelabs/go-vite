package p2p

import (
	"encoding/json"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vitelabs/go-vite/interval/common/log"
)

type linker struct {
	p2p    *p2p
	closed chan struct{}
	loopWg sync.WaitGroup
	url    url.URL
}

func newLinker(p2p *p2p, u url.URL) *linker {
	return &linker{p2p: p2p, url: u, closed: make(chan struct{})}
}

func (self *linker) start() {
	log.Info("connecting to boot note[%s]", self.url.String())
	c, _, err := websocket.DefaultDialer.Dial(self.url.String(), nil)
	if err != nil {
		log.Error("dial:", err)
		return
	}

	log.Info("connected to boot node[%s]", self.url.String())
	go self.loopRead(c)
	go self.loopWrite(c)
}
func (self *linker) stop() {
	close(self.closed)
	self.loopWg.Wait()
}
func (self *linker) loopRead(conn *websocket.Conn) {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Error("read error: %v", err)
			return
		}
		//log.Info("recv: %s", string(message))
		res := []bootReq{}
		json.Unmarshal(message, &res)
		for _, r := range res {
			id := r.Id
			addr := r.Addr
			yes := self.p2p.addDial(id, addr)
			if yes {
				log.Info("recv: add dial success. targetId:%s, selfId:%s", id, self.p2p.id)
			}
		}
	}
}

func (self *linker) loopWrite(conn *websocket.Conn) {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	defer conn.Close()
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	conn.WriteJSON(&bootReq{Id: self.p2p.id, Addr: self.p2p.addr})
	for {
		select {
		case <-self.closed:
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case <-ticker.C:
			err := conn.WriteJSON(&bootReq{Tp: 1})
			if err != nil {
				return
			}
		}
	}
}
