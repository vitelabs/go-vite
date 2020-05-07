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

func (bt *bootnode) Start() {
	bt.start(bt.cfg.BootAddr)
}

func (bt *bootnode) addPeer(peer *peer) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	old, ok := bt.peers[peer.peerId]
	if ok && old != peer {
		log.Warn("peer exist, close old peer: %s", peer.info())
		old.close()
	}
	bt.peers[peer.peerId] = peer
	go bt.loopread(peer)
}
func (bt *bootnode) removePeer(peer *peer) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	old, ok := bt.peers[peer.peerId]
	if ok && old == peer {
		log.Info("remove peer %v from bootnode.", peer.info())
		old.close()
		delete(bt.peers, peer.peerId)
	}
}

func (bt *bootnode) loopread(peer *peer) {
	defer log.Info("bootnode loopread closed.")
	bt.wg.Add(1)
	defer bt.wg.Done()
	conn := peer.conn
	defer bt.removePeer(peer)

	for {
		select {
		case <-bt.closed:
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
				conn.WriteJSON(bt.All())
				continue
			}
		}

	}
}

func (bt *bootnode) start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", bt.ws)
	server := &http.Server{Addr: addr, Handler: mux}

	//idleConnsClosed := make(chan struct{})
	bt.wg.Add(1)
	go func() {
		<-bt.closed

		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Info("HTTP server Shutdown: %v", err)
		}
		defer log.Info("bootnode shutdown await closed.")
		bt.wg.Done()
		//close(idleConnsClosed)
	}()

	bt.wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Info("HTTP server ListenAndServe: %v", err)
		}
		defer log.Info("bootnode listening closed.")
		bt.wg.Done()
	}()
	bt.server = server
	//<-idleConnsClosed
}

func (bt *bootnode) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		req := bootReq{}
		err = c.ReadJSON(&req)
		if err != nil {
			log.Info("read fail.", err)
		}
		bytes, _ := json.Marshal(&req)
		log.Info("upgrade success, add new peer. %s", string(bytes))
		peer := newPeer(req.Id, bt.id, req.Addr, c, nil)
		closeHandler := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			bt.removePeer(peer)
			closeHandler(code, text)
			return nil
		})

		bt.addPeer(peer)
	} else {
		log.Error("upgrade error.", err)
	}
}

func (bt *bootnode) All() []*BootLinkPeer {
	var results []*BootLinkPeer
	for _, peer := range bt.peers {
		results = append(results, &BootLinkPeer{Id: peer.peerId, Addr: peer.peerSrvAddr})
	}
	return results
}
func (bt *bootnode) Stop() {
	for _, peer := range bt.peers {
		peer.close()
	}
	bt.server.Shutdown(context.Background())
	close(bt.closed)
	bt.wg.Wait()
}

type bootReq struct {
	Tp   int
	Id   string
	Addr string
}
