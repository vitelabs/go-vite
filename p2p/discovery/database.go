package discovery

import (
	"bytes"
	"encoding/binary"
	"os"
	"time"

	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type nodeDB struct {
	db *leveldb.DB
	id NodeID
}

var (
	versionKey = []byte("version")
	nodePrefix = []byte("n")
	nodeKey    = []byte(":node")
	markKey    = []byte(":mark")
	activeKey  = []byte(":active")
	pingKey    = []byte(":ping")
	findKey    = []byte(":find")
)

func newDB(path string, version int, id NodeID) (db *nodeDB, err error) {
	if path == "" {
		db, err = newMemDB(id)
	} else {
		db, err = newFileDB(path, version, id)
	}

	if err != nil {
		return
	}

	return
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
	oldVBytes, err := db.Get(versionKey, nil)

	if err == leveldb.ErrNotFound {
		err = db.Put(versionKey, vBytes, nil)

		if err != nil {
			db.Close()
			return nil, err
		}
		return &nodeDB{
			db: db,
			id: id,
		}, nil
	} else if err == nil {
		if bytes.Equal(oldVBytes, vBytes) {
			return &nodeDB{
				db: db,
				id: id,
			}, err
		}

		db.Close()
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

func (db *nodeDB) retrieveNode(ID NodeID) *Node {
	key := bytes.Join([][]byte{nodePrefix, nodeKey, ID[:]}, nil)
	// retrieve node
	data, err := db.db.Get(key, nil)
	if err != nil {
		return nil
	}

	node := new(Node)
	if err = node.Deserialize(data); err != nil {
		return nil
	}

	// retrieve mark
	key = bytes.Join([][]byte{nodePrefix, markKey, ID[:]}, nil)
	node.mark = db.retrieveInt64(key)
	// retrieve active
	key = bytes.Join([][]byte{nodePrefix, activeKey, ID[:]}, nil)
	node.activeAt = time.Unix(db.retrieveInt64(key), 0)
	// retrieve ping
	key = bytes.Join([][]byte{nodePrefix, pingKey, ID[:]}, nil)
	node.lastPing = time.Unix(db.retrieveInt64(key), 0)
	// retrieve find
	key = bytes.Join([][]byte{nodePrefix, findKey, ID[:]}, nil)
	node.lastFind = time.Unix(db.retrieveInt64(key), 0)

	return node
}

func (db *nodeDB) storeNode(node *Node) {
	data, err := node.Serialize()
	if err != nil {
		return
	}

	// store node
	key := bytes.Join([][]byte{nodePrefix, nodeKey, node.ID[:]}, nil)
	err = db.db.Put(key, data, nil)
	if err != nil {
		return
	}

	// store mark
	key = bytes.Join([][]byte{nodePrefix, markKey, node.ID[:]}, nil)
	db.storeInt64(key, node.mark)
	// store active
	key = bytes.Join([][]byte{nodePrefix, activeKey, node.ID[:]}, nil)
	db.storeInt64(key, node.activeAt.Unix())
	// store ping
	key = bytes.Join([][]byte{nodePrefix, pingKey, node.ID[:]}, nil)
	db.storeInt64(key, node.lastPing.Unix())
	// store find
	key = bytes.Join([][]byte{nodePrefix, findKey, node.ID[:]}, nil)
	db.storeInt64(key, node.lastFind.Unix())
}

// deleteNode data about the specific NodeID
func (db *nodeDB) deleteNode(ID NodeID) {
	for _, field := range [][]byte{nodeKey, markKey, activeKey, findKey, pingKey} {
		key := bytes.Join([][]byte{nodePrefix, field, ID[:]}, nil)
		db.db.Delete(key, nil)
	}
}

func (db *nodeDB) randomNodes(count int, maxAge time.Duration) []*Node {
	prefix := bytes.Join([][]byte{nodePrefix, activeKey}, nil)
	prefixLen := len(prefix)

	itr := db.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer itr.Release()

	nodes := make([]*Node, 0, count)
	now := time.Now()
	var id NodeID

	for itr.Next() {
		key := itr.Key()
		active := time.Unix(db.retrieveInt64(key), 0)

		if now.Sub(active) < maxAge {
			copy(id[:], key[prefixLen:])
			if node := db.retrieveNode(id); node != nil {
				nodes = append(nodes, node)
			} else {
				db.deleteNode(id)
			}
		}

		if len(nodes) > count {
			break
		}
	}

	return nodes
}

func (db *nodeDB) retrieveInt64(key []byte) int64 {
	buf, err := db.db.Get(key, nil)
	if err != nil {
		return 0
	}

	return decodeVarint(buf)
}

func (db *nodeDB) storeInt64(key []byte, n int64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]

	return db.db.Put(key, buf, nil)
}

func (db *nodeDB) cleanStaleNodes() {
	now := time.Now()

	prefix := bytes.Join([][]byte{nodePrefix, activeKey}, nil)
	prefixLen := len(prefix)

	itr := db.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer itr.Release()

	var id NodeID

	for itr.Next() {
		key := itr.Key()
		active := time.Unix(db.retrieveInt64(key), 0)

		if now.Sub(active) > seedMaxAge {
			copy(id[:], key[prefixLen:])
			db.deleteNode(id)
		}
	}
}

func (db *nodeDB) close() {
	db.db.Close()
}
