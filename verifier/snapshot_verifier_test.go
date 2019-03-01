package verifier

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/chain/unittest"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/consensus"

	"github.com/vitelabs/go-vite/chain"
)

var innerChainInstance chain.Chain

func getChainInstance(path string) chain.Chain {
	if path == "" {
		path = "Documents/vite/src/github.com/vitelabs/aaaaaaaa/devdata"
	}
	if innerChainInstance == nil {
		innerChainInstance = chain_unittest.NewChainInstanceFromAbsPath(path, false)
		innerChainInstance.Start()
	}

	return innerChainInstance
}

func TestSnapshotBlockVerify(t *testing.T) {
	chainInstance := getChainInstance("")

	v := NewSnapshotVerifier(chainInstance, nil)
	head := chainInstance.GetLatestSnapshotBlock()
	t.Log(head)
	for i := uint64(1); i <= head.Height; i++ {
		block, e := chainInstance.GetSnapshotBlockByHeight(i)
		if e == nil {
			if v.VerifyNetSb(block) != nil {
				t.Log(fmt.Sprintf("%+v\n", block))
			}
		}
	}
}

func TestVerifyGenesis(t *testing.T) {
	c := getChainInstance("")
	block := c.GetGenesisSnapshotBlock()
	snapshotBlock, _ := c.GetSnapshotBlockByHeight(1)
	if block.Hash != snapshotBlock.Hash {
		t.Error("snapshot block error.", snapshotBlock, block)
	}
	t.Log(c.GetLatestSnapshotBlock())
}

func TestContractProducerVerify(t *testing.T) {
	c := getChainInstance("/Users/jie/Library/GVite/testdata")

	cs := consensus.NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)

	//v := NewAccountVerifier(c, cs)

	blocks, err := c.GetAllLatestAccountBlock()
	if err != nil {
		t.Fatal(err)
		return
	}
	for _, b := range blocks {
		u, e := c.AccountType(&b.AccountAddress)
		if e != nil {
			t.Fatal(e)
		}
		if u == ledger.AccountTypeContract {
			fmt.Println(fmt.Sprintf("verify account %s, max:%d", b.AccountAddress, b.Height))
			for i := uint64(1); i <= b.Height; i++ {

				block, err := c.GetAccountBlockByHeight(&b.AccountAddress, i)
				if err != nil {
					t.Fatal(e)
				}
				if block.IsReceiveBlock() {
					//fmt.Println(fmt.Sprintf("verify account %s, height:%d, time:%s", b.AccountAddress, i, block.Timestamp))
					result, err := cs.VerifyAccountProducer(block)

					if err != nil || !result {
						meta, r := c.GetAccountBlockMetaByHash(&block.Hash)
						if r != nil {
							panic(r)
						}

						fmt.Println(fmt.Sprintf("time:%s:account:%s:height:%d:snapshotHeight:%d:result:%t:error:%s",
							block.Timestamp, block.AccountAddress, block.Height, meta.SnapshotHeight, result, err))
					}
				}

				//producerLegality := v.VerifyProducerLegality(block, u)
				//if producerLegality != nil {
				//	fmt.Println(fmt.Sprintf("[B]time:%s:account:%s:height:%d:error:%s", block.Timestamp, block.AccountAddress, block.Height, producerLegality))
				//}
			}
		}
	}
}

func TestContractProducerVerify2(t *testing.T) {
	c := getChainInstance("/Users/jie/Library/GVite/testdata")

	cs := consensus.NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)
	addr, err := types.HexToAddress("vite_00000000000000000000000000000000000000056ad6d26692")
	if err != nil {
		t.Fatal(err)
	}
	block, err := c.GetAccountBlockByHeight(&addr, uint64(1))

	result, err := cs.VerifyAccountProducer(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(result)

	events, u, e := cs.ReadByTime(types.DELEGATE_GID, *block.Timestamp)
	if e != nil {
		t.Log(e)
	}
	fmt.Println(fmt.Sprintf("index is %d", u))

	fmt.Println("len ", len(events))
	for _, event := range events {
		fmt.Printf("producer:%s\n", event.Address)
	}
	fmt.Println("real producer", block.Producer())
}

