package nodemanager

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/vm_context"
	"gopkg.in/urfave/cli.v1"
	"math/big"
)

type ExportNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewExportNodeManager(ctx *cli.Context, maker NodeMaker) (*ExportNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	// single mode
	node.Config().Single = true
	node.ViteConfig().Net.Single = true

	// no miner
	node.Config().MinerEnabled = false
	node.ViteConfig().Producer.Producer = false

	// no ledger gc
	ledgerGc := false
	node.Config().LedgerGc = &ledgerGc
	node.ViteConfig().Chain.LedgerGc = ledgerGc

	return &ExportNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *ExportNodeManager) getSbHeight() uint64 {
	sbHeight := uint64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportSbHeightFlags.Name) {
		sbHeight = nodeManager.ctx.GlobalUint64(utils.ExportSbHeightFlags.Name)
	}
	return sbHeight
}

func (nodeManager *ExportNodeManager) Start() error {

	allAddress := make([]types.Address, 0)

	generalAddressMap := make(map[types.Address]struct{})
	contractAddressMap := make(map[types.Address]struct{})

	sumBalanceMap := make(map[types.Address]*big.Int)
	inAccountBalanceMap := make(map[types.Address]*big.Int)
	onroadBalanceMap := make(map[types.Address]*big.Int)

	// Start up the node
	node := nodeManager.node
	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}
	chainInstance := node.Vite().Chain()

	sbHeight := nodeManager.getSbHeight()
	if sbHeight <= 0 {
		return errors.New("`--sbHeight` must not be 0")
	}

	sb, err := chainInstance.GetSnapshotBlockByHeight(sbHeight)
	if err != nil {
		return errors.New(fmt.Sprintf("chainInstance.GetSnapshotBlockByHeight failed, height is %d, error is %s", sbHeight, err.Error()))
	}
	if sb == nil {
		return errors.New(fmt.Sprintf("Snapshot block is nil, height is %d", sbHeight))
	}
	sbStateTrie := chainInstance.GetStateTrie(&sb.StateHash)
	if sbStateTrie == nil || sbStateTrie.Root == nil {
		return errors.New(fmt.Sprintf("The state trie of snapshot block is nil, height is %d. "+
			"The trie may be garbage collected, please set `--sbHeight` value greater than %d or execute the command `gvite recover --trie` to recover all trie.", sbHeight, chainInstance.TrieGc().RetainMinHeight()))
	}

	fmt.Printf("The snapshot block: height is %d, hash is %s\n", sb.Height, sb.Hash)

	iter := sbStateTrie.NewIterator(nil)

	viteBalanceKey := vm_context.BalanceKey(&ledger.ViteTokenId)

	// query balance that already belongs to the account.
	fmt.Printf("Start query balance that already belongs to the account.\n")
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}

		addr, err := types.BytesToAddress(key)
		if err != nil {
			return errors.New(fmt.Sprintf("Convert key to address failed, error is " + err.Error()))
		}
		allAddress = append(allAddress, addr)

		accountType, err := chainInstance.AccountType(&addr)
		if err != nil {
			return errors.New("Get account type failed, error is " + err.Error())
		}

		switch accountType {
		case 2:
			generalAddressMap[addr] = struct{}{}
		case 3:
			contractAddressMap[addr] = struct{}{}
			continue
		default:
			return errors.New(fmt.Sprintf("Account type is %d, addr is %s", accountType, addr))

		}
		accountStateHash, err := types.BytesToHash(value)
		if err != nil {
			return errors.New(fmt.Sprintf("Convert value to accountStateHash failed, error is " + err.Error()))
		}

		accountStateTrie := chainInstance.GetStateTrie(&accountStateHash)
		if accountStateTrie == nil {
			return errors.New(fmt.Sprintf("The state trie of account is nil, addr is %s", addr.String()))
		}

		var balance = big.NewInt(0)
		if balanceBytes := accountStateTrie.GetValue(viteBalanceKey); balanceBytes != nil {
			balance.SetBytes(balanceBytes)
		}

		inAccountBalanceMap[addr] = balance
		sumBalanceMap[addr] = balance
	}

	fmt.Printf("Complete the balance that already belongs to the account query. "+
		"There are %d accounts, %d accounts is general account, %d accounts is contract account\n", len(allAddress), len(generalAddressList), len(contractAddressList))

	// query balance that is onroad.
	fmt.Printf("Start query balance that is onroad.\n")

	isGeneralAccount := func(addr types.Address) bool {
		_, ok := generalAddressMap[addr]
		return ok
	}

	isContractAccount := func(addr types.Address) bool {
		_, ok := contractAddressMap[addr]
		return ok
	}

	for _, addr := range allAddress {
		onroadBlocks, err := chainInstance.GetOnRoadBlocksBySendAccount(addr, sb.Hash)
		if err != nil {
			return errors.New(fmt.Sprintf("GetOnRoadBlocksBySendAccount failed, addr is %s, sb.height is %d, sb.hash is %s, error is %s",
				addr.String(), sb.Height, sb.Hash, err.Error()))
		}

		for _, onroadBlock := range onroadBlocks {
			fromAddress := onroadBlock.AccountAddress
			toAddress := onroadBlock.ToAddress

			if _, ok := onroadBalanceMap[toAddress]; !ok {
				onroadBalanceMap[toAddress] = big.NewInt(0)
			}

			if isGeneralAccount(toAddress) {
				// auto receive money
				onroadBalanceMap[toAddress].Add(onroadBalanceMap[toAddress], onroadBlock.Amount)
				sumBalanceMap[toAddress].Add(sumBalanceMap[toAddress], onroadBlock.Amount)
			} else if isContractAccount(toAddress) {
				// revert the money
				onroadBalanceMap[fromAddress].Add(onroadBalanceMap[fromAddress], onroadBlock.Amount)
				sumBalanceMap[fromAddress].Add(sumBalanceMap[fromAddress], onroadBlock.Amount)
			} else {
				return errors.New(fmt.Sprintf("To address is not existed, toAddress is %s, addr is %s, onroadBlock height is %d, onroadBlock hash is %s",
					toAddress, addr, onroadBlock.Height, onroadBlock.Hash))
			}

		}

	}
	fmt.Printf("Complete the balance that is onroad query.\n")

	//deleteToHeight := nodeManager.getDeleteToHeight()
	//
	//if deleteToHeight <= 0 {
	//	err := errors.New("deleteToHeight is 0.\n")
	//	panic(err)
	//}
	//
	//
	//fmt.Printf("Latest snapshot block height is %d\n", c.GetLatestSnapshotBlock().Height)
	//fmt.Printf("Delete target height is %d\n", deleteToHeight)
	//
	//tmpDeleteToHeight := c.GetLatestSnapshotBlock().Height + 1
	//
	//for tmpDeleteToHeight > deleteToHeight {
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
	//
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

func (nodeManager *ExportNodeManager) Stop() error {
	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *ExportNodeManager) Node() *node.Node {
	return nodeManager.node
}
