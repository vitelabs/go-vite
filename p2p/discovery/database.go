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
)

type nodeDB struct {
	db *leveldb.DB
	id NodeID
}

const (
	dbVersion  = "version"
	dbPrefix   = "n:"
	dbDiscover = "discover"
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
	nilID := NodeID{}
	if id == nilID {
		return []byte(field)
	}
	data := append([]byte(dbPrefix), id[:]...)
	return append(data, field...)
}

func parseKey(key []byte) (id NodeID, field string) {
	if bytes.HasPrefix(key, []byte(dbPrefix)) {
		rest := key[len(dbPrefix):]
		idLength := len(id)
		copy(id[:], rest[:idLength])
		return id, string(key[idLength:])
	}
	return NodeID{}, string(key)
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
	data, err := db.db.Get(genKey(ID, dbDiscover), nil)
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
	key := genKey(node.ID, dbDiscover)
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

func (db *nodeDB) randomNodes(count int) []*Node {
	iterator := db.db.NewIterator(nil, nil)
	defer iterator.Release()

	nodes := make([]*Node, 0, count)
	var id NodeID

	for i := 0; len(nodes) < count && i < count*5; i++ {
		h := id[0]
		rand.Read(id[:])
		id[0] = h + id[0]%16

		iterator.Seek(genKey(id, dbDiscover))

		node := nextNode(iterator)

		if node == nil {
			id[0] = 0
			continue
		}
		if contains(nodes, node) || node.ID == db.id {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes
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

		if field != dbDiscover {
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
