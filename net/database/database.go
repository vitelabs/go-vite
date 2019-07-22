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

package database

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"sort"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/net/vnode"
)

type DB struct {
	*leveldb.DB
	id vnode.NodeID
}

var (
	versionKey = []byte("version")

	nodeDataPrefix   = []byte("node:data:")   // ID IP Port
	nodeActivePrefix = []byte("node:active:") // activeAt
	nodeCheckPrefix  = []byte("node:check:")  // checkAt
	nodeMarkPrefix   = []byte("node:mark:")   // mark

	nodeBlockIPPrefix = []byte("node:block:ip:") // block expiration
	nodeBlockIDPrefix = []byte("node:block:id:") // block expiration
)

func New(path string, version int, id vnode.NodeID) (db *DB, err error) {
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

func newMemDB(id vnode.NodeID) (*DB, error) {
	ldb, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &DB{
		DB: ldb,
		id: id,
	}, nil
}

func newFileDB(path string, version int, id vnode.NodeID) (*DB, error) {
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
		return &DB{
			DB: ldb,
			id: id,
		}, nil
	} else if err == nil {
		if bytes.Equal(oldVBytes, vBytes) {
			return &DB{
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

func (db *DB) RetrieveActiveAt(id vnode.NodeID) int64 {
	key := append(nodeActivePrefix, id.Bytes()...)
	return db.RetrieveInt64(key)
}

func (db *DB) StoreActiveAt(id vnode.NodeID, v int64) {
	key := append(nodeActivePrefix, id.Bytes()...)
	db.StoreInt64(key, v)
}

func (db *DB) RetrieveCheckAt(id vnode.NodeID) int64 {
	key := append(nodeCheckPrefix, id.Bytes()...)
	return db.RetrieveInt64(key)
}

func (db *DB) StoreCheckAt(id vnode.NodeID, v int64) {
	key := append(nodeCheckPrefix, id.Bytes()...)
	db.StoreInt64(key, v)
}

func (db *DB) RetrieveMark(id vnode.NodeID) int64 {
	key := append(nodeMarkPrefix, id.Bytes()...)
	return db.RetrieveInt64(key)
}

// value + time
func (db *DB) StoreMark(id vnode.NodeID, v int64) {
	key := append(nodeMarkPrefix, id.Bytes()...)

	value := make([]byte, 16)
	binary.BigEndian.PutUint64(value, uint64(v))
	binary.BigEndian.PutUint64(value[8:], uint64(time.Now().Unix()))

	_ = db.DB.Put(key, value, nil)
}

func (db *DB) BlockIP(ip net.IP, expiration int64) {
	key := append(nodeBlockIPPrefix, ip...)
	db.StoreInt64(key, expiration)
}

func (db *DB) BlockId(id vnode.NodeID, expiration int64) {
	key := append(nodeBlockIDPrefix, id.Bytes()...)
	db.StoreInt64(key, expiration)
}

// RetrieveNode Node according to the special nodeID
func (db *DB) RetrieveNode(id vnode.NodeID) (node *vnode.Node, err error) {
	key := append(nodeDataPrefix, id.Bytes()...)
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
func (db *DB) StoreNode(node *vnode.Node) (err error) {
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
func (db *DB) RemoveNode(ID vnode.NodeID) {
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

// ReadNodes from database, if time.Now().Unix() - node.activeAt < expiration
func (db *DB) ReadNodes(expiration int64) (nodes []*vnode.Node) {
	itr := db.NewIterator(util.BytesPrefix(nodeActivePrefix), nil)
	defer itr.Release()

	now := time.Now().Unix()
	prefixLen := len(nodeActivePrefix)

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		// too old
		active := decodeVarint(itr.Value())
		if now-active > expiration {
			db.RemoveNode(id)
			continue
		}

		node, err := db.RetrieveNode(id)
		if err != nil {
			db.RemoveNode(id)
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func (db *DB) RetrieveInt64(key []byte) int64 {
	buf, err := db.Get(key, nil)
	if err != nil {
		return 0
	}

	return decodeVarint(buf)
}

func (db *DB) StoreInt64(key []byte, n int64) {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]

	_ = db.Put(key, buf, nil)
}

// Clean nodes if time.Now().Unix() - node.activeAt > expiration
func (db *DB) Clean(expiration int64) {
	itr := db.NewIterator(util.BytesPrefix(nodeActivePrefix), nil)
	defer itr.Release()

	now := time.Now().Unix()
	prefixLen := len(nodeActivePrefix)

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		active := decodeVarint(itr.Value())
		if now-active > expiration {
			db.RemoveNode(id)
			continue
		}
	}
}

type mark struct {
	id     vnode.NodeID
	weight int64
}

type marks []mark

func (ms marks) Len() int {
	return len(ms)
}

func (ms marks) Less(i, j int) bool {
	return ms[i].weight > ms[j].weight
}

func (ms marks) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}

func (db *DB) ReadMarkNodes(n int) (nodes []*vnode.Node) {
	itr := db.NewIterator(util.BytesPrefix(nodeMarkPrefix), nil)
	defer itr.Release()

	var ms marks
	prefixLen := len(nodeMarkPrefix)

	now := time.Now().Unix()

	for itr.Next() {
		key := itr.Key()
		id, err := vnode.Bytes2NodeID(key[prefixLen:])
		if err != nil {
			_ = db.Delete(key, nil)
			continue
		}

		data := itr.Value()
		if len(data) < 16 {
			_ = db.Delete(key, nil)
			continue
		}

		markValue := int64(binary.BigEndian.Uint64(data[:8]))
		markAt := int64(binary.BigEndian.Uint64(data[8:]))

		// 7d
		if now-markAt > 24*3600*7 {
			_ = db.Delete(key, nil)
			continue
		}

		ms = append(ms, mark{id, markValue})
	}

	sort.Sort(ms)

	if len(ms) > n {
		ms = ms[:n]
	}

	var err error
	var node *vnode.Node
	for _, m := range ms {
		node, err = db.RetrieveNode(m.id)
		if err != nil {
			db.RemoveNode(m.id)
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func (db *DB) Iterate(prefix []byte, fn func(key, value []byte) bool) {
	itr := db.NewIterator(util.BytesPrefix(nodeMarkPrefix), nil)
	defer itr.Release()

	for itr.Next() {
		if fn(itr.Key(), itr.Value()) == false {
			break
		}
	}
}

func (db *DB) Register(prefix []byte) *prefixDB {
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
