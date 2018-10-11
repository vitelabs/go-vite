package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"
)

type TestNode struct {
	do chan int
	// Channel to wait for termination notifications
	stop chan struct{}
	lock sync.RWMutex
}

type TestObject struct {
	Name *string
	age  int
}

type TestNodeConfig struct {
	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// template：["broker1,broker2,...|topic",""]
	KafkaProducers []string `json:"KafkaProducers"`

	// p2p
	NetSelect            string
	Identity             string   `json:"Identity"`
	privateKey           string   `json:"PrivateKey"`
	MaxPeers             uint     `json:"MaxPeers"`
	MaxPassivePeersRatio uint     `json:"MaxPassivePeersRatio"`
	MaxPendingPeers      uint     `json:"MaxPendingPeers"`
	bootNodes            []string `json:"BootNodes"`
	Port                 uint     `json:"Port"`
	NetID                uint     `json:"NetID"`

	//rpc
	RPCEnabled bool `json:"RPCEnabled"`
	IPCEnabled bool `json:"IPCEnabled"`
	WSEnabled  bool `json:"WSEnabled"`

	IPCPath          string   `json:"IPCPath"`
	HttpHost         string   `json:"HttpHost"`
	HttpPort         int      `json:"HttpPort"`
	HttpVirtualHosts []string `json:"HttpVirtualHosts"`
	WSHost           string   `json:"WSHost"`
	WSPort           int      `json:"WSPort"`
}

func NewTestNode() *TestNode {
	return &TestNode{
		stop: make(chan struct{}),
	}
}

func TestNodeConfigParse(t *testing.T) {

	testNodeConfig := TestNodeConfig{}
	file := "node_config.json"

	if jsonConf, err := ioutil.ReadFile(file); err == nil {
		err = json.Unmarshal(jsonConf, &testNodeConfig)
		if err != nil {
			log.Info("cannot unmarshal the config file content, will use the default config", "error", err)
		}
	} else {
		log.Info("cannot read the config file, will use the default config", "error", err)
	}

	log.Info(fmt.Sprintf("nodeConfig content: %v", testNodeConfig))

}

func TestSlice(t *testing.T) {
	testNodes := []TestObject{}
	//testNodes[0] = TestObject{
	//	Name: "yuanyulei",
	//	age:  1,
	//}
	log.Info(fmt.Sprintf("%v", testNodes))

	testNodes1 := make([]TestObject, 1)
	testNodes1[0] = TestObject{
		age: 1,
	}
	log.Info(fmt.Sprintf("%v", testNodes1))
}

func (t *TestNode) Start() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	log.Info("testNode start...")
	time.AfterFunc(time.Minute, func() {
		log.Info("testNode time do stop...")
		t.do <- 1
	})
	log.Info("testNode wait...")
	<-t.do
	return nil
}

func (t *TestNode) Stop() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	// unblock n.Wait
	defer close(t.do)
	defer close(t.stop)
	log.Info("testNode stop...")
	return nil
}

func TestChannel(t *testing.T) {

	node := NewTestNode()

	// Start the node
	if err := node.Start(); err != nil {
		log.Error(fmt.Sprintf("Error staring protocol node: %v", err))
	}

	// Listening event closes the node
	log.Info("Listening event closes the node...")

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)
		<-c
		log.Info("Got interrupt, shutting down...")
		go node.Stop()
		for i := 10; i > 0; i-- {
			<-c
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}

	}()

}

func TestNewApp(t *testing.T) {
	fmt.Println("sss")
}

func TestMergeFlags(t *testing.T) {

	// Network Settings
	OneFlag := cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}

	// Network Settings
	TwoFlag := cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}

	fmt.Println(fmt.Sprintf("merge flags: %v", len(MergeFlags([]cli.Flag{OneFlag, TwoFlag}, []cli.Flag{OneFlag, TwoFlag}))))
}

func MergeFlags(flagsSet ...[]cli.Flag) []cli.Flag {

	mergeFlags := []cli.Flag{}

	for _, flags := range flagsSet {

		mergeFlags = append(mergeFlags, flags...)
	}
	return mergeFlags
}
