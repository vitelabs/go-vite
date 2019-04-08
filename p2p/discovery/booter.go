package discovery

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

// booter can supply bootNodes
type booter interface {
	// getBootNodes return count nodes
	getBootNodes(count int) []*Node
}

// booterDB can retrieve bootNodes
type booterDB interface {
	ReadNodes(count int, maxAge time.Duration) []*Node
}

// dbBooter supply bootNodes from database
type dbBooter struct {
	db booterDB
}

func newDBBooter(db booterDB) booter {
	return &dbBooter{
		db: db,
	}
}

func (d *dbBooter) getBootNodes(count int) []*Node {
	return d.db.ReadNodes(count, seedMaxAge)
}

// cfgBooter supply random bootNodes from config
type cfgBooter struct {
	bootNodes []*Node
}

func newCfgBooter(bootNodes []string) (booter, error) {
	var c = &cfgBooter{
		bootNodes: make([]*Node, len(bootNodes)),
	}

	var n *vnode.Node
	var err error

	for i, url := range bootNodes {
		n, err = vnode.ParseNode(url)

		if err != nil {
			return nil, fmt.Errorf("Failed to parse bootNode: %s", url)
		}

		c.bootNodes[i] = &Node{
			Node: *n,
		}
	}

	return c, nil
}

func (c *cfgBooter) getBootNodes(count int) []*Node {
	total := len(c.bootNodes)

	if count < total {
		nodes := make([]*Node, count)
		indexes := rand.Perm(total)

		for i := 0; i < count; i++ {
			nodes[i] = c.bootNodes[indexes[i]]
		}

		return nodes
	}

	return c.bootNodes
}

// netBooter supply bootNodes from service, eg. HTTP request.
type netBooter struct {
	self  *vnode.Node
	seeds []string
}

func newNetBooter(self *vnode.Node, seeds []string) booter {
	return &netBooter{
		self:  self,
		seeds: seeds,
	}
}

func (n *netBooter) getBootNodes(count int) []*Node {
	// todo
	return nil
}
