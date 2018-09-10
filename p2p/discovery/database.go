package discovery

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"time"
)

type nodeDB struct {
	db *leveldb.DB
	id NodeID
}

const (
	dbVersion  = "version"
	dbPrefix   = "discv"
	dbNode     = "node"
	dbPing     = "ping"
	dbPong     = "pong"
	dbFindFail = "findfail"
)

func newDB(path string, version int, id NodeID) (*nodeDB, error) {
	if path == "" {
		return newMemDB(id)
	} else {
		return newFileDB(path, version, id)
	}
}

func newMemDB(id NodeID) (*nodeDB, error) {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &nodeDB{
		db: db,
		id: id,
	}, nil
}

func newFileDB(path string, version int, id NodeID) (*nodeDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if _, ok := err.(*errors.ErrCorrupted); ok {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if err != nil {
		return nil, err
	}

	vBytes := encodeVarint(int64(version))
	oldVBytes, err := db.Get([]byte(dbVersion), nil)

	if err == leveldb.ErrNotFound {
		err = db.Put([]byte(dbVersion), vBytes, nil)

		if err != nil {
			db.Close()
			return nil, err
		} else {
			return &nodeDB{
				db: db,
				id: id,
			}, nil
		}
	} else if err == nil {
		if bytes.Equal(oldVBytes, vBytes) {
			return &nodeDB{
				db: db,
				id: id,
			}, err
		} else {
			db.Close()
			err = os.RemoveAll(path)
			if err != nil {
				return nil, err
			}
			return newFileDB(path, version, id)
		}
	}

	return nil, err
}

func genKey(id NodeID, field string) []byte {
	if id.IsZero() {
		return []byte(field)
	}

	return bytes.Join([][]byte{
		[]byte(dbPrefix),
		id[:],
		[]byte(field),
	}, nil)
}

func parseKey(key []byte) (id NodeID, field string) {
	if bytes.HasPrefix(key, []byte(dbPrefix)) {
		prefixLen := len(dbPrefix)
		idLength := len(id)
		headLength := prefixLen + idLength

		copy(id[:], key[idLength:headLength])
		return id, string(key[headLength:])
	}

	return id, string(key)
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

func (db *nodeDB) retrieveNode(ID NodeID) *Node {
	data, err := db.db.Get(genKey(ID, dbNode), nil)
	if err != nil {
		return nil
	}

	node := new(Node)
	err = node.Deserialize(data)
	if err != nil {
		return nil
	}
	return node
}

func (db *nodeDB) updateNode(node *Node) error {
	key := genKey(node.ID, dbNode)
	data, err := node.Serialize()
	if err != nil {
		return err
	}
	return db.db.Put(key, data, nil)
}

// remove all data about the specific NodeID
func (db *nodeDB) deleteNode(ID NodeID) error {
	itr := db.db.NewIterator(util.BytesPrefix(genKey(ID, "")), nil)
	defer itr.Release()

	for itr.Next() {
		err := db.db.Delete(itr.Key(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *nodeDB) randomNodes(count int, maxAge time.Duration) []*Node {
	iterator := db.db.NewIterator(nil, nil)
	defer iterator.Release()

	nodes := make([]*Node, 0, count)
	var id NodeID
	now := time.Now()

	for i := 0; len(nodes) < count && i < count*5; i++ {
		h := id[0]
		rand.Read(id[:])
		id[0] = h + id[0]%16

		iterator.Seek(genKey(id, dbNode))

		node := nextNode(iterator)

		if node == nil {
			id[0] = 0
			continue
		}

		if contains(nodes, node) || node.ID == db.id {
			continue
		}

		if now.Sub(db.getLastPong(node.ID)) > maxAge {
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func (db *nodeDB) retrieveInt64(key []byte) int64 {
	buf, err := db.db.Get(key, nil)
	if err != nil {
		return 0
	}
	val, read := binary.Varint(buf)
	if read <= 0 {
		return 0
	}
	return val
}

func (db *nodeDB) storeInt64(key []byte, n int64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]

	return db.db.Put(key, buf, nil)
}

// get the last time when receive ping msg from id
func (db *nodeDB) getLastPing(id NodeID) time.Time {
	return time.Unix(db.retrieveInt64(genKey(id, dbPing)), 0)
}

// set the last time when receive ping msg from id
func (db *nodeDB) setLastPing(id NodeID, instance time.Time) error {
	return db.storeInt64(genKey(id, dbPing), instance.Unix())
}

// get the last time when receive pong msg from id
func (db *nodeDB) getLastPong(id NodeID) time.Time {
	return time.Unix(db.retrieveInt64(genKey(id, dbPong)), 0)
}

// set the last time when receive pong msg from id
func (db *nodeDB) setLastPong(id NodeID, instance time.Time) error {
	return db.storeInt64(genKey(id, dbPong), instance.Unix())
}

// in the last 24 hours, id has been pingpong checked
func (db *nodeDB) hasChecked(id NodeID) bool {
	return time.Since(db.getLastPong(id)) < tExpire
}

// get the times of findnode from id fails
func (db *nodeDB) getFindNodeFails(id NodeID) int {
	return int(db.retrieveInt64(genKey(id, dbFindFail)))
}

// set the times of findnode from id fails
func (db *nodeDB) setFindNodeFails(id NodeID, fails int) error {
	return db.storeInt64(genKey(id, dbFindFail), int64(fails))
}

func (db *nodeDB) close() {
	db.db.Close()
}

// helper functions
func contains(nodes []*Node, node *Node) bool {
	for _, n := range nodes {
		if n != nil && n.ID == node.ID {
			return true
		}
	}
	return false
}

func nextNode(iterator iterator.Iterator) *Node {
	var field string
	var data []byte
	var err error

	node := new(Node)
	for iterator.Next() {
		_, field = parseKey(iterator.Key())

		if field != dbNode {
			continue
		}

		data = iterator.Value()
		err = node.Deserialize(data)

		if err != nil {
			continue
		}

		return node
	}

	return nil
}
