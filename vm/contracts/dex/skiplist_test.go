package dex

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

type localMapStorage struct {
	data map[string][]byte
	logs []*ledger.VmLog
}

func NewMapStorage() localMapStorage {
	return localMapStorage{data : make(map[string][]byte, 0), logs : make([]*ledger.VmLog, 0, 10)}
}

func (ls *localMapStorage) GetStorage(addr *types.Address, key []byte) []byte {
	if v, ok := ls.data[string(key)]; ok {
		return v
	} else {
		return nil
	}
}

func (ls *localMapStorage) SetStorage(key []byte, value []byte) {
	ls.data[string(key)] = value
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

func getAddress() *types.Address {
	add, _ := types.BytesToAddress([]byte("12345678901234567890"))
	return &add
}

func TestSkiplist(t *testing.T) {
	skiplist := newSkipListTest(t)
	insertNodes(skiplist)
	insertTest(t, skiplist)
	deleteTest(t, skiplist)
	clearStorage(skiplist)
	insertNodes(skiplist)
	truncateTest(t, skiplist)
}

func newSkipListTest (t *testing.T) *skiplist {
	localStorage := NewMapStorage()
	st := baseStorage(&localStorage)
	var po nodePayloadProtocol = &OrderNodeProtocol{}

	// Test new
	skiplist := newSkiplist("listName", getAddress(), &st, &po)
	assert.Equal(t, skiplist.level, int8(1))
	return skiplist
}

func insertNodes(skiplist *skiplist) {
	key2, payload2 := newNodeInfo(2)
	skiplist.insert(key2, payload2)

	key5, payload5 := newNodeInfo(5)
	skiplist.insert(key5, payload5)

	key1, payload1 := newNodeInfo(1)
	skiplist.insert(key1, payload1)

	key3, payload3 := newNodeInfo(3)
	skiplist.insert(key3, payload3)
}

func clearStorage(skiplist *skiplist) {
	localStorage := NewMapStorage()
	st := baseStorage(&localStorage)
	skiplist.storage = &st
}

func insertTest(t *testing.T, skiplist *skiplist) {
	// Test Insert
	assert.Equal(t, 4, skiplist.length)
	assert.Equal(t, uint64(5), skiplist.header.(orderKey).value)
	assert.Equal(t, uint64(1), skiplist.tail.(orderKey).value)

	var pl *nodePayload
	var fwk, bwk nodeKeyType
	pl, fwk, bwk, _ = skiplist.peek()
	assert.NotNil(t, pl)
	var od Order

	od, _ = (*pl).(Order)
	assert.Equal(t, uint64(5), od.Id)
	assert.Equal(t, uint64(3), fwk.(orderKey).value)
	assert.True(t, bwk.isHeader())

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, uint64(3), od.Id)
	assert.Equal(t, uint64(2), fwk.(orderKey).value)
	assert.Equal(t, uint64(5), bwk.(orderKey).value)

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, uint64(2), od.Id)
	assert.Equal(t, uint64(1), fwk.(orderKey).value)
	assert.Equal(t, uint64(3), bwk.(orderKey).value, )

	pl, fwk, bwk, _ = skiplist.getByKey(fwk)
	od, _ = (*pl).(Order)
	assert.Equal(t, uint64(1), od.Id)
	assert.True(t, fwk.isNil())
	assert.Equal(t, uint64(2), bwk.(orderKey).value)
}

func deleteTest(t *testing.T, skiplist *skiplist) {
	var fwk, bwk nodeKeyType
	var err error

	err = skiplist.delete(newOrderKey(2))
	assert.True(t, err == nil)
	assert.Equal(t, 3, skiplist.length)

	_, fwk, bwk, _ = skiplist.getByKey(newOrderKey(3))
	assert.Equal(t, uint64(1), fwk.(orderKey).value)

	_, fwk, bwk, _ = skiplist.getByKey(newOrderKey(1))
	assert.Equal(t, uint64(3), bwk.(orderKey).value)

	skiplist.delete(newOrderKey(1))
	_, fwk, bwk, _ = skiplist.getByKey(newOrderKey(3))
	assert.True(t, fwk.isNil())
	assert.Equal(t, uint64(3), skiplist.tail.(orderKey).value)

	skiplist.delete(newOrderKey(5))
	_, fwk, bwk, _ = skiplist.getByKey(newOrderKey(3))
	assert.True(t, bwk.isHeader())
	assert.Equal(t, uint64(3), skiplist.header.(orderKey).value)

	skiplist.delete(newOrderKey(3))
	assert.True(t, skiplist.length == 0)
	assert.Equal(t, uint64(0), skiplist.header.(orderKey).value)
	assert.Equal(t, uint64(0), skiplist.tail.(orderKey).value)
}

func truncateTest(t *testing.T, skiplist *skiplist) {
	var err error
	var fwk, bwk nodeKeyType

	err = skiplist.truncateHeadTo(newOrderKey(2), 3)
	assert.True(t, err == nil)
	assert.Equal(t, 1, skiplist.length)
	assert.Equal(t, uint64(1), skiplist.header.(orderKey).value)
	assert.Equal(t, uint64(1), skiplist.tail.(orderKey).value)

	_, fwk, bwk, _ = skiplist.getByKey(newOrderKey(1))
	assert.True(t, bwk.isHeader())
	assert.True(t, fwk.isNil())

	err = skiplist.truncateHeadTo(newOrderKey(1), 1)
	assert.Equal(t, 0, skiplist.length)
	assert.Equal(t, uint64(0), skiplist.header.(orderKey).value)
	assert.Equal(t, uint64(0), skiplist.tail.(orderKey).value)
}
