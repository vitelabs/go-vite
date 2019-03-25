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
	"github.com/vitelabs/go-vite/vm_context"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"sort"
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

func (nodeManager *ExportNodeManager) Start() error {

	allAddress := make(map[types.Address]struct{})
	generalAddressMap := make(map[types.Address]struct{})
	contractAddressMap := make(map[types.Address]struct{})

	inAccountBalanceMap := make(map[types.Address]*big.Int)
	inContractBalanceMap := make(map[types.Address]*big.Int)

	inAccountVCPBalanceMap := make(map[types.Address]*big.Int)
	contractRevertBalanceMap := make(map[types.Address]*big.Int)

	onroadBalanceMap := make(map[types.Address]*big.Int)
	onroadVCPBalanceMap := make(map[types.Address]*big.Int)

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

	vcpTokenId, err := types.HexToTokenTypeId("tti_251a3e67a41b5ea2373936c8")
	if err != nil {
		return errors.New("HexToTokenTypeId failed, error is " + err.Error())
	}
	vcpBalanceKey := vm_context.BalanceKey(&vcpTokenId)

	// query balance that already belongs to the account.
	fmt.Printf("Start query balance that already belongs to the account or needs to be refunded by the contract.\n")
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}

		addr, err := types.BytesToAddress(key)
		if err != nil {
			return errors.New(fmt.Sprintf("Convert key to address failed, error is " + err.Error()))
		}
		allAddress[addr] = struct{}{}

		accountType, err := chainInstance.AccountType(&addr)
		if err != nil {
			return errors.New("Get account type failed, error is " + err.Error())
		}

		var balance = big.NewInt(0)
		accountStateHash, err := types.BytesToHash(value)
		if err != nil {
			return errors.New(fmt.Sprintf("Convert value to accountStateHash failed, error is " + err.Error()))
		}

		accountStateTrie := chainInstance.GetStateTrie(&accountStateHash)
		if accountStateTrie == nil {
			return errors.New(fmt.Sprintf("The state trie of account is nil, addr is %s", addr.String()))
		}

		if balanceBytes := accountStateTrie.GetValue(viteBalanceKey); len(balanceBytes) > 0 {
			balance.SetBytes(balanceBytes)
		}

		switch accountType {
		case 2:
			generalAddressMap[addr] = struct{}{}

			inAccountBalanceMap[addr] = balance

			if vcpBalanceBytes := accountStateTrie.GetValue(vcpBalanceKey); len(vcpBalanceBytes) > 0 {
				vcpBalance := big.NewInt(0)
				vcpBalance.SetBytes(vcpBalanceBytes)
				inAccountVCPBalanceMap[addr] = vcpBalance
			}

		case 3:
			contractAddressMap[addr] = struct{}{}
			//balance *big.Int, trie *trie.Trie, c chain.Chain
			inContractBalanceMap[addr] = balance

			var err error
			contractRevertBalanceMap, err = exportContractBalance(contractRevertBalanceMap, addr, balance, accountStateTrie, chainInstance)
			if err != nil {
				return err
			}

		default:
			return errors.New(fmt.Sprintf("Account type is %d, addr is %s", accountType, addr))

		}

	}

	fmt.Printf("Complete calculating the balance that already belongs to the account or needs to be refunded by the contract. "+
		"There are %d accounts, %d accounts is general account, %d accounts is contract account\n", len(allAddress), len(generalAddressMap), len(contractAddressMap))

	// query balance that is onroad.
	fmt.Printf("Start query balance that is onroad.\n")

	getAccountType := func(addr types.Address) (uint64, error) {
		if _, ok := generalAddressMap[addr]; ok {
			return 2, nil
		}

		if _, ok := contractAddressMap[addr]; ok {
			return 3, nil
		}

		return chainInstance.AccountType(&addr)
	}

	inexistentAccountMap := make(map[types.Address]struct{})

	for addr := range allAddress {
		onroadBlocks, err := chainInstance.GetOnRoadBlocksBySendAccount(&addr, sb.Height)
		if err != nil {
			return errors.New(fmt.Sprintf("GetOnRoadBlocksBySendAccount failed, addr is %s, sb.height is %d, sb.hash is %s, error is %s",
				addr.String(), sb.Height, sb.Hash, err.Error()))
		}

		for _, onroadBlock := range onroadBlocks {
			if onroadBlock.TokenId != ledger.ViteTokenId &&
				onroadBlock.TokenId != vcpTokenId {

				continue
			}

			fromAddress := onroadBlock.AccountAddress
			toAddress := onroadBlock.ToAddress

			accountType, err := getAccountType(toAddress)
			if err != nil {
				return errors.New("getAccountType failed, error is " + err.Error())
			}
			switch accountType {
			case 1:
				fallthrough
			case 2:
				// auto receive money
				if onroadBlock.TokenId == ledger.ViteTokenId {
					if _, ok := onroadBalanceMap[toAddress]; !ok {
						onroadBalanceMap[toAddress] = big.NewInt(0)
					}

					onroadBalanceMap[toAddress].Add(onroadBalanceMap[toAddress], onroadBlock.Amount)
				} else {
					if _, ok := onroadVCPBalanceMap[toAddress]; !ok {
						onroadVCPBalanceMap[toAddress] = big.NewInt(0)
					}
					onroadVCPBalanceMap[toAddress].Add(onroadVCPBalanceMap[toAddress], onroadBlock.Amount)
				}

				if _, ok := allAddress[toAddress]; !ok {
					inexistentAccountMap[toAddress] = struct{}{}
					generalAddressMap[toAddress] = struct{}{}
				}
			case 3:
				if onroadBlock.TokenId != ledger.ViteTokenId {
					fmt.Printf("Error: The token id of contract onroad block is not vite token id, but is %s, hash is %s, addr is %s\n",
						onroadBlock.TokenId, onroadBlock.Hash, onroadBlock.AccountAddress)
					break
				}
				// revert the money
				if _, ok := onroadBalanceMap[fromAddress]; !ok {
					onroadBalanceMap[fromAddress] = big.NewInt(0)
				}

				onroadBalanceMap[fromAddress].Add(onroadBalanceMap[fromAddress], onroadBlock.Amount)
			default:
				return errors.New(fmt.Sprintf("ToAddress is not existed, toAddress is %s, addr is %s, onroadBlock height is %d, onroadBlock hash is %s",
					toAddress, addr, onroadBlock.Height, onroadBlock.Hash))
			}

		}
	}

	for addr := range inexistentAccountMap {
		allAddress[addr] = struct{}{}
	}
	fmt.Printf("Complete calculating the balance that is onroad.There are %d accounts, %d accounts is general account, %d accounts is contract account\n",
		len(allAddress), len(generalAddressMap), len(contractAddressMap))

	fmt.Println("======inAccount balance map======")
	nodeManager.printBalanceMap(inAccountBalanceMap, "vite")
	fmt.Println("======inAccount balance map======")

	fmt.Println("======inContract balance map======")
	nodeManager.printBalanceMap(inContractBalanceMap, "vite")
	fmt.Println("======inContract balance map======")

	fmt.Println("======inAccount vcp balance map======")
	nodeManager.printBalanceMap(inAccountVCPBalanceMap, "vcp")
	fmt.Println("======inAccount vcp balance map======")

	fmt.Println("======contractRevert balance map======")
	nodeManager.printBalanceMap(contractRevertBalanceMap, "vite")
	fmt.Println("======contractRevert balance map======")

	fmt.Println("======onroad balance map======")
	nodeManager.printBalanceMap(onroadBalanceMap, "vite")
	fmt.Println("======onroad balance map======")

	fmt.Println("======onroad vcp balance map======")
	nodeManager.printBalanceMap(onroadVCPBalanceMap, "vcp")
	fmt.Println("======onroad vcp balance map======")

	sumBalanceMap := nodeManager.calculateSumBalanceMap(inAccountBalanceMap, contractRevertBalanceMap, onroadBalanceMap)
	sumVCPBalanceMap := nodeManager.calculateSumBalanceMap(inAccountVCPBalanceMap, onroadVCPBalanceMap)
	fmt.Println("======sum balance map======")
	nodeManager.printBalanceMap(sumBalanceMap, "vite")
	fmt.Println("======sum balance map======")

	fmt.Println("======sum vcp balance map======")
	nodeManager.printBalanceMap(sumVCPBalanceMap, "vcp")
	fmt.Println("======sum vcp balance map======")

	return nil
}

