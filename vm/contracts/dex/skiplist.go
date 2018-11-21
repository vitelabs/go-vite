package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/rand"
)

const skiplistMaxLevel int8 = 13 // 2^13 8192
const metaStorageSalt = "listMeta:"

type nodeKeyType interface {
	getStorageKey() []byte
	isNil() bool
	isHeader() bool
	equals(nodeKeyType) bool
	toString() string
}

type nodePayload interface {
	compareTo(payload *nodePayload) int8
}

type nodePayloadProtocol interface {
	getNilKey() nodeKeyType
	getHeaderKey() nodeKeyType
	serialize(node *skiplistNode) []byte
	deSerialize(nodeData []byte) *skiplistNode
	serializeMeta(meta *skiplistMeta) []byte
	deSerializeMeta(nodeData []byte) *skiplistMeta
}

type BaseStorage interface {
	GetStorage(addr *types.Address, key []byte) []byte
	SetStorage(key []byte, value []byte)
	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash
}

type skiplistNode struct {
	nodeKey nodeKeyType
	payload *nodePayload
	forwardOnLevel []nodeKeyType
	backwardOnLevel []nodeKeyType
}

type skiplist struct {
	name string
	header, tail nodeKeyType
	length int
	level int8
	storage *BaseStorage
	protocol *nodePayloadProtocol
	headerNode *skiplistNode
	contractAddress *types.Address
}

type skiplistMeta struct {
	header, tail nodeKeyType
	length int
	level int8
	forwardOnLevel []nodeKeyType
}

func randomLevel() int8 {
	return int8(rand.Intn(int(skiplistMaxLevel)))+1
}

func (meta *skiplistMeta) fromList(list *skiplist) {
	meta.header = list.header
	meta.tail = list.tail
	meta.length = list.length
	meta.level = list.level
	copy(meta.forwardOnLevel, list.headerNode.forwardOnLevel)
}

func (meta *skiplistMeta) getMetaStorageKey(name string) []byte {
	return []byte(metaStorageSalt + name)
}

func newSkiplist(name string, contractAddress *types.Address, storage *BaseStorage, protocol *nodePayloadProtocol) *skiplist {
	skl := &skiplist{}
	skl.name = name
	skl.header = (*protocol).getNilKey()
	skl.tail = (*protocol).getNilKey()
	skl.length = 0
	skl.level = 1
	skl.storage = storage
	skl.protocol = protocol
	skl.headerNode, _ = skl.createNode((*skl.protocol).getHeaderKey(),nil, skiplistMaxLevel)
	skl.contractAddress = contractAddress
	skl.initMeta(name)
	return skl
}

func (skl *skiplist) createNode(key nodeKeyType, payload *nodePayload, level int8) (*skiplistNode, int8) {
	node := skiplistNode{}
	node.nodeKey = key
	node.payload = payload
	node.forwardOnLevel = make([]nodeKeyType, level)
	for i := 0; i < int(level); i++ {
		node.forwardOnLevel[i] = (*skl.protocol).getNilKey()
	}
	node.backwardOnLevel = make([]nodeKeyType, level)
	for i := 0; i < int(level); i++ {
		node.backwardOnLevel[i] = (*skl.protocol).getNilKey()
	}
	return &node, level
}

func (skl *skiplist) getNodeWithDirtyFilter(nodeKey nodeKeyType, dirtyNodes map[string]*skiplistNode) *skiplistNode {
	if nodeKey == nil || nodeKey.isNil() {
		return nil
	}
	if nodeKey.isHeader() {
		return skl.headerNode
	}
	if len(dirtyNodes) > 0 {
		if node, ok := dirtyNodes[nodeKey.toString()]; ok {
			return node
		}
	}
	nodeData := (*skl.storage).GetStorage(skl.contractAddress, nodeKey.getStorageKey())
	if len(nodeData) != 0 {
		return (*skl.protocol).deSerialize(nodeData)
	} else {
		return nil
	}
}

//TODO whether delete node can be get from storage by key
func (skl *skiplist) getNode(nodeKey nodeKeyType) *skiplistNode {
	return skl.getNodeWithDirtyFilter(nodeKey, nil)
}

func (skl *skiplist) saveNode(node *skiplistNode) error {
	if node == nil || node.nodeKey.isNil() {
		return fmt.Errorf("node to save is invalid")
	}
	if node.nodeKey.isHeader() {
		return nil
	}
	nodeData := (*skl.protocol).serialize(node)
	(*skl.storage).SetStorage(node.nodeKey.getStorageKey(), nodeData)
	return nil
}


