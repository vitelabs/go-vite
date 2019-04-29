package dex

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"math/rand"
	"testing"
)

type localMapStorage struct {
	data map[string][]byte
	logs []*ledger.VmLog
}

func NewMapStorage() localMapStorage {
	return localMapStorage{data : make(map[string][]byte, 0), logs : make([]*ledger.VmLog, 0, 10)}
}

func (ls *localMapStorage) GetValue(key []byte) ([]byte, error) {
	if v, ok := ls.data[string(key)]; ok {
		return v, nil
	} else {
		return nil, nil
	}
}

func (ls *localMapStorage) SetValue(key []byte, value []byte) error {
	ls.data[string(key)] = value
	return nil
}

func (ls *localMapStorage) ClearStorage(key []byte, value []byte) {
	for k, _ := range ls.data {
		delete(ls.data, k)
	}
}

func (ls *localMapStorage) AddLog(log *ledger.VmLog) {
	ls.logs = append(ls.logs, log)
}

func (ls *localMapStorage) GetLogListHash() *types.Hash {
	if len(ls.logs) == 0 {
		return nil
	}
	var source []byte

	// Nonce
	for _, vmLog := range ls.logs {
		for _, topic := range vmLog.Topics {
			source = append(source, topic.Bytes()...)
		}
		source = append(source, vmLog.Data...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash
	return nil
}

var baseKeyInt, key2Int, key5Int, key1Int, key3Int int

func TestSkiplist(t *testing.T) {
	skiplist := newSkipListTest(t)
	for i := 0; i < 100; i++ { // run multi time for random level case for every node
		baseKeyInt = rand.Intn(100)
		key2Int = baseKeyInt + 2
		key5Int = baseKeyInt + 5
		key1Int = baseKeyInt + 1
		key3Int = baseKeyInt + 3
		batchTest(t, skiplist)
	}
}

func batchTest(t *testing.T, skiplist *skiplist) {
	insertNodes(skiplist)
	insertTest(t, skiplist)
	deleteTest(t, skiplist)
	insertNodes(skiplist)
	insertTest(t, skiplist)
	truncateTest(t, skiplist)
	//fmt.Printf("Finish batchTest\n\n\n")
}

func newSkipListTest (t *testing.T) *skiplist {
	localStorage := NewMapStorage()
	st := BaseStorage(&localStorage)
	var po nodePayloadProtocol = &OrderNodeProtocol{}

	// Test new
	listId := SkipListId{}
	listId.SetBytes([]byte("skiplistName"))
	skiplist, _ := newSkiplist(listId, &st, &po)
	assert.Equal(t, skiplist.level, int8(1))
	return skiplist
}

func insertNodes(skiplist *skiplist) {
	key2, payload2 := newNodeInfo(key2Int)
	skiplist.insert(key2, payload2)

	key5, payload5 := newNodeInfo(key5Int)
	skiplist.insert(key5, payload5)

	key1, payload1 := newNodeInfo(key1Int)
	skiplist.insert(key1, payload1)

	key3, payload3 := newNodeInfo(key3Int)
	skiplist.insert(key3, payload3)
}

func insertTest(t *testing.T, skiplist *skiplist) {
	// Test Insert
	assert.Equal(t, int32(4), skiplist.length)
	assert.Equal(t, key5Int, fromOrderIdToInt(skiplist.header))
	assert.Equal(t, key1Int, fromOrderIdToInt(skiplist.tail))

	var pl *nodePayload
	var fwk, bwk nodeKeyType
	pl, fwk, bwk, _ = skiplist.peek()
	assert.NotNil(t, pl)
	var od Order

	od, _ = (*pl).(Order)
	assert.Equal(t, key5Int, fromOrderIdBytesToInt(od.Id))
	assert.Equal(t, key3Int, fromOrderIdToInt(fwk))
	assert.True(t, bwk.isBarrierKey())

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, key3Int, fromOrderIdBytesToInt(od.Id))
	assert.Equal(t, key2Int, fromOrderIdToInt(fwk))
	assert.Equal(t, key5Int, fromOrderIdToInt(bwk))

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, key2Int, fromOrderIdBytesToInt(od.Id))
	assert.Equal(t, key1Int, fromOrderIdToInt(fwk))
	assert.Equal(t, key3Int, fromOrderIdToInt(bwk))

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, key1Int, fromOrderIdBytesToInt(od.Id))
	assert.True(t, fwk.isNilKey())
	assert.Equal(t, key2Int, fromOrderIdToInt(bwk))
}

