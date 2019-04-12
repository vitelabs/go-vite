package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const skiplistMaxLevel int8 = 18 // 2^18 262144
const metaStorageSalt = "listMeta:"
const skiplistIdLength = 21

type SkipListId [skiplistIdLength]byte // TradeToken[10] + QuoteToken[10] + side[1]

func (id *SkipListId) Bytes() []byte {
	return id[:]
}

func (id *SkipListId) SetBytes(data []byte) error {
	if length := len(data); length != skiplistIdLength {
		return fmt.Errorf("error SkipListId size error %v", length)
	}
	copy(id[:], data)
	return nil
}

type nodeKeyType interface {
	getStorageKey() []byte
	isNilKey() bool
	isBarrierKey() bool
	equals(nodeKeyType) bool
	toString() string
}

type nodePayload interface {
	compareTo(payload *nodePayload) int8
	randomSeed() int64
}

type nodePayloadProtocol interface {
	getNilKey() nodeKeyType
	getBarrierKey() nodeKeyType
	serialize(node *skiplistNode) ([]byte, error)
	deSerialize(nodeData []byte) (*skiplistNode, error)
	serializeMeta(meta *skiplistMeta) ([]byte, error)
	deSerializeMeta(nodeData []byte) (*skiplistMeta, error)
}

type BaseStorage interface {
	GetStorage(addr *types.Address, key []byte) []byte
	SetStorage(key []byte, value []byte)
	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash
}

type skiplistNode struct {
	nodeKey         nodeKeyType
	payload         *nodePayload
	forwardOnLevel  []nodeKeyType
	backwardOnLevel []nodeKeyType
}

type skiplist struct {
	listId          SkipListId
	header, tail    nodeKeyType
	length          int32
	level           int8
	storage         *BaseStorage
	protocol        *nodePayloadProtocol
	barrierNode     *skiplistNode
	contractAddress *types.Address
}

type skiplistMeta struct {
	header, tail   nodeKeyType
	length         int32
	level          int8
	forwardOnLevel []nodeKeyType
}

func randomLevel(seed int64) int8 {
	return int8(seed%int64(skiplistMaxLevel)) + 1
}

func (meta *skiplistMeta) fromList(list *skiplist) {
	meta.header = list.header
	meta.tail = list.tail
	meta.length = list.length
	meta.level = list.level
	meta.forwardOnLevel = make([]nodeKeyType, len(list.barrierNode.forwardOnLevel))
	copy(meta.forwardOnLevel, list.barrierNode.forwardOnLevel)
}

func (meta *skiplistMeta) getMetaStorageKey(listId SkipListId) []byte {
	return append([]byte(metaStorageSalt), listId.Bytes()...)
}

func newSkiplist(listId SkipListId, contractAddress *types.Address, storage *BaseStorage, protocol *nodePayloadProtocol) (*skiplist, error) {
	skl := &skiplist{}
	skl.listId = listId
	skl.header = (*protocol).getNilKey()
	skl.tail = (*protocol).getNilKey()
	skl.length = 0
	skl.level = 1
	skl.storage = storage
	skl.protocol = protocol
	skl.barrierNode = skl.createNode((*skl.protocol).getBarrierKey(), nil, skiplistMaxLevel)
	skl.contractAddress = contractAddress
	if err := skl.initMeta(listId); err != nil {
		return nil, err
	}
	return skl, nil
}

func (skl *skiplist) createNode(key nodeKeyType, payload *nodePayload, level int8) (*skiplistNode) {
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
	return &node
}

func (skl *skiplist) getNodeWithDirtyFilter(nodeKey nodeKeyType, dirtyNodes map[string]*skiplistNode) (*skiplistNode, error) {
	if nodeKey == nil || nodeKey.isNilKey() {
		return nil, fmt.Errorf("try get node for nil key")
	}
	if nodeKey.isBarrierKey() {
		return skl.barrierNode, nil
	}
	if len(dirtyNodes) > 0 {
		if node, ok := dirtyNodes[nodeKey.toString()]; ok {
			return node, nil
		}
	}
	nodeData := (*skl.storage).GetStorage(skl.contractAddress, nodeKey.getStorageKey())
	if len(nodeData) > 0 {
		return (*skl.protocol).deSerialize(nodeData)
	} else {
		return nil, nil
	}
}

