package discovery

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/p2p/config"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

const tableRefreshInterval = 24 * time.Hour
const maxSeedAge = 7 * 24 * time.Hour
const storeInterval = 15 * time.Minute
const liveToStore = 5 * time.Minute
const unactiveToCheck = 15 * time.Minute

type Discover interface {
	Start() error
	Stop() error
	Check(addr string)
	Remove(id vnode.NodeID)
	// GetNodes get a batch nodes to connect
	GetNodes() []*vnode.Node
	// VerifyNode if id or addr or other info get from TCP is different with UDP
	VerifyNode(id vnode.NodeID, addr string)
}

type discovery struct {
	cfg config.Config

	self *vnode.Node

	mode vnode.NodeMode

	finder Finder

	socket Socket

	tab nodeTable

	db db

	refreshing bool
	mu         sync.Mutex
	cond       *sync.Cond

	booters []booter

	log log15.Logger
}

func (d *discovery) init() {
	var nodes []*Node
	for _, bt := range d.booters {
		nodes = append(nodes, bt.getBootNodes()...)
	}

	// todo ping-pong check stale nodes
	// ...

	d.tab.reset()
	for _, n := range nodes {
		d.tab.add(n)
	}

	for dist := uint(0); dist < vnode.IDBits; dist++ {
		id := vnode.RandFromDistance(d.self.ID, dist)
		go d.lookup(id)
	}
}

// lookup nodes near the target id, these nodes will be put into nodeTable
// the lookup process will be blocked and time-consuming
func (d *discovery) lookup(id vnode.NodeID) []*Node {

}

// loop keep table fresh, check old nodes alive, store table to db
func (d *discovery) loop() {

}

func (d *discovery) Start() (err error) {
	var dbPath string
	if d.cfg.DataDir != "" {
		dbPath = filepath.Join(d.cfg.DataDir, "p2p")
	}

	d.db, err = newDB(dbPath, 3, d.self.ID)
	if err != nil {
		return
	}

	// todo

	err = d.socket.Start()
	if err != nil {
		return
	}

	return nil
}

func (d *discovery) Stop() (err error) {
	if d.socket != nil {
		err = d.socket.Stop()
	}

	var err2 error
	if d.db != nil {
		if err2 = d.db.close(); err == nil {
			err = err2
		}
	}

	return
}

func New(cfg config.Config) Discover {
	d := discovery{
		cfg: cfg,
		tab: newTable(&cfg.Node, bucketSize, bucketNum, newListBucket),
		log: log15.New("module", "discover"),
	}

	// from network
	if len(cfg.BootSeed) > 0 {
		d.booters = append(d.booters, newNetBooter(&cfg.Node, cfg.BootSeed))
	}

	// from config
	if len(cfg.BootNodes) > 0 {
		b, err := newCfgBooter(cfg.BootNodes)
		if err != nil {
			panic(err)
		}

		d.booters = append(d.booters, b)
	}

	// from db
	d.booters = append(d.booters, newDBBooter(d.db))

	//d.socket = newSocket(cfg.ListenAddress, , d.log.New("sock", "sock"))

	d.cond = sync.NewCond(&d.mu)

	return nil
}