func deleteTest(t *testing.T, skiplist *skiplist) {
	var fwk, bwk nodeKeyType
	var err error
	err = skiplist.delete(orderIdFromInt(key2Int))

	checkLevel(t, skiplist)
	assert.True(t, err == nil)
	assert.Equal(t, int32(3), skiplist.length)
	_, fwk, bwk, _ = skiplist.getByKey(orderIdFromInt(key3Int))
	assert.Equal(t, key1Int, fromOrderIdToInt(fwk))

	_, fwk, bwk, _ = skiplist.getByKey(orderIdFromInt(key1Int))
	assert.Equal(t, key3Int, fromOrderIdToInt(bwk))

	skiplist.delete(orderIdFromInt(key1Int))
	checkLevel(t, skiplist)
	_, fwk, bwk, _ = skiplist.getByKey(orderIdFromInt(key3Int))
	assert.True(t, fwk.isNilKey())
	assert.Equal(t, key3Int, fromOrderIdToInt(skiplist.tail))

	skiplist.delete(orderIdFromInt(key5Int))
	checkLevel(t, skiplist)
	_, fwk, bwk, _ = skiplist.getByKey(orderIdFromInt(key3Int))
	assert.True(t, bwk.isBarrierKey())
	assert.Equal(t, key3Int, fromOrderIdToInt(skiplist.header))

	skiplist.delete(orderIdFromInt(key3Int))
	assert.True(t, skiplist.length == 0)
	checkLevel(t, skiplist)
	assert.Equal(t, 0, fromOrderIdToInt(skiplist.header))
	assert.Equal(t, 0, fromOrderIdToInt(skiplist.tail))
}

func truncateTest(t *testing.T, skiplist *skiplist) {
	var err error
	var fwk, bwk nodeKeyType

	err = skiplist.truncateHeadTo(orderIdFromInt(key2Int), 3)
	checkLevel(t, skiplist)
	assert.True(t, err == nil)
	assert.Equal(t, int32(1), skiplist.length)
	assert.Equal(t, key1Int, fromOrderIdToInt(skiplist.header))
	assert.Equal(t, key1Int, fromOrderIdToInt(skiplist.tail))

	_, fwk, bwk, _ = skiplist.getByKey(orderIdFromInt(key1Int))
	assert.True(t, bwk.isBarrierKey())
	assert.True(t, fwk.isNilKey())

	err = skiplist.truncateHeadTo(orderIdFromInt(key1Int), 1)
	checkLevel(t, skiplist)
	assert.Equal(t, int32(0), skiplist.length)
	assert.Equal(t, 0, fromOrderIdToInt(skiplist.header))
	assert.Equal(t, 0, fromOrderIdToInt(skiplist.tail))
}

func checkLevel(t *testing.T, skiplist *skiplist) {
	if skiplist.length == 0 {
		assert.Equal(t, 1, int(skiplist.level))
	} else {
		key := skiplist.header
		assert.True(t, !key.isNilKey())
		maxLevel := 0
		for ;; {
			node, err := skiplist.getNode(key)
			assert.Equal(t, nil, err)
			if maxLevel < len(node.forwardOnLevel) {
				maxLevel = len(node.forwardOnLevel)
			}
			key = node.forwardOnLevel[0]
			if key.isNilKey() {
				break
			}
		}
		assert.Equal(t, int8(maxLevel), skiplist.level)
	}
}

func orderIdFromInt(v int) OrderId {
	key := OrderId{}
	key.setBytes(orderIdBytesFromInt(v))
	return key
}