//TODO whether delete node can be get from storage by key
func (skl *skiplist) getNode(nodeKey nodeKeyType) (*skiplistNode, error) {
	return skl.getNodeWithDirtyFilter(nodeKey, nil)
}

func (skl *skiplist) saveNode(node *skiplistNode) error {
	if node == nil || node.nodeKey.isNilKey() {
		return fmt.Errorf("node to save is invalid")
	}
	if node.isBarrier() {
		return nil
	}
	var (
		nodeData []byte
		err error
	)
	if nodeData, err = (*skl.protocol).serialize(node); err != nil {
		return err
	}
	(*skl.storage).SetStorage(node.nodeKey.getStorageKey(), nodeData)
	return nil
}

func (skl *skiplist) saveMeta() error {
	meta := &skiplistMeta{}
	meta.fromList(skl)
	var (
		metaData []byte
		err error
	)
	if metaData, err = (*skl.protocol).serializeMeta(meta); err != nil {
		return err
	}
	(*skl.storage).SetStorage(meta.getMetaStorageKey(skl.listId), metaData)
	return nil
}

func (skl *skiplist) initMeta(name SkipListId) error {
	var (
		meta = &skiplistMeta{}
		err error
	)
	metaData := (*skl.storage).GetStorage(skl.contractAddress, meta.getMetaStorageKey(name))
	if len(metaData) == 0 {
		return nil
	} else {
		if meta, err = (*skl.protocol).deSerializeMeta(metaData); err != nil {
			return err
		}
		skl.length = meta.length
		skl.header = meta.header
		skl.tail = meta.tail
		skl.level = meta.level
		skl.barrierNode.forwardOnLevel = make([]nodeKeyType, len(meta.forwardOnLevel))
		copy(skl.barrierNode.forwardOnLevel, meta.forwardOnLevel)
		//fmt.Printf("meta.length %d, meta.header %d, meta.tail %d, meta.level %d\n", meta.length, meta.header, meta.tail, meta.level)
		//for l, v := range skl.barrierNode.forwardOnLevel {
		//	fmt.Printf("meta.level %d, forward %s\n", l, v.toString())
		//}
		return nil
	}
}

func (nd *skiplistNode) isBarrier() bool {
	return nd.nodeKey.isBarrierKey()
}

func (skl *skiplist) insert(key nodeKeyType, payload *nodePayload) (err error) {
	//fmt.Printf("enter into insert for %s\n", key.toString())
	var (
		dirtyNodes  = make(map[string]*skiplistNode, skiplistMaxLevel)
		updateNodes = make([]*skiplistNode, skiplistMaxLevel)
		currentNode = skl.barrierNode
		forwardNode *skiplistNode
		i           int8
	)
	for i = skl.level - 1; i >= 0; i-- {
		// skl is desc list
		// find node where newNode will insert after this node
		for !currentNode.forwardOnLevel[i].isNilKey() {
			if forwardNode, err = skl.getNode(currentNode.forwardOnLevel[i]); err != nil {
				return err
			}
			if (*payload).compareTo(forwardNode.payload) <= 0 {
				currentNode = forwardNode
			} else {
				break
			}
		}
		updateNodes[i] = currentNode
	}

	level := randomLevel((*payload).randomSeed())
	newNode := skl.createNode(key, payload, level)
	if level > skl.level {
		for i = skl.level; i < level; i++ {
			updateNodes[i] = skl.barrierNode
		}
		skl.level = level
	}
	for i = 0; i < level; i++ {
		forwardKey := updateNodes[i].forwardOnLevel[i]
		newNode.forwardOnLevel[i] = forwardKey
		//fmt.Printf("level %d : set forward to %s for node %s\n", i, forwardKey.toString(), newNode.nodeKey.toString())
		if !forwardKey.isNilKey() {
			var forwardNode *skiplistNode
			var ok bool
			if forwardNode, ok = dirtyNodes[forwardKey.toString()]; !ok {
				if forwardNode, err = skl.getNodeWithDirtyFilter(updateNodes[i].forwardOnLevel[i], dirtyNodes); err != nil {
					return err
				}
			}
			forwardNode.backwardOnLevel[i] = key
			//fmt.Printf("level %d : set backward to %s for node %s\n", i, key.toString(), forwardNode.nodeKey.toString())
			dirtyNodes[forwardKey.toString()] = forwardNode
		}
		updateNodes[i].forwardOnLevel[i] = key
		//fmt.Printf("level %d : set forward to %s for node %s\n", i, key.toString(), updateNodes[i].nodeKey.toString())
		newNode.backwardOnLevel[i] = updateNodes[i].nodeKey
		//fmt.Printf("level %d : set backward to %s for node %s\n", i, updateNodes[i].nodeKey.toString(), newNode.nodeKey.toString())
		dirtyNodes[updateNodes[i].nodeKey.toString()] = updateNodes[i]
	}
	dirtyNodes[key.toString()] = newNode
	if newNode.forwardOnLevel[0].isNilKey() {
		skl.tail = key
	}
	if newNode.backwardOnLevel[0].isBarrierKey() {
		skl.header = key
	}
	skl.length++
	if err = skl.saveMeta(); err != nil {
		return err
	}
	if err = skl.saveNodes(dirtyNodes); err != nil {
		return err
	}
	//skl.traverse()
	return nil
}