func (skl *skiplist) saveMeta() {
	meta := &skiplistMeta{}
	meta.fromList(skl)
	metaData := (*skl.protocol).serializeMeta(meta)
	(*skl.storage).SetStorage(meta.getMetaStorageKey(skl.name), metaData)
}

func (skl *skiplist) initMeta(name string) {
	meta := &skiplistMeta{}
	metaData := (*skl.storage).GetStorage(skl.contractAddress, meta.getMetaStorageKey(name))
	if metaData == nil {
		return
	} else {
		meta = (*skl.protocol).deSerializeMeta(metaData)
		skl.length = meta.length
		skl.header = meta.header
		skl.tail = meta.tail
		skl.level = meta.level
		copy(skl.headerNode.forwardOnLevel, meta.forwardOnLevel)
	}
}

func (nd *skiplistNode) isHeader() bool {
	return nd.nodeKey.isHeader()
}

func (skl *skiplist) insert(key nodeKeyType, payload *nodePayload) {
	//fmt.Printf("enter into insert for %s\n", key.toString())
	var dirtyNodes = make(map[string]*skiplistNode, skiplistMaxLevel)
	var updateNodes = make([]*skiplistNode, skiplistMaxLevel)
	var headerNode = skl.headerNode
	var currentNode = headerNode
	var i int8
	for i = skl.level - 1; i >= 0; i-- {
		// skl is desc list
		for !currentNode.forwardOnLevel[i].isNil() {
		   forwardNode := skl.getNode(currentNode.forwardOnLevel[i])
			if (*payload).compareTo(forwardNode.payload) <= 0 {
				currentNode = forwardNode
			} else {
				break
			}
		}
		updateNodes[i] = currentNode
	}
	level := randomLevel()
	newNode, level := skl.createNode(key, payload, level)
	if level > skl.level {
		for i = skl.level; i < level; i++ {
			updateNodes[i] = headerNode
		}
		skl.level = level
	}
	for i = 0; i < level; i++ {
		forwardKey := updateNodes[i].forwardOnLevel[i]
		newNode.forwardOnLevel[i] = forwardKey
		if !forwardKey.isNil() {
			var forwardNode *skiplistNode
			var ok bool
			if forwardNode, ok = dirtyNodes[forwardKey.toString()]; !ok {
				forwardNode = skl.getNodeWithDirtyFilter(updateNodes[i].forwardOnLevel[i], dirtyNodes)
			}
			forwardNode.backwardOnLevel[i] = key
			dirtyNodes[forwardNode.nodeKey.toString()] = forwardNode
			//fmt.Printf("level %d : set backward to %s for node %s\n", i, key.toString(), forwardNode.nodeKey.toString())
		}
		updateNodes[i].forwardOnLevel[i] = key
		newNode.backwardOnLevel[i] = updateNodes[i].nodeKey
		dirtyNodes[updateNodes[i].nodeKey.toString()] = updateNodes[i]
	}
	dirtyNodes[key.toString()] = newNode
	//fmt.Printf("newNode.forwardOnLevel[0].len %d, level : %d\n", len(newNode.forwardOnLevel), level)
	if newNode.forwardOnLevel[0].isNil() {
		skl.tail = key
	}
	if newNode.backwardOnLevel[0].isHeader() {
		skl.header = key
	}
	skl.length++
	skl.saveMeta()
	skl.saveNodes(dirtyNodes)
}

func (skl *skiplist) delete(key nodeKeyType) error {
	var dirtyNodes = make(map[string]*skiplistNode, skiplistMaxLevel)
	deleteNode := skl.getNode(key)
	if deleteNode == nil {
		return fmt.Errorf("node not exists for %s", key.toString())
	}
	var i int8
	for i = 0; int(i) < len(deleteNode.backwardOnLevel); i++ {
		backwardKey := deleteNode.backwardOnLevel[i]
		var backwardNode *skiplistNode
		if backwardKey.isNil() || backwardKey.isHeader() {
			backwardNode = skl.headerNode
		} else {
			backwardNode = skl.getNodeWithDirtyFilter(deleteNode.backwardOnLevel[i], dirtyNodes)
			if backwardNode == nil {
				return fmt.Errorf("invalid backward node for %s, at index %d", key.toString(), i)
			}
		}
		backwardNode.forwardOnLevel[i] = deleteNode.forwardOnLevel[i]
		dirtyNodes[backwardNode.nodeKey.toString()] = backwardNode
		forwardKey := deleteNode.forwardOnLevel[i]
		if !forwardKey.isNil() {
			forwardNode := skl.getNodeWithDirtyFilter(forwardKey, dirtyNodes)
			forwardNode.backwardOnLevel[i] = backwardKey
			dirtyNodes[forwardNode.nodeKey.toString()] = forwardNode
		}
		if i == 0 {
			if deleteNode.backwardOnLevel[0].isHeader() {
				skl.header = forwardKey
			}
			if deleteNode.forwardOnLevel[0].isNil() {
				// delete last node
				if backwardKey.isHeader() {
					skl.tail = (*skl.protocol).getNilKey()
				} else {
					skl.tail = backwardKey
				}
			}
		}
	}
	skl.length--
	skl.saveMeta()
	skl.saveNodes(dirtyNodes)
	return nil
}

