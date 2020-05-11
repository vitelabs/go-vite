package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/gops/agent"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/consensus"
	"github.com/vitelabs/go-vite/interval/node"
	"github.com/vitelabs/go-vite/interval/utils"
)

import (
	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	//log.InitPath()
	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal("%v", err)
	}
	defaultBoot := "localhost:8000"
	//boot := startBoot(defaultBoot)
	n := startNode(defaultBoot, 8081, "100")
	time.Sleep(time.Second)

	balance := n.Leger().GetAccountBalance("jie")
	if balance != 200 {
		return
	}

	balance = n.Leger().GetAccountBalance("viteshan")
	if balance != 200 {
		return
	}
	N := 4
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

	for i := 0; i < N; i++ {
		from := "jie" + strconv.Itoa(i)
		to := "jie" + strconv.Itoa(i-1)
		if i == 0 {
			to = "jie" + strconv.Itoa(N-1)
		}
		go func(from string, to string) {
			for {
				balance := n.Leger().GetAccountBalance(from)
				if balance > 1 {
					log.Info("from:%s, to:%s, %d", from, to, balance)
					for j := 0; j < balance-1; j++ {
						err := n.Leger().RequestAccountBlock(from, to, -1)
						if err != nil {
							log.Error("%v", err)
							//return
						}
					}

				}
				time.Sleep(time.Millisecond * 400)
			}

		}(from, to)
	}

	for i := 0; i < N; i++ {
		go func(addr string) {
			for {
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
				time.Sleep(time.Second)
			}

		}("jie" + strconv.Itoa(i))
	}

	i := make(chan struct{})
	<-i
	//time.Sleep(30 * time.Second)
	//boot.Stop()
}

func startNode(bootAddr string, port int, nodeId string) node.Node {
	cfg := &config.Node{
		P2pCfg:       &config.P2P{NodeId: nodeId, Port: port, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: &config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
	}
	n, err := node.NewNode(cfg)
	utils.Nil(err)
	n.Init()
	n.Start()
	return n
}