func (skl *skiplist) delete(key nodeKeyType) (err error)  {
	var (
		dirtyNodes = make(map[string]*skiplistNode, skiplistMaxLevel)
		deleteNode, forwardNode, backwardNode *skiplistNode
	)
	if deleteNode, err = skl.getNode(key); err != nil {
		return err
	}
	if deleteNode == nil {
		return fmt.Errorf("node not exists for %s", key.toString())
	}
	var i int8
	for i = 0; int(i) < len(deleteNode.backwardOnLevel); i++ {
		// handle backwardNode
		backwardKey := deleteNode.backwardOnLevel[i]
		if backwardNode, err = skl.getNodeWithDirtyFilter(backwardKey, dirtyNodes); err != nil {
			return err
		}
		if backwardNode == nil {
			return fmt.Errorf("invalid backward node for %s", key.toString())
		}
		forwardKey := deleteNode.forwardOnLevel[i]
		backwardNode.forwardOnLevel[i] = forwardKey
		dirtyNodes[backwardKey.toString()] = backwardNode

		// handle forwardNode
		if !forwardKey.isNilKey() {
			if forwardNode, err = skl.getNodeWithDirtyFilter(forwardKey, dirtyNodes); err != nil {
				return err
			}
			forwardNode.backwardOnLevel[i] = backwardKey
			dirtyNodes[forwardKey.toString()] = forwardNode
		}

		if i == 0 {
			// delete the first node
			if deleteNode.backwardOnLevel[0].isBarrierKey() {
				skl.header = forwardKey
			}
			// delete last node
			if deleteNode.forwardOnLevel[0].isNilKey() {
				if backwardKey.isBarrierKey() {
					skl.tail = (*skl.protocol).getNilKey()
				} else {
					skl.tail = backwardKey
				}
			}
		}
	}
	skl.length--
	skl.adjustLevel()
	if err = skl.saveMeta(); err != nil {
		return err
	}
	if err = skl.saveNodes(dirtyNodes); err != nil {
		return err
	}
	//skl.traverse()
	return nil
}

// node of the key is the last to be deleted
// NOTE: key must be exits before truncate
func (skl *skiplist) truncateHeadTo(key nodeKeyType, length int32) (err error) {
	var (
		truncateNode, backwardNode, forwardNode *skiplistNode
	)
	if truncateNode, err = skl.getNode(key); err != nil {
		return err
	}
	if truncateNode == nil {
		return fmt.Errorf("node not exists for %s", key.toString())
	}
	splitNodes := make([]*skiplistNode, skiplistMaxLevel)
	var dirtyNodes = make(map[string]*skiplistNode, skiplistMaxLevel)
	truncateNodeLevel := 0
	for ; truncateNodeLevel < len(truncateNode.forwardOnLevel); truncateNodeLevel++ {
		splitNodes[truncateNodeLevel] = truncateNode
	}
	if truncateNodeLevel < int(skl.level) {
		if backwardNode, err = skl.getNode(truncateNode.backwardOnLevel[truncateNodeLevel-1]); err != nil {
			return err
		}
		if backwardNode == nil {
			return fmt.Errorf("node not exists for %s", truncateNode.backwardOnLevel[truncateNodeLevel-1])
		}
		// break in condition truncateNodeLevel == skl.level, len(splitNodes) == skl.level
		for ; truncateNodeLevel < int(skl.level); {
			// find node with higher level
			for ; !backwardNode.isBarrier() && len(backwardNode.backwardOnLevel) <= int(truncateNodeLevel); {
				if backwardNode, err = skl.getNode(backwardNode.backwardOnLevel[truncateNodeLevel-1]); err != nil {
					return err
				}
			}
			// break in condition barrier found, truncateNodeLevel will be equal to len(splitNodes)
			if backwardNode.isBarrier() {
				break
			}
			for ; truncateNodeLevel < int(skl.level) && truncateNodeLevel < len(backwardNode.backwardOnLevel); truncateNodeLevel++ {
				splitNodes[truncateNodeLevel] = backwardNode
			}
		}
	}
	for levelIndex := 0; levelIndex < truncateNodeLevel; levelIndex++ {
		forwardKey := splitNodes[levelIndex].forwardOnLevel[levelIndex]
		skl.barrierNode.forwardOnLevel[levelIndex] = forwardKey
		if levelIndex == 0 {
			skl.header = forwardKey
			if forwardKey.isNilKey() {
				skl.tail = forwardKey
			}
		}
		if !forwardKey.isNilKey() {
			if forwardNode, err = skl.getNodeWithDirtyFilter(forwardKey, dirtyNodes); err != nil {
				return err
			}
			forwardNode.backwardOnLevel[levelIndex] = skl.barrierNode.nodeKey
			dirtyNodes[forwardKey.toString()] = forwardNode
		}
	}
	skl.length -= length
	skl.adjustLevel()
	if err = skl.saveMeta(); err != nil {
		return err
	}
	if err := skl.saveNodes(dirtyNodes); err != nil {
		return err
	}
	//skl.traverse()
	return nil
}

