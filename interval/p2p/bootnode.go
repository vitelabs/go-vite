package p2p

import (
	"encoding/json"
	"net/http"
	"sync"

	"context"
	"strings"

	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
)

func NewBoot(config config.Boot) Boot {
	if config.BootAddr == "" {
		log.Warn("bootAddr is empty.")
		return nil
	}
	b := bootnode{peers: make(map[string]*peer), cfg: config, closed: make(chan struct{})}
	return &b
}

type bootnode struct {
	id     string
	peers  map[string]*peer
	mu     sync.Mutex
	server *http.Server
	cfg    config.Boot
	closed chan struct{}
	wg     sync.WaitGroup
}

func (self *bootnode) Start() {
	self.start(self.cfg.BootAddr)
}

func (self *bootnode) addPeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.peerId]
	if ok && old != peer {
		log.Warn("peer exist, close old peer: %s", peer.info())
		old.close()
	}
	self.peers[peer.peerId] = peer
	go self.loopread(peer)
}
func (self *bootnode) removePeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.peerId]
	if ok && old == peer {
		log.Info("remove peer %v from bootnode.", peer.info())
		old.close()
		delete(self.peers, peer.peerId)
	}
}

func (self *bootnode) loopread(peer *peer) {
	defer log.Info("bootnode loopread closed.")
	self.wg.Add(1)
	defer self.wg.Done()
	conn := peer.conn
	defer self.removePeer(peer)

	for {
		select {
		case <-self.closed:
			return
		case <-peer.closed:
			return
		default:
			req := bootReq{}
			err := conn.ReadJSON(&req)
			if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				log.Error("read message error, peer: %s, %v", peer.info(), err)
				return
			}
			if err != nil {
				log.Error("read message error, peer: %s, %v", peer.info(), err)
				return
			}
			if req.Tp == 1 {
				conn.WriteJSON(self.All())
				continue
			}
		}

	}
}

func (self *bootnode) start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", self.ws)
	server := &http.Server{Addr: addr, Handler: mux}

	//idleConnsClosed := make(chan struct{})
	self.wg.Add(1)
	go func() {
		<-self.closed

		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Info("HTTP server Shutdown: %v", err)
		}
		defer log.Info("bootnode shutdown await closed.")
		self.wg.Done()
		//close(idleConnsClosed)
	}()

	self.wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Info("HTTP server ListenAndServe: %v", err)
		}
		defer log.Info("bootnode listening closed.")
		self.wg.Done()
	}()
	self.server = server
	//<-idleConnsClosed
}

func (self *bootnode) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		req := bootReq{}
		err = c.ReadJSON(&req)
		if err != nil {
			log.Info("read fail.", err)
		}
		bytes, _ := json.Marshal(&req)
		log.Info("upgrade success, add new peer. %s", string(bytes))
		peer := newPeer(req.Id, self.id, req.Addr, c, nil)
		closeHandler := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			self.removePeer(peer)
			closeHandler(code, text)
			return nil
		})

		self.addPeer(peer)
	} else {
		log.Error("upgrade error.", err)
	}
}

func (self *bootnode) All() []*BootLinkPeer {
	var results []*BootLinkPeer
	for _, peer := range self.peers {
		results = append(results, &BootLinkPeer{Id: peer.peerId, Addr: peer.peerSrvAddr})
	}
	return results
}
func (self *bootnode) Stop() {
	for _, peer := range self.peers {
		peer.close()
	}
	self.server.Shutdown(context.Background())
	close(self.closed)
	self.wg.Wait()
}

type bootReq struct {
	Tp   int
	Id   string
	Addr string
}
