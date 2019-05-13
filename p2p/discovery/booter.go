package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

// booter can supply bootNodes
type booter interface {
	// getBootNodes return count nodes
	getBootNodes(count int) []*Node
}

// booterDB can retrieve bootNodes
type booterDB interface {
	ReadNodes(count int, expiration time.Duration) []*Node
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
	node      *vnode.Node
}

func newCfgBooter(bootNodes []string, node *vnode.Node) (booter, error) {
	var c = &cfgBooter{
		bootNodes: make([]*Node, len(bootNodes)),
		node:      node,
	}

	var n *vnode.Node
	var err error

	for i, url := range bootNodes {
		n, err = vnode.ParseNode(url)

		if err != nil {
			return nil, fmt.Errorf("failed to parse bootNode: %s", url)
		}

		if n.Net != 0 && n.Net != node.Net {
			return nil, errDifferentNet
		}

		n.Net = node.Net

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
	self   *vnode.Node
	reader *requestReader
	seeds  []string
	log    log15.Logger
}

type Request struct {
	Node  *vnode.Node `json:"node"`
	Count int         `json:"count"`
}

type Result struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type requestReader struct {
	request Request
	buf     []byte
	r       int
}

func (r *requestReader) Read(p []byte) (n int, err error) {
	to := r.r + len(p)
	if to > len(r.buf) {
		to = len(r.buf)
	}

	n = copy(p, r.buf[r.r:to])
	r.r = to
	if r.r == len(r.buf) {
		err = io.EOF
	}

	return
}

func (r *requestReader) reset(count int) {
	r.request.Count = count

	r.buf, _ = json.Marshal(r.request)

	r.r = 0
}

func newNetBooter(self *vnode.Node, seeds []string) booter {
	return &netBooter{
		self: self,
		reader: &requestReader{
			request: Request{
				Node:  self,
				Count: 0,
			},
			r: 0,
		},
		seeds: seeds,
		log:   discvLog.New("module", "netBooter"),
	}
}

func (n *netBooter) getBootNodes(count int) (nodes []*Node) {
	var seed string
	if len(n.seeds) > 0 {
		seed = n.seeds[rand.Intn(len(n.seeds))]
	}

	if seed == "" {
		return nil
	}

	n.reader.reset(count)

	resp, err := http.Post(seed, "application/json", n.reader)

	if err != nil {
		n.log.Error(fmt.Sprintf("failed to request seed %s: %v", seed, err))
		return nil
	}

	var ret []byte
	var buf = make([]byte, 1024)

	i, err := resp.Body.Read(buf)
	ret = append(ret, buf[:i]...)

	if err != nil && err != io.EOF {
		n.log.Error(fmt.Sprintf("failed to parse response from seed %s: %v", seed, err))
		return nil
	}

	var result = new(Result)
	err = json.Unmarshal(ret, result)
	if err != nil {
		n.log.Error(fmt.Sprintf("failed to parse response from seed %s: %v", seed, err))
		return nil
	}

	if result.Code != 0 {
		n.log.Error(fmt.Sprintf("failed to get bootNodes from seed %s: %d %s", seed, result.Code, result.Message))
		return nil
	}

	var node *vnode.Node
	for _, str := range result.Data {
		node, err = vnode.ParseNode(str)
		if err != nil {
			n.log.Error(fmt.Sprintf("failed to parse bootNode %s from seed %s: %v", str, seed, err))
		} else {
			nodes = append(nodes, &Node{
				Node: *node,
			})
		}
	}

	return nodes
}
