package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"math/big"
	"net/http"
	"os/user"
	"path"
	"strconv"

	"github.com/abiosoft/ishell"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/wallet"
)

import _ "net/http/pprof"

var log = log15.New("module", "cmd")

func main() {
	logFile()

	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	var coinBaseAddr string

	//configPath := flag.String("config", path.Join(config.HomeDir, "/naive_vite/etc/default.yaml"), "config path")
	//log.Info(*configPath)
	cfg := new(config.Config)
	flag.StringVar(&coinBaseAddr, "p", "", "")
	flag.Parse()
	//e := cfg.Parse(*configPath)
	//if e != nil {
	//	panic(e)
	//}
	//e = cfg.Check()
	//if e != nil {
	//	panic(e)
	//}
	//
	//log.InitPath()
	// create new shell.
	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()

	// display welcome info.
	shell.Println("go-vite Interactive Shell")

	w := wallet.New(nil)
	var coinbase *types.Address = nil
	if coinBaseAddr != "" {
		tmpAddr := unlock(w, coinBaseAddr)
		coinbase = &tmpAddr
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "config",
			Help: "show config info",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "info",
			Help: "show config info",
			Func: func(c *ishell.Context) {
				b, err := json.Marshal(cfg)
				if err != nil {
					c.Err(err)
				}

				var out bytes.Buffer
				err = json.Indent(&out, b, "", "\t")

				if err != nil {
					c.Err(err)
				}

				c.Println(out.String())
			},
		})
		shell.AddCmd(autoCmd)

	}

	// subcommands and custom autocomplete.
	//{
	//	var bootNode p2p.Boot
	//	autoCmd := &ishell.Cmd{
	//		Name: "boot",
	//		Help: "start or stop boot node.",
	//	}
	//	autoCmd.AddCmd(&ishell.Cmd{
	//		Name: "start",
	//		Help: "start boot node.",
	//		Func: func(c *ishell.Context) {
	//			if bootNode != nil {
	//				c.Println("boot has started.")
	//				return
	//			}
	//			if cfg.BootCfg == nil || !cfg.BootCfg.Enabled {
	//				c.Println("boot config must be set")
	//				return
	//			}
	//			bootNode = startBoot(cfg.BootCfg.BootAddr)
	//			c.Println("boot start for[" + cfg.BootCfg.BootAddr + "] successfully.")
	//		},
	//	})
	//
	//	autoCmd.AddCmd(&ishell.Cmd{
	//		Name: "stop",
	//		Help: "stop boot node.",
	//		Func: func(c *ishell.Context) {
	//			if bootNode == nil {
	//				c.Println("boot has stopped.")
	//				return
	//			}
	//			bootNode.Stop()
	//			bootNode = nil
	//			c.Println("boot stop successfully.")
	//		},
	//	})
	//
	//	autoCmd.AddCmd(&ishell.Cmd{
	//		Name: "list",
	//		Help: "list linked node info.",
	//		Func: func(c *ishell.Context) {
	//			if bootNode == nil {
	//				c.Println("boot should be started.")
	//				return
	//			}
	//			all := bootNode.All()
	//			c.Printf("-----boot links -----\n")
	//			c.Println("Id\tAddr")
	//
	//			for _, p := range all {
	//				c.Printf("%s\t%s\n", p.Id, p.Addr)
	//			}
	//		},
	//	})
	//
	//	shell.AddCmd(autoCmd)
	//}

	var node *vite.Vite
	var p2p *p2p.Server
	{
		autoCmd := &ishell.Cmd{
			Name: "node",
			Help: "start or stop node.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "start",
			Help: "start node.",
			Func: func(c *ishell.Context) {
				if node != nil {
					c.Println("node has started.")
					return
				}

				var err error
				node, p2p, err = startNode(w, coinbase)
				if err != nil {
					c.Err(err)
					return
				}
				c.Println("node start successfully.", err)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "stop",
			Help: "stop node.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node has stopped.")
					return
				}
				node.Stop()
				node = nil
				c.Println("node stop successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "p2p",
			Help: "p2p info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node has stopped.")
					return
				}
				peers := p2p.Peers()
				c.Println("self:", p2p.URL())
				for _, p := range peers {
					c.Println(p.Address, p.Name, p.ID)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "net",
			Help: "net info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node has stopped.")
					return
				}
				state := node.Net().Status()

				bt, _ := json.Marshal(state)
				c.Printf("net status: %s\n", string(bt))
				c.Printf("net status: %s\n", string(bt))

			},
		})

		//autoCmd.AddCmd(&ishell.Cmd{
		//	Name: "peers",
		//	Help: "list peers for node.",
		//	Func: func(c *ishell.Context) {
		//		if node == nil {
		//			c.Println("node must be started.")
		//			return
		//		}
		//		net := node.P2P()
		//		peers, _ := net.AllPeer()
		//		c.Printf("-----net peers -----\n")
		//		c.Println("Id\tRemote\tState")
		//
		//		for _, p := range peers {
		//			bt, _ := json.Marshal(p.GetState())
		//			c.Printf("%s\t%s\t%s\n", p.Id(), p.RemoteAddr(), string(bt))
		//		}
		//	},
		//})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "account",
			Help: "start or stop node.",
		}
		//[list,create,balance,send,receive]
		//autoCmd.AddCmd(&ishell.Cmd{
		//	Name: "list",
		//	Help: "list accounts.",
		//	Func: func(c *ishell.Context) {
		//		if node == nil {
		//			c.Println("node should be started.")
		//			return
		//		}
		//		node.Leger().ListRequest()
		//		node.StartMiner()
		//		c.Println("miner start successfully.")
		//	},
		//})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "self",
			Help: "self address info.",
			Func: func(c *ishell.Context) {
				if coinbase != nil {
					c.Println("coinbase address", coinbase.String())
				} else {
					c.Println("coinbase address is nil")
				}

			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "set",
			Help: "set address.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				address := ""
				if len(c.Args) == 1 {
					address = c.Args[0]
				} else {
					c.ShowPrompt(false)
					defer c.ShowPrompt(true)
					c.Print("Address: ")
					address = c.ReadLine()
				}
				if address == "" {
					c.Println("address is empty.")
					return
				}

				tmp, err := types.HexToAddress(address)
				if err != nil {
					c.Err(err)
				} else {
					coinbase = &tmp
				}
				c.Println("set address successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "import",
			Help: "import account.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("PriKey: ")
				prikey := c.ReadLine()

				if prikey == "" {
					c.Println("address is empty.")
					return
				}
				addr := unlockAddr(w, "123456", prikey)
				c.Println("unlock address successfully.", addr)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "unlockAll",
			Help: "unlock all account.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}

				addrList := unlockAll(w)
				c.Println("unlock address successfully.")
				for _, a := range addrList {
					c.Println(a.String())
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "balance",
			Help: "balance info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}

				if coinbase == nil {
					c.Println("please set current address.")
					return
				}
				balance := printBalance(node, *coinbase)
				c.Println("balance is ", balance.String())
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "quota",
			Help: "quota info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}

				if coinbase == nil {
					c.Println("please set current address.")
					return
				}
				quota := printQuota(node, *coinbase)
				c.Println("quota is ", quota.String())
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "pledge",
			Help: "pledge info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}

				if coinbase == nil {
					c.Println("please set current address.")
					return
				}
				pledge := printPledge(node, *coinbase)
				c.Println("pledge is ", pledge.String())
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "send",
			Help: "send tx.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				if coinbase == nil {
					c.Println("please set coinBase.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("to Address: ")
				toAddress := c.ReadLine()

				if toAddress == "" {
					c.Println("to address is empty.")
					return
				}
				to, _ := types.HexToAddress(toAddress)
				c.Print("to Amount: ")
				amount, _ := strconv.Atoi(c.ReadLine())

				err := transfer(node, *coinbase, to, big.NewInt(int64(amount)))
				if err != nil {
					c.Println("send tx fail.", err, coinbase.String(), to.String())
				} else {
					c.Println("send tx success.", coinbase.String(), to.String(), amount)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "getQuota",
			Help: "get quota",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				if coinbase == nil {
					c.Println("please set coinBase.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("to Address: ")
				to := contracts.AddressPledge

				c.Print("to Amount: ")
				amount, _ := strconv.Atoi(c.ReadLine())

				err := transferPledge(node, *coinbase, to)
				if err != nil {
					c.Println("send quota pledge fail.", err, coinbase.String(), to.String())
				} else {
					c.Println("send quota pledge success.", coinbase.String(), to.String(), amount)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "autoReceive",
			Help: "auto receive tx.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				addr := *coinbase
				if len(c.Args) == 1 {
					addr, _ = types.HexToAddress(c.Args[0])
				}
				err := api.NewPrivateOnroadApi(node.OnRoad()).StartAutoReceive(addr, nil)
				c.Println("auto receive successfully.", err)
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "ablock",
			Help: "get account block info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "list",
			Help: "list account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				addr := *coinbase
				if len(c.Args) == 1 {
					addr, _ = types.HexToAddress(c.Args[0])
				}
				c.Printf("-----address[%s] blocks-----\n", addr)
				c.Println("Height\tHash\tPrevHash")
				blocks, _ := node.Chain().GetAccountBlocksByAddress(&addr, 0, 1000, 1000)
				for _, b := range blocks {
					c.Printf("%s\t%s\t%s\n", strconv.FormatUint(b.Height, 10), b.Hash, b.PrevHash)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "head",
			Help: "head account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				addr := *coinbase
				if len(c.Args) == 1 {
					addr, _ = types.HexToAddress(c.Args[0])
				}

				head, _ := node.Chain().GetLatestAccountBlock(&addr)

				if head != nil {
					c.Printf("head info, height:%d, hash:%s, prev:%s\n", head.Height, head.Hash, head.PrevHash)
				} else {
					c.Println("head is empty.")
				}

			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "reqs",
			Help: "account requests.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				addr := *coinbase
				if len(c.Args) == 1 {
					addr, _ = types.HexToAddress(c.Args[0])
				}
				c.Printf("-----address[%s] request blocks-----\n", addr)
				c.Println("From\tAmount\tReqHash")
				blocks, _ := api.NewPrivateOnroadApi(node.OnRoad()).GetOnroadBlocksByAddress(addr, 0, 1000)
				for _, b := range blocks {
					c.Printf("%s\t%s\t%s\n", b.FromAddress.String(), b.Amount, b.FromBlockHash)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "detail",
			Help: "detail for account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				height := uint64(1)
				addr := *coinbase
				if len(c.Args) == 2 {
					addr, _ = types.HexToAddress(c.Args[0])
					height, _ = strconv.ParseUint(c.Args[1], 10, 64)
				}
				block, _ := node.Chain().GetAccountBlockByHeight(&addr, height)
				bytes, _ := json.Marshal(block)
				c.Printf("detail info, block:%s\n", string(bytes))
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "sblock",
			Help: "get snapshot block info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "list",
			Help: "list snapshot blocks.",
			Func: func(c *ishell.Context) {

				c.Printf("-----snapshot blocks-----\n")
				c.Println("Height\tHash\tPrevHash\tAccountLen\tTime")
				head := node.Chain().GetLatestSnapshotBlock()
				blocks, _ := node.Chain().GetSnapshotBlocksByHeight(1, head.Height, true, true)
				for _, b := range blocks {
					c.Printf("%d\t%s\t%s\t%d\t%s\t%s\n", b.Height, b.Hash, b.PrevHash, len(b.SnapshotContent), b.Timestamp.Format("2018-01-01 15:04:05"), b.Producer())
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "head",
			Help: "head snapshot block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				head := node.Chain().GetLatestSnapshotBlock()

				c.Printf("head info, height:%d, hash:%s, prev:%s\n", head.Height, head.Hash, head.PrevHash)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "detail",
			Help: "detail for snapshot block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				height := uint64(1)
				if len(c.Args) == 1 {
					height, _ = strconv.ParseUint(c.Args[0], 10, 64)
				}
				block, _ := node.Chain().GetSnapshotBlockByHeight(height)
				bytes, _ := json.Marshal(block)
				c.Printf("detail info, block:%s\n", string(bytes))
			},
		})

		shell.AddCmd(autoCmd)
	}

	//{
	//	autoCmd := &ishell.Cmd{
	//		Name: "consensus",
	//		Help: "get consensus info.",
	//	}
	//	autoCmd.AddCmd(&ishell.Cmd{
	//		Name: "list",
	//		Help: "list snapshot blocks.",
	//		Func: func(c *ishell.Context) {
	//
	//			events, e := node.Consensus().ReadByTime()
	//			c.Printf("-----snapshot blocks-----\n")
	//			c.Println("Height\tHash\tPrevHash\tAccountLen\tTime")
	//
	//			head := node.Chain().GetLatestSnapshotBlock()
	//			blocks, _ := node.Chain().GetSnapshotBlocksByHeight(1, head.Height, true, false)
	//			for _, b := range blocks {
	//				c.Printf("%d\t%s\t%s\t%d\t%s\t%s\n", b.Height, b.Hash, b.PrevHash, len(b.SnapshotContent), b.Timestamp.Format("2018-01-01 15:04:05"), b.Producer())
	//			}
	//		},
	//	})
	//
	//	shell.AddCmd(autoCmd)
	//}

	{
		autoCmd := &ishell.Cmd{
			Name: "pool",
			Help: "print pool info info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "sprint",
			Help: "print snapshot pool info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				info := node.Pool().Info(nil)
				c.Println(info)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "aprint",
			Help: "print account pool info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				if len(c.Args) == 1 {
					addr, _ := types.HexToAddress(c.Args[0])
					info := node.Pool().Info(&addr)
					c.Println(info)
				} else {
					info := node.Pool().Info(coinbase)
					c.Println(info)
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	//{
	//	autoCmd := &ishell.Cmd{
	//		Name: "monitor",
	//		Help: "print stat info.",
	//	}
	//	autoCmd.AddCmd(&ishell.Cmd{
	//		Name: "stat",
	//		Help: "print monitor stat info.",
	//		Func: func(c *ishell.Context) {
	//			stats := monitor.Stats()
	//			for _, v := range stats {
	//				if v.Cnt > 0 {
	//					c.Printf("%s-%s", v.Type, v.Name)
	//					c.Println()
	//					c.Printf("\t\t\t%f\t%f\t%d\t%d", v.CntMean, v.Avg, v.Cnt, v.Cap)
	//					c.Println()
	//				}
	//			}
	//		},
	//	})
	//
	//	shell.AddCmd(autoCmd)
	//}

	// run shell
	shell.Run()
}
func startNode(w *wallet.Manager, tmp *types.Address) (*vite.Vite, *p2p.Server, error) {
	p2pServer, err := p2p.New(&p2p.Config{
		BootNodes: []string{
			"vnode://6d72c01e467e5280acf1b63f87afd5b6dcf8a596d849ddfc9ca70aab08f10191@192.168.31.146:8483",
			"vnode://1ceabc6c2b751b352a6d719b4987f828bb1cf51baafa4efac38bc525ed61059d@192.168.31.190:8483",
			"vnode://8343b3f2bc4e8e521d460cadab3e9f1e61ba57529b3fb48c5c076845c92e75d2@192.168.31.193:8483",
			"vnode://48d589104db59e822f16f029c5f430efa70094311ba7ddd3c9d292678fee1732@192.168.31.45:8483",
		},
		DataDir: path.Join(common.DefaultDataDir(), "/p2p"),
	})

	coinbase := "vite_a60e507124c2059ccb61039870dac3b5219aca014abc3807d0"
	if tmp != nil {
		coinbase = tmp.String()
	}
	config := &config.Config{
		DataDir: common.DefaultDataDir(),
		Producer: &config.Producer{
			Producer: true,
			Coinbase: coinbase,
		},
		Vm: &config.Vm{IsVmTest: false},
		Net: &config.Net{
			Single: false,
		},
	}
	vite, err := vite.New(config, w)
	if err != nil {
		panic(err)
		return nil, nil, err
	}
	p2pServer.Protocols = append(p2pServer.Protocols, vite.Net().Protocols()...)

	err = vite.Init()
	if err != nil {
		panic(err)
		return nil, nil, err
	}
	err = vite.Start(p2pServer)
	if err != nil {
		panic(err)
		return nil, nil, err
	}
	err = p2pServer.Start()
	if err != nil {
		panic(err)
		return nil, nil, err
	}
	return vite, p2pServer, nil
}

//func checkArgs(args []string) (bool, string) {
//	if len(args) != 1 {
//		return false, ""
//	}
//	return true, args[0]
//}
//
//func startBoot(bootAddr string) p2p.Boot {
//	cfg := config.Boot{BootAddr: bootAddr}
//	boot := p2p.NewBoot(cfg)
//	boot.Start()
//	return boot
//}

/**

- boot[start,stop,list]
- node[start,stop,peers]
- miner[start,stop]


- account[list,create,balance,send,receive]
- ablock[list,head,reqs,detail]
- sblock[list,head,detail]
- pool[sprint,aprint]
- monitor[stat]
- profile[start]
*/

var password = "123456"

func transfer(vite *vite.Vite, from types.Address, to types.Address, amount *big.Int) error {
	parms := api.CreateTransferTxParms{
		SelfAddr:    from,
		ToAddr:      to,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      amount.String(),
		Data:        nil,
		Difficulty:  new(big.Int).SetUint64(0x000000000),
	}
	err := api.NewWalletApi(vite).CreateTxWithPassphrase(parms)
	if err != nil {
		return err
	}
	return nil
}
func transferPledge(vite *vite.Vite, from types.Address, to types.Address) error {
	difficulty := new(big.Int).SetUint64(pow.FullThreshold)
	if printQuota(vite, from).Sign() > 0 {
		difficulty = new(big.Int).SetUint64(0x00)
	}
	byt, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, from)
	parms := api.CreateTransferTxParms{
		SelfAddr:    from,
		ToAddr:      to,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)).String(),
		Data:        byt,
		Difficulty:  difficulty,
	}
	err := api.NewWalletApi(vite).CreateTxWithPassphrase(parms)
	if err != nil {
		return err
	}
	return nil
}

func unlockAddr(w *wallet.Manager, passwd string, priKey string) types.Address {
	w.KeystoreManager.ImportPriv(priKey, passwd)
	accountPrivKey, _ := ed25519.HexToPrivateKey(priKey)
	accountPubKey := accountPrivKey.PubByte()
	addr := types.PubkeyToAddress(accountPubKey)

	w.KeystoreManager.Lock(addr)
	w.KeystoreManager.Unlock(addr, passwd, 0)
	//wLog.Info("unlock address", "address", addr.String(), "r", err)
	return addr
}

func unlock(w *wallet.Manager, addr string) types.Address {
	addresses, _ := types.HexToAddress(addr)
	err := w.KeystoreManager.Unlock(addresses, password, 0)
	if err != nil {
		log.Error("unlock fail.", "err", err, "address", addresses)
	}
	return addresses
}

func unlockAll(w *wallet.Manager) []types.Address {
	results := w.KeystoreManager.Addresses()

	for _, r := range results {
		err := w.KeystoreManager.Unlock(r, password, 0)
		if err != nil {
			log.Error("unlock fail.", "err", err, "address", r.String())
		}
	}
	return results
}

func printBalance(vite *vite.Vite, addr types.Address) *big.Int {
	balance, _ := vite.Chain().GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
	return balance
}

func printQuota(vite *vite.Vite, addr types.Address) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeQuota(head.Hash, addr)
	return big.NewInt(0).SetUint64(amount)
}

func printPledge(vite *vite.Vite, addr types.Address) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	return amount
}

func logFile() {

	logLevel, err := log15.LvlFromString("dbug")
	if err != nil {
		logLevel = log15.LvlInfo
	}
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	dir := path.Join(usr.HomeDir, "viteisbest", "log", "tmpvite.log")
	log15.Root().SetHandler(
		log15.LvlFilterHandler(logLevel, log15.Must.FileHandler(dir, log15.TerminalFormat())),
	)
}
