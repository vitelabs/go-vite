package nodemanager

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"gopkg.in/urfave/cli.v1"
	"math/big"
)

type ExportNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

var digits = big.NewInt(1000000000000000000)

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

// TODO unknown to address
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
		"There are %d accounts, %d accounts is general account, %d accounts is contract account\n", len(allAddress), len(generalAddressMap), len(contractAddressMap))

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
		onroadBlocks, err := chainInstance.GetOnRoadBlocksBySendAccount(&addr, sb.Height)
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
				return errors.New(fmt.Sprintf("ToAddress is not existed, toAddress is %s, addr is %s, onroadBlock height is %d, onroadBlock hash is %s",
					toAddress, addr, onroadBlock.Height, onroadBlock.Hash))
			}

		}
	}
	fmt.Printf("Complete the balance that is onroad query.\n")

	nodeManager.PrintBalanceMap(sumBalanceMap)
	return nil
}

func (nodeManager *ExportNodeManager) PrintBalanceMap(balanceMap map[types.Address]*big.Int) {
	totalBalance := big.NewInt(0)
	for addr, balance := range balanceMap {
		humanBalance := big.NewInt(0)
		humanBalance.Div(balance, digits)
		fmt.Printf("%s: %s | %s vite \n", addr, balance, humanBalance)
		totalBalance = totalBalance.Add(totalBalance, balance)
	}
	humanBalance := big.NewInt(0)
	humanBalance.Div(totalBalance, digits)

	fmt.Printf("total: %s | %s vite \n", totalBalance, totalBalance)

}
func exportContractBalance(m map[types.Address]*big.Int, addr types.Address, balance *big.Int, trie *trie.Trie, c chain.Chain) (map[types.Address]*big.Int, error) {
	if addr == types.AddressRegister {
		return exportRegisterBalance(m, trie), nil
	} else if addr == types.AddressPledge {
		return exportPledgeBalance(m, trie), nil
	} else if addr == types.AddressMintage {
		return exportMintageBalance(m, trie), nil
	} else if addr == types.AddressVote {
		return m, nil
	} else {
		// for other contract, return to creator
		responseBlock, err := c.GetAccountBlockByHeight(&addr, 1)
		if err != nil {
			return m, err
		}
		requestBlock, err := c.GetAccountBlockByHash(&responseBlock.FromBlockHash)
		if err != nil {
			return m, err
		}
		updateBalance(m, requestBlock.AccountAddress, new(big.Int).Add(requestBlock.Fee, balance))
		return nil, err
	}
}

func exportRegisterBalance(m map[types.Address]*big.Int, trie *trie.Trie) map[types.Address]*big.Int {
	// for register contract, return to register pledge addr
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		if abi.IsRegisterKey(key) {
			registration := new(types.Registration)
			if err := abi.ABIRegister.UnpackVariable(registration, abi.VariableNameRegistration, value); err == nil && registration.Amount != nil && registration.Amount.Sign() > 0 {
				updateBalance(m, registration.PledgeAddr, registration.Amount)
			}
		}
	}
	return m
}

func exportPledgeBalance(m map[types.Address]*big.Int, trie *trie.Trie) map[types.Address]*big.Int {
	// for pledge contract, return to pledge addr
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		if abi.IsPledgeKey(key) {
			pledgeInfo := new(abi.PledgeInfo)
			if err := abi.ABIPledge.UnpackVariable(pledgeInfo, abi.VariableNamePledgeInfo, value); err == nil && pledgeInfo.Amount != nil && pledgeInfo.Amount.Sign() > 0 {
				updateBalance(m, abi.GetPledgeAddrFromPledgeKey(key), pledgeInfo.Amount)
			}
		}
	}
	return m
}

var mintageFee = new(big.Int).Mul(big.NewInt(1e3), big.NewInt(1e18))

func exportMintageBalance(m map[types.Address]*big.Int, trie *trie.Trie) map[types.Address]*big.Int {
	// for mintage contract, return 1000 vite to owner except for vite token
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		tokenId := abi.GetTokenIdFromMintageKey(key)
		if tokenId == ledger.ViteTokenId {
			continue
		}
		tokenInfo := new(types.TokenInfo)
		if err := abi.ABIMintage.UnpackVariable(tokenInfo, abi.VariableNameMintage, value); err == nil {
			updateBalance(m, tokenInfo.Owner, mintageFee)
		}
	}
	return m
}

func updateBalance(m map[types.Address]*big.Int, addr types.Address, balance *big.Int) map[types.Address]*big.Int {
	if v, ok := m[addr]; ok {
		v = v.Add(v, balance)
		m[addr] = v
	} else {
		m[addr] = balance
	}
	return m
}

func (nodeManager *ExportNodeManager) Stop() error {
	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *ExportNodeManager) Node() *node.Node {
	return nodeManager.node
}
