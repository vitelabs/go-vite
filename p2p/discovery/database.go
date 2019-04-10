package discovery

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type nodeDB struct {
	*leveldb.DB
	id vnode.NodeID
}

// key -> value
// version -> version
// node:data:ID -> node	// store node data, like IP port ext and so on
// node:active:ID -> int64	// active time, easy to compare time and clean
// node:mark:ID -> int64	// mark weight
// node:check:ID -> int64 // check time
var (
	versionKey       = []byte("version")
	nodeDataPrefix   = []byte("node:data:")
	nodeActivePrefix = []byte("node:active:")
	nodeCheckPrefix  = []byte("node:check:")
	nodeMarkPrefix   = []byte("node:mark:")
)

func newDB(path string, version int, id vnode.NodeID) (db *nodeDB, err error) {
	if path == "" {
		db, err = newMemDB(id)
	} else {
		db, err = newFileDB(path, version, id)
	}

	if err != nil {
		return nil, err
	}

	return
}

func newMemDB(id vnode.NodeID) (*nodeDB, error) {
	ldb, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &nodeDB{
		DB: ldb,
		id: id,
	}, nil
}

func newFileDB(path string, version int, id vnode.NodeID) (*nodeDB, error) {
	ldb, err := leveldb.OpenFile(path, nil)
	if _, ok := err.(*errors.ErrCorrupted); ok {
		ldb, err = leveldb.RecoverFile(path, nil)
	}

	if err != nil {
		return nil, err
	}

	vBytes := encodeVarint(int64(version))
	oldVBytes, err := ldb.Get(versionKey, nil)

	if err == leveldb.ErrNotFound {
		err = ldb.Put(versionKey, vBytes, nil)

		if err != nil {
			_ = ldb.Close()
			return nil, err
		}
		return &nodeDB{
			DB: ldb,
			id: id,
		}, nil
	} else if err == nil {
		if bytes.Equal(oldVBytes, vBytes) {
			return &nodeDB{
				DB: ldb,
				id: id,
			}, err
		}

		_ = ldb.Close()
		err = os.RemoveAll(path)
		if err != nil {
			return nil, err
		}
		return newFileDB(path, version, id)
	}

	return nil, err
}

func decodeVarint(varint []byte) int64 {
	i, n := binary.Varint(varint)
	if n <= 0 {
		return 0
	}
	return i
}

func encodeVarint(i int64) []byte {
	data := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(data, i)
	return data[:n]
}

// Retrieve Node according to the special nodeID
func (db *nodeDB) Retrieve(ID vnode.NodeID) (node *Node, err error) {
	id := ID.Bytes()

	key := append(nodeDataPrefix, id...)
	// retrieve node
	data, err := db.Get(key, nil)
	if err != nil {
		return
	}

	node = new(Node)
	err = node.Deserialize(data)
	if err != nil {
		return
	}

	key = append(nodeActivePrefix, id...)
	node.activeAt = time.Unix(db.retrieveInt64(key), 0)

	key = append(nodeCheckPrefix, id...)
	node.checkAt = time.Unix(db.retrieveInt64(key), 0)

	return
}

// Store node into database
// Node.activeAt and Node.checkAt is store separately
func (db *nodeDB) Store(node *Node) (err error) {
	data, err := node.Serialize()
	if err != nil {
		return
	}

	id := node.ID.Bytes()
	// store node
	key := append(nodeDataPrefix, id...)
	err = db.Put(key, data, nil)

	key = append(nodeActivePrefix, id...)
	err = db.storeInt64(key, node.activeAt.Unix())

	key = append(nodeCheckPrefix, id...)
	err = db.storeInt64(key, node.checkAt.Unix())

	return
}

// Remove data about the specific NodeID
func (db *nodeDB) Remove(ID vnode.NodeID) {
	id := ID.Bytes()

	key := append(nodeDataPrefix, id...)
	_ = db.Delete(key, nil)

	key = append(nodeActivePrefix, id...)
	_ = db.Delete(key, nil)

	key = append(nodeCheckPrefix, id...)
	_ = db.Delete(key, nil)

	key = append(nodeMarkPrefix, id...)
	_ = db.Delete(key, nil)
}

// ReadNodes max count nodes from database, if time.Now().Sub(node.activeAt) < expiration
func (db *nodeDB) ReadNodes(count int, expiration time.Duration) []*Node {
	itr := db.NewIterator(util.BytesPrefix(nodeActivePrefix), nil)
	defer itr.Release()

	nodes := make([]*Node, 0, count)
	now := time.Now()
	prefixLen := len(nodeActivePrefix)

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		activeKey := append(nodeActivePrefix, id.Bytes()...)
		if now.Sub(time.Unix(db.retrieveInt64(activeKey), 0)) > expiration {
			db.Remove(id)
			continue
		}

		node, err := db.Retrieve(id)
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		nodes = append(nodes, node)

		if len(nodes) > count {
			break
		}
	}

	return nodes
}

func (db *nodeDB) retrieveInt64(key []byte) int64 {
	buf, err := db.Get(key, nil)
	if err != nil {
		return 0
	}

	return decodeVarint(buf)
}

func (db *nodeDB) storeInt64(key []byte, n int64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]

	return db.Put(key, buf, nil)
}

// Clean nodes if time.Now().Sub(node.activeAt) > expiration
func (db *nodeDB) Clean(expiration time.Duration) {
	itr := db.NewIterator(util.BytesPrefix(nodeActivePrefix), nil)
	defer itr.Release()

	now := time.Now()
	prefixLen := len(nodeActivePrefix)

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		activeKey := append(nodeActivePrefix, id.Bytes()...)
		if now.Sub(time.Unix(db.retrieveInt64(activeKey), 0)) > expiration {
			db.Remove(id)
			continue
		}
	}
}

// MarkNode larger weight, more front
func (db *nodeDB) MarkNode(id vnode.NodeID, weight int64) {
	key := append(nodeMarkPrefix, id.Bytes()...)
	_ = db.storeInt64(key, weight)
}

type mark struct {
	mark int64
	id   vnode.NodeID
}
type marks []mark

func (m marks) Len() int {
	return len(m)
}

func (m marks) Less(i, j int) bool {
	return m[i].mark > m[j].mark
}

func (m marks) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

// RetrieveMarkNodes return max count nodes, sort by mark from large to small
func (db *nodeDB) RetrieveMarkNodes(count int) (nodes []vnode.Node) {
	itr := db.NewIterator(util.BytesPrefix(nodeMarkPrefix), nil)
	defer itr.Release()

	prefixLen := len(nodeMarkPrefix)
	ms := make(marks, 0, count*2)

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		ms = append(ms, mark{
			mark: db.retrieveInt64(key),
			id:   id,
		})
	}

	sort.Sort(ms)

	nodes = make([]vnode.Node, 0, count)
	for _, m := range ms {
		n, err := db.Retrieve(m.id)
		if err != nil {
			db.Remove(m.id)
			continue
		}

		nodes = append(nodes, n.Node)

		if len(nodes) > count {
			break
		}
	}

	return nodes
}
