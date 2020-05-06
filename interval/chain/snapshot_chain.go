package chain

import (
	"time"

	"strconv"

	"encoding/json"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/store"
)

// snapshot block chain
type snapshotChain struct {
	head  *common.SnapshotBlock
	store store.BlockStore
}

func GetGenesisSnapshot() *common.SnapshotBlock {
	return genesisSnapshot
}

var genesisAccounts = []*common.AccountHashH{
	{common.HashHeight{Hash: "ccf131dac37a3ec9328290a9ad39c160baee02596daf303ad87d93815fce0a5a", Height: common.FirstHeight}, "viteshan"},
	{common.HashHeight{Hash: "904196e430c52d0687064a1723fa5124da7708e7e82d75924a846c4e84ac49c3", Height: common.FirstHeight}, "jie"},
}

var genesisSnapshot = common.NewSnapshotBlock(common.FirstHeight, "a601ad0af8123a9dd85a201273276a82e41d6cc1e708bd62ea432dea76038639", "", "viteshan", time.Unix(1533550878, 0), genesisAccounts)

func newSnapshotChain(store store.BlockStore) *snapshotChain {
	chain := &snapshotChain{}
	chain.store = store
	// init genesis block
	head := store.GetSnapshotHead()
	if head != nil {
		storeGenesis := store.GetSnapshotByHeight(genesisSnapshot.Height())
		if storeGenesis.Hash() != genesisSnapshot.Hash() {
			panic("error store snapshot hash. code:" + genesisSnapshot.Hash() + ", store:" + storeGenesis.Hash())
		} else {
			chain.head = chain.store.GetSnapshotByHeight(head.Height)
		}
	} else {
		chain.head = genesisSnapshot
		chain.store.PutSnapshot(genesisSnapshot)
		chain.store.SetSnapshotHead(&common.HashHeight{Hash: genesisSnapshot.Hash(), Height: genesisSnapshot.Height()})
	}
	return chain
}

func (self *snapshotChain) Head() *common.SnapshotBlock {
	return self.head
}

func (self *snapshotChain) GetBlockHeight(height uint64) *common.SnapshotBlock {
	if height < 0 {
		panic("height:" + strconv.FormatUint(height, 10))
		log.Error("can't request height 0 block.[snapshotChain]", height)
		return nil
	}
	block := self.store.GetSnapshotByHeight(height)
	return block
}

func (self *snapshotChain) GetBlockByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	if hashH.Height < 0 {
		log.Error("can't request height 0 block.[snapshotChain]", hashH.Height)
		return nil
	}
	head := self.head
	if hashH.Height == head.Height() && hashH.Hash == head.Hash() {
		return head
	}
	block := self.store.GetSnapshotByHeight(hashH.Height)
	if block != nil && hashH.Hash == block.Hash() {
		return block
	}
	return nil
}
func (self *snapshotChain) getBlockByHash(hash string) *common.SnapshotBlock {
	block := self.store.GetSnapshotByHash(hash)
	return block
}

func j(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

func (self *snapshotChain) insertChain(block *common.SnapshotBlock) error {
	log.Info("insert to snapshot Chain: %s", j(block))
	self.store.PutSnapshot(block)
	self.head = block
	self.store.SetSnapshotHead(&common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	return nil
}
func (self *snapshotChain) removeChain(block *common.SnapshotBlock) error {
	log.Info("remove from snapshot Chain: %s", block)

	head := self.store.GetSnapshotByHash(block.PreHash())
	self.store.DeleteSnapshot(common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	self.head = head
	if head == nil {
		self.store.SetSnapshotHead(nil)
	} else {
		self.store.SetSnapshotHead(&common.HashHeight{Hash: head.Hash(), Height: head.Height()})
	}

	return nil
}