func (nodeManager *ExportNodeManager) calculateSumBalanceMap(balanceMapList ...map[types.Address]*big.Int) map[types.Address]*big.Int {
	sumBalanceMap := make(map[types.Address]*big.Int)
	for _, balanceMap := range balanceMapList {
		for addr, balance := range balanceMap {
			if _, ok := sumBalanceMap[addr]; !ok {
				sumBalanceMap[addr] = big.NewInt(0)
			}
			sumBalanceMap[addr].Add(sumBalanceMap[addr], balance)
		}
	}

	return sumBalanceMap

}

type sortedBalanceItem struct {
	addr    types.Address
	balance *big.Int
}
type sortedBalanceList []*sortedBalanceItem

func (list sortedBalanceList) Len() int      { return len(list) }
func (list sortedBalanceList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list sortedBalanceList) Less(i, j int) bool {
	return list[i].balance.Cmp(list[j].balance) < 0
}
func getSortedBalanceList(balanceMap map[types.Address]*big.Int) sortedBalanceList {
	list := make(sortedBalanceList, 0, len(balanceMap))
	for addr, balance := range balanceMap {
		list = append(list, &sortedBalanceItem{
			addr:    addr,
			balance: balance,
		})
	}
	sort.Sort(sort.Reverse(list))
	return list
}