// node of the key is the last to be deleted
// NOTE: key must be exits before truncate
func (skl *skiplist) truncateHeadTo(key nodeKeyType, length int) error {
	splitNode := skl.getNode(key)
	if splitNode == nil {
		return fmt.Errorf("node not exists for %s", key.toString())
	}
	splitNodes := make([]*skiplistNode, skiplistMaxLevel)
	var dirtyNodes = make(map[string]*skiplistNode, skiplistMaxLevel)
	i := 0
	for ;i < len(splitNode.forwardOnLevel); i++ {
		splitNodes[i] = splitNode
	}
	// len(edgeNode.forwardOnLevel) = 4
	// i = 4
	// level = 6
	if i < int(skl.level) {
		backwardNode := skl.getNode(splitNode.backwardOnLevel[i-1])
		if backwardNode == nil {
			return fmt.Errorf("node not exists for %s", splitNode.backwardOnLevel[i-1])
		}
		// find node with higher level
		for ; i < int(skl.level); i++ {
			for ; !backwardNode.isHeader() && len(backwardNode.backwardOnLevel) <= int(i); {
				backwardNode = skl.getNode(backwardNode.backwardOnLevel[i-1])
			}
			if backwardNode.isHeader() {
				break
			}
			for ; i < int(skl.level) && i < len(backwardNode.backwardOnLevel); i++ {
				splitNodes[i] = backwardNode
			}
			i--
		}
	}
	for j := 0 ; j < i; j++ {
		forwardKey := splitNodes[j].forwardOnLevel[j]
		skl.headerNode.forwardOnLevel[j] = splitNodes[j].forwardOnLevel[j]
		if j == 0 {
			skl.header = forwardKey
		}
		if !forwardKey.isNil() {
			forwardNode := skl.getNodeWithDirtyFilter(forwardKey, dirtyNodes)
			forwardNode.backwardOnLevel[j] = skl.headerNode.nodeKey
			dirtyNodes[forwardKey.toString()] = forwardNode
		}
	}
	skl.length -= length
	if skl.length == 0 {
		skl.tail = (*skl.protocol).getNilKey()
	}
	skl.saveMeta()
	skl.saveNodes(dirtyNodes)
	return nil
}

func (skl *skiplist) peek() (pl *nodePayload, forwardKey nodeKeyType, backwardKey nodeKeyType, err error)  {
	if skl.length == 0 {
		return nil, nil, nil, fmt.Errorf("skiplist is empty")
	}
	return skl.getByKey(skl.header)
}

func (skl *skiplist) getByKey(key nodeKeyType) (pl *nodePayload, forwardKey nodeKeyType, backwardKey nodeKeyType, err error) {
	node := skl.getNode(key)
	//fmt.Printf(">>>>>>>>>>>>>> info for key %s\n", key.toString())
	//fmt.Printf("node.forwardOnLevel.len %d, node.backwardOnLevel.len %d\n", len(node.forwardOnLevel), len(node.backwardOnLevel))
	//fmt.Printf("forwardOnLevel\n")
	//for  i , v := range node.forwardOnLevel {
	//	fmt.Printf("%d : %s\n", i, v.toString())
	//}
	//fmt.Printf("backwardOnLevel\n")
	//for  i , v := range node.backwardOnLevel {
	//	fmt.Printf("%d : %s\n", i, v.toString())
	//}
	if node != nil {
		return node.payload, node.forwardOnLevel[0], node.backwardOnLevel[0], nil
	} else {
		return nil, nil, nil, fmt.Errorf("node not exists for key : %s", skl.header.toString())
	}
}

func (skl *skiplist) saveNodes(needSaveNodes map[string]*skiplistNode) {
	for _, v := range needSaveNodes {
		skl.saveNode(v)
	}
}

func (skl *skiplist) updatePayload(key nodeKeyType, pl *nodePayload) error {
	node := skl.getNode(key)
	if node == nil {
		return fmt.Errorf("key not exists")
	} else {
		node.payload = pl
		return skl.saveNode(node)
	}
}