package dex

import (
	"container/list"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var orderInfos map[int]*nodePayload
var usedNodes map[int]*list.Element
var freeBuffer = list.New() // *nodePayload
var usedBuffer = list.New() // *NodeWithBufferIndex

var storage BaseStorage
var po nodePayloadProtocol = &OrderNodeProtocol{}

const idRange int = 100000
const batchOrderCount int = 400 //4000
const priceRange int = 50000
const maxTruncateLength = 6 // 53
const normalTruncateCeil = 3 // 13
const loopTimes = 30000    //300000

var emptyCount = 0
var deleteToEmptyCount = 0
var truncateToEmptyCount = 0
var fullCount = 0
var truncateMaxCount = 0
var shouldClear = false
var factor float32 = 0.8 // should bigger than 0.5
var factors = []float32{0.8, 0.82, 0.84, 0.86, 0.878, 0.892, 0.90, 0.91, 0.922, 0.933, 0.945, 0.95}

const (
	AddOne = iota
	DeleteOne
	Truncate
)

func TestBatchSkiplist(t *testing.T) {
	orderInfos = make(map[int]*nodePayload, batchOrderCount)
	usedNodes = make(map[int]*list.Element, batchOrderCount)
	batchGenerateOrderInfo()
	initStorage()
	reverseCount := 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < loopTimes; i++ {
		if randAction(t) { // check reverse is true
			reverseCount++
			if reverseCount%5 == 0 {
				rand.Seed(time.Now().UnixNano())
				fmt.Printf("reverseCount %d, changeFactor\n", reverseCount)
				factorRandInt := rand.Intn(len(factors))
				factor = factors[factorRandInt]
			}
		}
	}
	fmt.Printf("emptyCount %d, fullCount %d, deleteToEmptyCount %d, truncateToEmptyCount %d, truncateMaxCount %d\n", emptyCount, fullCount, deleteToEmptyCount, truncateToEmptyCount, truncateMaxCount)
}

func initStorage() {
	localStorage := NewMapStorage()
	storage = BaseStorage(&localStorage)
}

func batchGenerateOrderInfo() {
	for ; len(orderInfos) < batchOrderCount; {
		id := rand.Intn(idRange) + 1
		if _, ok := orderInfos[id]; !ok {
			price := rand.Intn(priceRange) + 1
			_, payload := newNodeInfoWithPrice(id, strconv.Itoa(price))
			orderInfos[id] = payload
			freeBuffer.PushBack(id)
		}
	}
}

func randAction(t *testing.T) bool {
	reverse := false
	actionInt := rand.Intn(3)
	switch actionInt {
	case AddOne:
		if shouldInsert() {
			if freeBuffer.Len() > 0 {
				insertNode(t)
				if freeBuffer.Len() == 0 {
					fullCount++
					shouldClear = true
					reverse = true
				}
			}
		}
	case DeleteOne, Truncate:
		if shouldDelete() {
			if usedBuffer.Len() > 0 {
				if actionInt == DeleteOne {
					deleteNode(t)
				} else {
					truncateNodes(t)
				}
				if usedBuffer.Len() == 0 {
					emptyCount++
					shouldClear = false
					reverse = true
					if actionInt == DeleteOne {
						deleteToEmptyCount++
					} else {
						truncateToEmptyCount++
					}
				}
			}
		}
	}
	skl := getSkipList(t)
	assert.Equal(t, batchOrderCount, usedBuffer.Len()+freeBuffer.Len())
	assert.Equal(t, int(skl.length), usedBuffer.Len())
	assert.Equal(t, len(usedNodes), usedBuffer.Len())
	if skl.length > 0 {
		assert.True(t, !skl.header.isNilKey())
		assert.True(t, !skl.tail.isNilKey())
	} else {
		assert.Equal(t, int8(1), skl.level)
		assert.True(t, skl.header.isNilKey())
		assert.True(t, skl.tail.isNilKey())
	}
	return reverse
}

func insertNode(t *testing.T) {
	id := randomAdd(t)
	skl := getSkipList(t)
	err := skl.insert(orderIdFromInt(id), orderInfos[id])
	assert.True(t, err == nil)
}

func deleteNode(t *testing.T) {
	id := randomDelete(t)
	skl := getSkipList(t)
	err := skl.delete(orderIdFromInt(id))
	assert.True(t, err == nil)
}

func truncateNodes(t *testing.T) {
	skl := getSkipList(t)
	truncateLength := rand.Intn(maxTruncateLength) + 1
	if truncateLength == maxTruncateLength && truncateLength <= int(skl.length) {
		truncateMaxCount++
		//fmt.Printf("truncateMaxCount %d\n", truncateMaxCount)
	} else {
		truncateLength = rand.Intn(normalTruncateCeil) + 1
		if truncateLength >= int(skl.length) {
			truncateLength = int(skl.length)
		}
	}
	//fmt.Printf("truncateNodes truncateLength %d\n", truncateLength)
	current := skl.header
	ids := make([]int, truncateLength)
	for i := 0; i < truncateLength; i++ {
		orderId := current.(OrderId)
		id := fromOrderIdToInt(orderId)
		//fmt.Printf("truncateNodes id %d\n", id)
		ids[i] = id
		node, err := skl.getNode(current)
		assert.True(t, err == nil)
		assert.True(t, node != nil)
		current = node.forwardOnLevel[0]
	}
	doTruncate(t, ids)
}

func randomAdd(t *testing.T) int {
	id, element := randomMoveOne(t, freeBuffer, usedBuffer)
	//fmt.Printf("randomAdd id %d\n", id)
	assert.True(t, id > 0)
	usedNodes[id] = element
	return id
}

func randomDelete(t *testing.T) int {
	id, _ := randomMoveOne(t, usedBuffer, freeBuffer)
	assert.True(t, id > 0)
	delete(usedNodes, id)
	return id
}

func randomMoveOne(t *testing.T, fromBuffer, toBuffer *list.List) (id int, element *list.Element) {
	assert.True(t, fromBuffer.Len() > 0)
	index := rand.Intn(fromBuffer.Len())
	//fmt.Printf("randomMoveOne index %d, fromBuffer.Len() %d\n", index, fromBuffer.Len())
	current := fromBuffer.Front()
	for i := 0; i <= index; i++ {
		//fmt.Printf("%d, id %d\n", i, current.Value.(int))
		if i == index {
			fromBuffer.Remove(current)
			element = toBuffer.PushBack(current.Value.(int))
			return current.Value.(int), element
		} else {
			current = current.Next()
		}
	}
	return -1, nil
}

func doTruncate(t *testing.T, ids []int) {
	for _, id := range ids {
		assert.True(t, id > 0)
		node, ok := usedNodes[id]
		assert.True(t, ok)
		delete(usedNodes, id)
		nodeId := usedBuffer.Remove(node)
		assert.Equal(t, id, nodeId)
		freeBuffer.PushBack(id)
	}
	length := len(ids)
	skl := getSkipList(t)
	err := skl.truncateHeadTo(orderIdFromInt(ids[length-1]), int32(length))
	assert.True(t, err == nil)
}

func getSkipList(t *testing.T) *skiplist {
	listId := SkipListId{}
	listId.SetBytes([]byte("testBatchSkiplist"))
	add, _ := types.BytesToAddress([]byte("123456789A123456789A"))
	skiplist, err := newSkiplist(listId, &add, &storage, &po)
	assert.Equal(t, nil, err)
	return skiplist
}

func shouldInsert() bool {
	return !shouldClear || shouldClear && rand.Float32() > factor
}

func shouldDelete() bool {
	return shouldClear || !shouldClear && rand.Float32() > factor
}