func (nodeManager *ExportNodeManager) printBalanceMap(balanceMap map[types.Address]*big.Int, unit string) {
	totalBalance := big.NewInt(0)
	balanceList := getSortedBalanceList(balanceMap)
	for _, item := range balanceList {
		if item.balance.Sign() > 0 {
			fmt.Printf("%s: %s %s\n", item.addr, item.balance, unit)
			totalBalance = totalBalance.Add(totalBalance, item.balance)
		}
	}

	fmt.Printf("total: %s %s\n", totalBalance, unit)

}
func exportContractBalance(m map[types.Address]*big.Int, addr types.Address, balance *big.Int, trie *trie.Trie, c chain.Chain) (map[types.Address]*big.Int, error) {
	if addr == types.AddressConsensusGroup {
		return exportRegisterBalance(m, trie), nil
	} else if addr == types.AddressPledge {
		return exportPledgeBalance(m, trie), nil
	} else if addr == types.AddressMintage {
		return exportMintageBalance(m, trie), nil
	} else if addr == types.AddressConsensusGroup {
		return m, nil
	} else if addr == types.AddressConsensusGroup {
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
		m = updateBalance(m, requestBlock.AccountAddress, new(big.Int).Add(requestBlock.Fee, balance))
		return m, err
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
				m = updateBalance(m, registration.PledgeAddr, registration.Amount)
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
				m = updateBalance(m, abi.GetPledgeAddrFromPledgeKey(key), pledgeInfo.Amount)
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
		if !abi.IsMintageKey(key) {
			continue
		}
		tokenId := abi.GetTokenIdFromMintageKey(key)
		if tokenId == ledger.ViteTokenId {
			continue
		}
		if tokenInfo, err := abi.ParseTokenInfo(value); err == nil {
			m = updateBalance(m, tokenInfo.PledgeAddr, mintageFee)
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
