package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

const CountPerDelete = 10000

type RecoverNodeManager struct {
	ctx  *cli.Context
	node *node.Node

	chain chain.Chain
}

func NewRecoverNodeManager(ctx *cli.Context, maker NodeMaker) (*RecoverNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	return &RecoverNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *RecoverNodeManager) getDeleteToHeight() uint64 {
	deleteToHeight := uint64(0)
	if nodeManager.ctx.GlobalIsSet(utils.LedgerDeleteToHeight.Name) {
		deleteToHeight = nodeManager.ctx.GlobalUint64(utils.LedgerDeleteToHeight.Name)
	}
	return deleteToHeight
}

func (nodeManager *RecoverNodeManager) Start() error {

	// Start up the node

	node := nodeManager.node
	viteConfig := node.ViteConfig()

	dataDir := viteConfig.DataDir
	chainCfg := viteConfig.Chain
	genesisCfg := viteConfig.Genesis
	// set fork points
	fork.SetForkPoints(viteConfig.ForkPoints)

	c := chain.NewChain(dataDir, chainCfg, genesisCfg)

	nodeManager.chain = c

	if err := c.Init(); err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}

	deleteToHeight := nodeManager.getDeleteToHeight()

	if deleteToHeight <= 0 {
		err := errors.New("deleteToHeight is 0.\n")
		panic(err)
	}

	fmt.Printf("Latest snapshot block height is %d\n", c.GetLatestSnapshotBlock().Height)
	fmt.Printf("Delete target height is %d\n", deleteToHeight)

	fmt.Printf("Start deleting, don't shut down. View the deletion process through the log in %s\n", viteConfig.RunLogDir())
	if _, err := c.DeleteSnapshotBlocksToHeight(deleteToHeight); err != nil {
		return err
	}

	//	if tmpDeleteToHeight > CountPerDelete {
	//		tmpDeleteToHeight = tmpDeleteToHeight - CountPerDelete
	//	}
	//
	//	if tmpDeleteToHeight < deleteToHeight {
	//		tmpDeleteToHeight = deleteToHeight
	//	}
	//
	//	fmt.Printf("Deleting to %d...\n", tmpDeleteToHeight)
	//
	//	if _, _, err := c.DeleteSnapshotBlocksToHeight(tmpDeleteToHeight); err != nil {
	//		fmt.Printf("Delete to %d height failed. error is "+err.Error()+"\n", tmpDeleteToHeight)
	//		return err
	//	}
	//	fmt.Printf("Delete to %d successed!\n", tmpDeleteToHeight)
	//
	//}

	//if checkResult, checkErr := c.TrieGc().Check(); checkErr != nil {
	//	fmt.Printf("Check trie failed! error is %s\n", checkErr.Error())
	//} else if !checkResult {
	//	fmt.Printf("Rebuild data...\n")
	//	if err := c.TrieGc().Recover(); err != nil {
	//		fmt.Printf("Rebuild data failed! error is %s\n", err.Error())
	//	} else {
	//		fmt.Printf("Rebuild data successed!\n")
	//	}
	//}
	//
	//fmt.Printf("Latest snapshot block height is %d\n", c.GetLatestSnapshotBlock().Height)
	return nil
}

func (nodeManager *RecoverNodeManager) Stop() error {
	nodeManager.chain.Stop()
	return nil
}

func (nodeManager *RecoverNodeManager) Node() *node.Node {
	return nodeManager.node
}
