/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package p2p

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/p2p/vnode"
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
	versionKey         = []byte("version")
	nodeDataPrefix     = []byte("node:data:")
	nodeActivePrefix   = []byte("node:active:")
	nodeCheckPrefix    = []byte("node:check:")
	nodeMarkPrefix     = []byte("node:mark:")
	nodeEndPointPrefix = []byte("node:ep:")
)

func newNodeDB(path string, version int, id vnode.NodeID) (db *nodeDB, err error) {
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

func (db *nodeDB) RetrieveActiveAt(id vnode.NodeID) int64 {
	key := append(nodeActivePrefix, id.Bytes()...)
	return db.RetrieveInt64(key)
}

func (db *nodeDB) StoreActiveAt(id vnode.NodeID, v int64) {
	key := append(nodeActivePrefix, id.Bytes()...)
	db.StoreInt64(key, v)
}

func (db *nodeDB) RetrieveCheckAt(id vnode.NodeID) int64 {
	key := append(nodeCheckPrefix, id.Bytes()...)
	return db.RetrieveInt64(key)
}

func (db *nodeDB) StoreCheckAt(id vnode.NodeID, v int64) {
	key := append(nodeCheckPrefix, id.Bytes()...)
	db.StoreInt64(key, v)
}

// RetrieveNode Node according to the special nodeID
func (db *nodeDB) RetrieveNode(ID vnode.NodeID) (node *vnode.Node, err error) {
	id := ID.Bytes()

	key := append(nodeDataPrefix, id...)
	// retrieve node
	data, err := db.Get(key, nil)
	if err != nil {
		return
	}

	node = new(vnode.Node)
	err = node.Deserialize(data)

	return
}

// StoreNode node into database
// Node.activeAt and Node.checkAt is store separately
func (db *nodeDB) StoreNode(node *vnode.Node) (err error) {
	data, err := node.Serialize()
	if err != nil {
		return
	}

	id := node.ID.Bytes()
	// store node
	key := append(nodeDataPrefix, id...)
	err = db.Put(key, data, nil)

	return
}

// RemoveNode data about the specific NodeID
func (db *nodeDB) RemoveNode(ID vnode.NodeID) {
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
func (db *nodeDB) ReadNodes(count int, expiration time.Duration) []*vnode.Node {
	itr := db.NewIterator(util.BytesPrefix(nodeActivePrefix), nil)
	defer itr.Release()

	nodes := make([]*vnode.Node, 0, count)
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
		if now.Sub(time.Unix(db.RetrieveInt64(activeKey), 0)) > expiration {
			db.RemoveNode(id)
			continue
		}

		node, err := db.RetrieveNode(id)
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

func (db *nodeDB) RetrieveInt64(key []byte) int64 {
	buf, err := db.Get(key, nil)
	if err != nil {
		return 0
	}

	return decodeVarint(buf)
}

func (db *nodeDB) StoreInt64(key []byte, n int64) {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]

	_ = db.Put(key, buf, nil)
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
		if now.Sub(time.Unix(db.RetrieveInt64(activeKey), 0)) > expiration {
			db.RemoveNode(id)
			continue
		}
	}
}

// StoreEndPoint larger weight, more front
func (db *nodeDB) StoreEndPoint(id vnode.NodeID, ep vnode.EndPoint, weight int64) {
	data, err := ep.Serialize()
	if err != nil {
		return
	}

	key := append(nodeEndPointPrefix, id.Bytes()...)
	err = db.Put(key, data, nil)
	if err != nil {
		return
	}

	key = append(nodeMarkPrefix, id.Bytes()...)
	db.StoreInt64(key, weight)
}

func (db *nodeDB) RemoveEndPoint(id vnode.NodeID) {
	key := append(nodeEndPointPrefix, id.Bytes()...)
	_ = db.Delete(key, nil)

	key = append(nodeMarkPrefix, id.Bytes()...)
	_ = db.Delete(key, nil)
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

// RetrieveEndPoints return max count nodes, sort by mark from large to small
func (db *nodeDB) RetrieveEndPoints(count int) (nodes []*vnode.Node) {
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
			mark: db.RetrieveInt64(key),
			id:   id,
		})
	}

	sort.Sort(ms)

	nodes = make([]*vnode.Node, 0, count)
	for _, m := range ms {
		key := append(nodeEndPointPrefix, m.id.Bytes()...)

		value, err := db.Get(key, nil)
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		var e vnode.EndPoint
		err = e.Deserialize(value)
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		var n = &vnode.Node{
			ID:       m.id,
			EndPoint: e,
		}
		nodes = append(nodes, n)

		if len(nodes) >= count {
			break
		}
	}

	return nodes
}

func (db *nodeDB) Iterate(prefix []byte, fn func(key, value []byte) bool) {
	itr := db.NewIterator(util.BytesPrefix(nodeMarkPrefix), nil)
	defer itr.Release()

	for itr.Next() {
		if fn(itr.Key(), itr.Value()) == false {
			break
		}
	}
}

func (db *nodeDB) Register(prefix []byte) *prefixDB {
	return &prefixDB{
		db:     db.DB,
		prefix: prefix,
	}
}

type prefixDB struct {
	db     *leveldb.DB
	prefix []byte
}

func (pdb *prefixDB) Store(key, value []byte) error {
	key = append(pdb.prefix, key...)
	return pdb.db.Put(key, value, nil)
}

func (pdb *prefixDB) Retrieve(key []byte) []byte {
	key = append(pdb.prefix, key...)
	value, err := pdb.db.Get(key, nil)
	if err != nil {
		return nil
	}
	return value
}

func (pdb *prefixDB) Remove(key []byte) {
	key = append(pdb.prefix, key...)
	_ = pdb.db.Delete(key, nil)
}
