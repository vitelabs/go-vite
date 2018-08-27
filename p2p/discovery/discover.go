package discovery

import (
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"net"
)

func priv2ID(priv ed25519.PrivateKey) (id NodeID) {
	pub := priv.PubByte()
	copy(id[:], pub)
	return
}

// @section config
type Config struct {
	Priv      ed25519.PrivateKey
	DBPath    string
	BootNodes []*Node
	Addr      *net.UDPAddr
}

func New(cfg *Config) error {
	conn, err := net.ListenUDP("udp", cfg.Addr)
	if err != nil {
		discvLog.Crit("discv listen udp ", "error", err)
	}
	node := &Node{
		ID:   priv2ID(cfg.Priv),
		IP:   cfg.Addr.IP,
		Port: uint16(cfg.Addr.Port),
	}

	agent := &udpAgent{
		self:    node,
		conn:    conn,
		priv:    cfg.Priv,
		waiting: make(chan *wait),
		res:     make(chan *res),
		stopped: make(chan struct{}),
	}

	discvLog.Info(node.String())

	err = newTable(node, agent, cfg.DBPath, cfg.BootNodes)

	if err != nil {
		return err
	}

	go agent.loop()
	go agent.readLoop()

	return nil
}
