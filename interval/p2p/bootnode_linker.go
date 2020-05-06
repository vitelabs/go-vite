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

func (l *linker) start() {
	log.Info("connecting to boot note[%s]", l.url.String())
	c, _, err := websocket.DefaultDialer.Dial(l.url.String(), nil)
	if err != nil {
		log.Error("dial:", err)
		return
	}

	log.Info("connected to boot node[%s]", l.url.String())
	go l.loopRead(c)
	go l.loopWrite(c)
}
func (l *linker) stop() {
	close(l.closed)
	l.loopWg.Wait()
}
func (l *linker) loopRead(conn *websocket.Conn) {
	l.loopWg.Add(1)
	defer l.loopWg.Done()
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
			yes := l.p2p.addDial(id, addr)
			if yes {
				log.Info("recv: add dial success. targetId:%s, selfId:%s", id, l.p2p.id)
			}
		}
	}
}

func (l *linker) loopWrite(conn *websocket.Conn) {
	l.loopWg.Add(1)
	defer l.loopWg.Done()
	defer conn.Close()
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	conn.WriteJSON(&bootReq{Id: l.p2p.id, Addr: l.p2p.addr})
	for {
		select {
		case <-l.closed:
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
