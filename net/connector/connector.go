package connector

import (
	"net"
	"time"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net/vnode"
)

type Finder interface {
	Nodes() []*vnode.Node
}

type Connector struct {
	listenAddress string

	server *net.TCPListener
	finder Finder

	onNode       func(node *vnode.Node) bool
	onConnection func(c *net.TCPConn, inbound bool)

	term chan struct{}

	log log15.Logger
}

func New(listenAddress string, finder Finder, onNodeHandler func(node *vnode.Node) bool, onConnectionHandler func(c *net.TCPConn, inbound bool)) *Connector {
	var c = &Connector{
		listenAddress: listenAddress,
		server:        nil,
		finder:        finder,
		onNode:        onNodeHandler,
		onConnection:  onConnectionHandler,
		log:           log15.New("module", "connector"),
	}

	return c
}

func (c *Connector) Start() (err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", c.listenAddress)
	if err != nil {
		return
	}

	c.server, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return
	}

	c.term = make(chan struct{})

	go func() {
		var tempDelay time.Duration
		var maxDelay = time.Second

		for {
			var conn *net.TCPConn
			conn, err = c.server.AcceptTCP()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}

					if tempDelay > maxDelay {
						tempDelay = maxDelay
					}

					time.Sleep(tempDelay)

					continue
				}
				break
			}

			go c.onConnection(conn, true)
		}
	}()

	return
}

func (c *Connector) findLoop() {
	var initDuration = 10 * time.Second
	var maxDuration = 160 * time.Second
	var duration = initDuration
	var timer = time.NewTimer(initDuration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			nodes := c.finder.Nodes()

			if len(nodes) > 0 {
				for _, n := range nodes {
					if c.onNode(n) {
						_ = c.ConnectNode(n)
					}
				}
			} else {
				if duration < maxDuration {
					duration *= 2
				} else {
					duration = initDuration
				}
			}

			timer.Reset(duration)

		case <-c.term:
			return
		}
	}
}

func (c *Connector) ConnectNode(node *vnode.Node) (err error) {
	tcp, err := net.ResolveTCPAddr("tcp", node.Address())
	if err != nil {
		return
	}

	conn, err := net.DialTCP("tcp", nil, tcp)
	if err != nil {
		return
	}

	go c.onConnection(conn, false)

	return nil
}
