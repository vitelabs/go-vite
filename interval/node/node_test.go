package node

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/gops/agent"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/consensus"
	"github.com/vitelabs/go-vite/interval/p2p"

	_ "net/http/pprof"
	"path"
)

func TestNode(t *testing.T) {
	bootAddr := "localhost:8000"
	startBoot(bootAddr)
	//N := 2
	//for j := 0; j < N; j++ {
	//	go func(i int) {
	//		cfg := config.Node{
	//			P2pCfg:       config.P2P{NodeId: strconv.Itoa(i), Port: 8080 + i, LinkBootAddr: bootAddr, NetId: 0},
	//			ConsensusCfg: config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
	//		}
	//		n := node{}
	//		n.Init(cfg)
	//		n.Start()
	//	}(j)
	//}
	cfg := &config.Node{
		P2pCfg:       &config.P2P{NodeId: strconv.Itoa(3), Port: 8080 + 3, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: &config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
		MinerCfg:     &config.Miner{Enabled: true, HexCoinbase: "vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8"},
		ChainCfg:     &config.Chain{StoreType: common.Memory},
		BaseCfg:      &config.Base{DataDir: "/Users/jie/go_vite/data"},
	}
	n, e := NewNode(cfg)
	if e != nil {
		panic(e)
	}
	n.Init()
	n.Start()

	time.Sleep(200 * time.Second)
	n.Stop()
}

func startBoot(bootAddr string) p2p.Boot {
	cfg := config.Boot{BootAddr: bootAddr}
	boot := p2p.NewBoot(cfg)
	boot.Start()
	return boot
}

func TestStruct(t *testing.T) {
	node := config.Node{}
	if node.MinerCfg.Enabled == true {
		t.Error("error except")
	}
}

func TestBootNode(t *testing.T) {
	bootAddr := "localhost:8000"
	startBoot(bootAddr)
	time.Sleep(200 * time.Second)
}

func TestNode_Start(t *testing.T) {
	bootAddr := "localhost:8000"
	i := 0
	cfg := &config.Node{
		P2pCfg:       &config.P2P{NodeId: strconv.Itoa(i), Port: 8080 + i, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: &config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
		ChainCfg:     &config.Chain{StoreType: common.Memory},
		MinerCfg:     &config.Miner{Enabled: false},
		BaseCfg:      &config.Base{DataDir: "/Users/jie/go_vite/data"},
	}
	n, e := NewNode(cfg)
	if e != nil {
		panic(e)
	}
	n.Init()
	n.Start()
	time.Sleep(200 * time.Second)
}

func TestSendReceived(t *testing.T) {
	defaultBoot := "localhost:8000"
	boot := startBoot(defaultBoot)
	n := startNode(defaultBoot, 8091, "1")
	time.Sleep(time.Second)
	balance := n.Leger().GetAccountBalance("jie")
	if balance != 200 {
		t.Error("balance is wrong.", balance, 200)
	}
	err := n.Leger().RequestAccountBlock("jie", "jie2", -20)
	if err != nil {
		t.Error("send tx error.", err)
	}
	balance = n.Leger().GetAccountBalance("jie")
	if balance != 180 {
		t.Error("balance is wrong.", balance, 180)
	}
	reqs := n.Leger().ListRequest("jie2")
	if len(reqs) != 1 {
		t.Error("reqs size is wrong.", reqs)
		return
	}
	req := reqs[0]
	err = n.Leger().ResponseAccountBlock("jie", "jie2", req.ReqHash)
	if err != nil {
		t.Error("response error.", err, req.ReqHash)
	}
	n.Stop()
	boot.Stop()
}

func TestMuilt(t *testing.T) {
	//  "vite_2ad661b3b5fa90af7703936ba36f8093ef4260aaaeb5f15cf8",
	//	"vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c3a",
	//	"vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8",
	//	"vite_85e8adb768aed85f2445eb1d71b933370d2980916baa3c1f3c",
	//	"vite_93dd41694edd756512da7c4af429f3e875c374a53bfd217e00",
	defaultBoot := "localhost:8000"
	//boot := startBoot(defaultBoot)
	n := startNode(defaultBoot, 8081, "1")
	n.Wallet().SetCoinBase("vite_2ad661b3b5fa90af7703936ba36f8093ef4260aaaeb5f15cf8")
	n.StartMiner()

	n = startNode(defaultBoot, 8082, "2")
	n.Wallet().SetCoinBase("vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c3a")
	n.StartMiner()

	n = startNode(defaultBoot, 8083, "3")
	n.Wallet().SetCoinBase("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	n.StartMiner()

	n = startNode(defaultBoot, 8084, "4")
	n.Wallet().SetCoinBase("vite_85e8adb768aed85f2445eb1d71b933370d2980916baa3c1f3c")
	n.StartMiner()

	n = startNode(defaultBoot, 8085, "5")
	n.Wallet().SetCoinBase("vite_93dd41694edd756512da7c4af429f3e875c374a53bfd217e00")
	n.StartMiner()

	i := make(chan struct{})
	<-i
}

func TestSend(t *testing.T) {
	defaultBoot := "localhost:8000"
	//boot := startBoot(defaultBoot)
	startNode(defaultBoot, 8095, "5")

	i := make(chan struct{})
	<-i
}

func TestBenchmark(t *testing.T) {
	//log.InitPath()
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal("%v", err)
	}
	//boot := startBoot(defaultBoot)
	n := startNodeFromDefaul()
	time.Sleep(time.Second)

	jieBalance := n.Leger().GetAccountBalance("jie")
	if jieBalance != 200 {
		t.Error("balance is wrong.", jieBalance, 200)
	}

	viteshanBalance := n.Leger().GetAccountBalance("viteshan")
	if viteshanBalance != 200 {
		t.Error("balance is wrong.", viteshanBalance, 200)
	}
	N := 4
	if jieBalance == 200 && viteshanBalance == 200 {
		for i := 0; i < N; i++ {
			addr := "jie" + strconv.Itoa(i)
			err := n.Leger().RequestAccountBlock("jie", addr, -30)
			if err != nil {
				log.Error("%v", err)
				return
			}
		}
		for i := 0; i < N; i++ {
			addr := "jie" + strconv.Itoa(i)
			err := n.Leger().RequestAccountBlock("viteshan", addr, -30)
			if err != nil {
				log.Error("%v", err)
				return
			}
		}
	}
	go func() {
		for {
			for i := 0; i < N; i++ {
				from := "jie" + strconv.Itoa(i)
				to := "jie" + strconv.Itoa(i-1)
				if i == 0 {
					to = "jie" + strconv.Itoa(N-1)
				}
				balance := n.Leger().GetAccountBalance(from)
				if balance > 1 {
					log.Info("from:%s, to:%s, %d", from, to, balance)
					for j := 0; j < balance-1; j++ {
						err := n.Leger().RequestAccountBlock(from, to, -1)
						if err != nil {
							log.Error("%v", err)
						}
					}

				}

			}
			time.Sleep(time.Millisecond * 50)
		}
	}()

	go func() {
		for {
			for i := 0; i < N; i++ {
				addr := "jie" + strconv.Itoa(i)
				reqs := n.Leger().ListRequest(addr)
				if len(reqs) > 0 {
					for _, r := range reqs {
						log.Info("%v", r)
						err := n.Leger().ResponseAccountBlock(r.From, addr, r.ReqHash)
						if err != nil {
							log.Error("%v", err)
							//return
						}
					}
				}

			}
		}
	}()

	<-make(chan struct{})

	//time.Sleep(30 * time.Second)
	//boot.Stop()
}

func startNode(bootAddr string, port int, nodeId string) Node {
	cfg := &config.Node{
		P2pCfg:       &config.P2P{NodeId: nodeId, Port: port, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: &config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
		ChainCfg:     &config.Chain{StoreType: common.Memory},
		MinerCfg:     &config.Miner{Enabled: false},
		BaseCfg:      &config.Base{DataDir: "/Users/jie/go_vite/data"},
	}
	n, e := NewNode(cfg)
	if e != nil {
		panic(e)
	}
	n.Init()
	n.Start()
	return n
}

func startNodeFromDefaul() Node {
	cfg := &config.Node{}
	cfg.Parse(path.Join(config.HomeDir, "/go_vite/etc/default.yaml"))
	n, e := NewNode(cfg)
	if e != nil {
		panic(e)
	}
	n.Init()
	n.Start()
	return n
}

func TestNode_Disk(t *testing.T) {
	disk := startNodeFromDefaultDisk()
	disk.Leger().Chain().HeadAccount("jie3")
	<-make(chan struct{})

}

func startNodeFromDefaultDisk() Node {
	cfg := &config.Node{}
	cfg.Parse(path.Join(config.HomeDir, "/go_vite/etc/config_disk_1.yaml"))
	n, e := NewNode(cfg)
	if e != nil {
		panic(e)
	}
	n.Init()
	n.Start()
	return n
}