func TestContractProducerVerify3(t *testing.T) {
	c := getChainInstance("/Users/jie/Library/GVite/testdata")
	h1 := types.HexToHashPanic("f3b9187d69e0749e28f9c8172fd6b7b468cbe89acb14f8038bbbb6a402d738ae")
	h2 := types.HexToHashPanic("d7251a9d1da157dcfd20a729fc368f1dfc85a55e4057df229fbb1bacb3405385")

	m1, err := c.GetAccountBlockMetaByHash(&h1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("meta1:%+v\n", m1)
	m2, err := c.GetAccountBlockMetaByHash(&h2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("meta2:%+v\n", m2)

	fmt.Println(c.GetLatestSnapshotBlock())

	height := uint64(7400000)

	addr, err := types.HexToAddress("vite_c12af03b4ce71622b81a1bbe015e063e296b7d31943199c6ec")
	if err != nil {
		panic(err)
	}

	for i := height; i < 7771449; i++ {
		block, err := c.GetSnapshotBlockByHeight(i)
		if err != nil {
			panic(err)
		}
		hashHeight, ok := block.SnapshotContent[addr]
		if ok {
			fmt.Println(hashHeight)
			if hashHeight.Hash == h1 {
				fmt.Println("find hash", hashHeight)
			}
		}
	}
}

func TestContractProducerVerify4(t *testing.T) {
	c := getChainInstance("/Users/jie/Library/GVite/testdata")
	h1 := types.HexToHashPanic("03b66d224a20c7fb3d407e39b1d624d928f9277f1904ff6f9c4b6bdfb64ac4e4")

	m1, err := c.GetAccountBlockMetaByHash(&h1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("meta1:%+v\n", m1)

	fmt.Println(c.GetAccountBlockByHash(&h1))
	fmt.Println(c.GetLatestSnapshotBlock())

}

//time:2018-11-08 12:13:24 +0800 CST:account:vite_00000000000000000000000000000000000000056ad6d26692:height:1:snapshotHeight:0:result:false:error:%!s(<nil>)
//time:2018-11-08 12:13:24 +0800 CST:account:vite_00000000000000000000000000000000000000042d7ef71894:height:1:snapshotHeight:2:result:false:error:%!s(<nil>)
//time:2018-11-08 12:13:24 +0800 CST:account:vite_0000000000000000000000000000000000000001c9e9f25417:height:1:snapshotHeight:2:result:false:error:%!s(<nil>)
//time:2019-02-09 10:59:41 +0800 CST:account:vite_c12af03b4ce71622b81a1bbe015e063e296b7d31943199c6ec:height:18588:snapshotHeight:0:result:false:error:%!s(<nil>)
//time:2019-02-09 14:14:20 +0800 CST:account:vite_c12af03b4ce71622b81a1bbe015e063e296b7d31943199c6ec:height:18596:snapshotHeight:0:result:false:error:%!s(<nil>)

/**


vite_c12af03b4ce71622b81a1bbe015e063e296b7d31943199c6ec  03b66d224a20c7fb3d407e39b1d624d928f9277f1904ff6f9c4b6bdfb64ac4e4  18564   8 22:04:53 CST 2019

referSnapshot: 62cebf2866da619e03d7e8525cc0ed115269ee165f1bba53fa67c119ea8aa224  height:7430548   2019-02-08T21:26:43+08:00


801d1a7ebde474b05f8ebee29dd273e447a618bc2c06d6554c64e21f05a9ab5b  7430549   2019-02-08T22:09:05+08:00

"vite_c12af03b4ce71622b81a1bbe015e063e296b7d31943199c6ec": {
	"height": 18565,
	"hash": "f96d2da78b779995653ff7ea3e295ee0d4c4dbb00507b362bdd695266dcf793b"
},
*/
