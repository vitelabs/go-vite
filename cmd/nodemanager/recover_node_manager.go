package nodemanager

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

const CountPerDelete = uint64(10000)

type RecoverNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewRecoverNodeManager(ctx *cli.Context, maker NodeMaker) (*RecoverNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	// single mode
	node.Config().Single = true
	node.ViteConfig().Net.Single = true

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
	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}

	deleteToHeight := nodeManager.getDeleteToHeight()

	if deleteToHeight <= 0 {
		err := errors.New("deleteToHeight is 0.\n")
		panic(err)
	}

	c := node.Vite().Chain()

	fmt.Printf("Latest snapshot block height is %d\n", c.GetLatestSnapshotBlock().Height)
	fmt.Printf("Delete target height is %d\n", deleteToHeight)

	tmpDeleteToHeight := c.GetLatestSnapshotBlock().Height + 1

	for tmpDeleteToHeight > deleteToHeight {
		if tmpDeleteToHeight > CountPerDelete {
			tmpDeleteToHeight = tmpDeleteToHeight - CountPerDelete
		}

		if tmpDeleteToHeight < deleteToHeight {
			tmpDeleteToHeight = deleteToHeight
		}

		fmt.Printf("Deleting to %d...\n", tmpDeleteToHeight)

		if _, _, err := c.DeleteSnapshotBlocksToHeight(tmpDeleteToHeight); err != nil {
			fmt.Printf("Delete to %d height failed. error is "+err.Error()+"\n", tmpDeleteToHeight)
			return err
		}
		fmt.Printf("Delete to %d successed!\n", tmpDeleteToHeight)

	}

	if checkResult, checkErr := c.TrieGc().Check(); checkErr != nil {
		fmt.Printf("Check trie failed! error is %s\n", checkErr.Error())
	} else if !checkResult {
		fmt.Printf("Rebuild data...\n")
		if err := c.TrieGc().Recover(); err != nil {
			fmt.Printf("Rebuild data failed! error is %s\n", err.Error())
		} else {
			fmt.Printf("Rebuild data successed!\n")
		}
	}

	fmt.Printf("Latest snapshot block height is %d\n", c.GetLatestSnapshotBlock().Height)
	return nil
}

func (nodeManager *RecoverNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *RecoverNodeManager) Node() *node.Node {
	return nodeManager.node
}