func (skl *skiplist) peek() (pl *nodePayload, forwardKey nodeKeyType, backwardKey nodeKeyType, err error) {
	if skl.length == 0 {
		return nil, nil, nil, fmt.Errorf("skiplist is empty")
	}
	return skl.getByKey(skl.header)
}

func (skl *skiplist) getByKey(key nodeKeyType) (pl *nodePayload, forwardKey nodeKeyType, backwardKey nodeKeyType, err error) {
	if node, err := skl.getNode(key); err != nil {
		return nil, nil, nil, err
	} else if node != nil {
		//skl.showBackwardAndForward(node)
		return node.payload, node.forwardOnLevel[0], node.backwardOnLevel[0], nil
	} else {
		return nil, nil, nil, fmt.Errorf("node not exists for key : %s", skl.header.toString())
	}
}

func (skl *skiplist) saveNodes(needSaveNodes map[string]*skiplistNode) error {
	for _, v := range needSaveNodes {
		if err := skl.saveNode(v); err != nil {
			return err
		}
	}
	return nil
}

func (skl *skiplist) updatePayload(key nodeKeyType, pl *nodePayload) error {
	if node, err := skl.getNode(key); err != nil {
		return err
	} else if node == nil {
		return fmt.Errorf("key not exists")
	} else {
		node.payload = pl
		return skl.saveNode(node)
	}
}

func (skl *skiplist) adjustLevel() {
	if skl.length == 0 {
		skl.level = 1
	} else {
		var i int8 = 1
		for ; i <= skl.level; i++ {
			if !skl.barrierNode.forwardOnLevel[skl.level - i].isNilKey() {
				break
			}
		}
		skl.level = skl.level - i + 1
	}
}

func (skl *skiplist) showBackwardAndForward(node *skiplistNode) {
	fmt.Printf("node forward.len %d, backward.len %d\n", len(node.forwardOnLevel), len(node.backwardOnLevel))
	for i := 0; i < len(node.forwardOnLevel); i++ {
		fmt.Printf("level %d forward %s, backward %s\n", i, node.forwardOnLevel[i].toString(), node.backwardOnLevel[i].toString())
	}
}

func (skl *skiplist) traverse() {
	if skl.header.isNilKey() {
		fmt.Printf("skiplist to traverse is empty!!\n")
		return
	}
	if currentNode, err := skl.getNode(skl.header); err != nil {
		fmt.Printf("traverse failed get header node for key %s\n", skl.header.toString())
		return
	} else {
		for i := 0;; i++ {
			fmt.Printf("index %d, key is %s\n", i, currentNode.nodeKey.toString())
			if !currentNode.forwardOnLevel[0].isNilKey() {
				if currentNode, err = skl.getNode(currentNode.forwardOnLevel[0]); err != nil {
					fmt.Printf("failed get node for key %s\n", skl.header.toString())
					return
				}
			} else {
				return
			}
		}
	}
}
