package vm_context_test

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"testing"
)

var mockSnapshotBlockHash, _ = types.HexToHash("b31fd8301199fc08016060d8a8cc33c6290293c45e66089a2662771f6700bab5")
var mockAccountBlockHash, _ = types.HexToHash("d8c1989526f34211775b2bcdfb3714a4abbe700d9e7b40c064766059a6c38e6c")

var mockHash1, _ = types.HexToHash("7605588ecb33349955fefb2943406b8bc4330b386a6f8697b323663710ec1199")
var mockHash2, _ = types.HexToHash("cb604040e33597e8310acb9b5ac99e65a156f18dde187c4785f7399e3353ccf9")

var mockTokenTypeId, _ = types.HexToTokenTypeId("tti_000000000000000000004cfd")

var mockAddress, _ = types.HexToAddress("vite_73728b44bda289359fcb298b0d07a6489757a84fb5cfc74527")
var mockGid = types.Gid{12, 32, 43, 54, 23, 12, 23, 12, 4, 5}

var innerChainInstance chain.Chain

func getChainInstance() chain.Chain {
	if innerChainInstance == nil {
		innerChainInstance = chain.NewChain(&config.Config{
			DataDir: common.DefaultDataDir(),
			//Chain: &config.Chain{
			//	KafkaProducers: []*config.KafkaProducer{{
			//		Topic:      "test",
			//		BrokerList: []string{"abc", "def"},
			//	}},
			//},
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}

	return innerChainInstance
}

func TestVmContext_AddBalance(t *testing.T) {
	chainInstance := getChainInstance()

	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(vmContext)
	vmContext.AddLog(&ledger.VmLog{
		Topics: []types.Hash{mockSnapshotBlockHash, mockAccountBlockHash},
		Data:   []byte("YesViteIsBest"),
	})
	vmContext.AddLog(&ledger.VmLog{
		Topics: []types.Hash{mockHash1, mockHash2},
		Data:   []byte("YesViteIsBest222"),
	})
	fmt.Println(vmContext.UnsavedCache().LogList()[0])
	fmt.Println(vmContext.UnsavedCache().LogList()[1])

	fmt.Println(vmContext.GetStorage(nil, []byte("123")))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))
	fmt.Println(string(vmContext.GetContractCode(nil)))

	vmContext.SetStorage([]byte("123"), []byte("456123asdf"))
	vmContext.SetStorage([]byte("a12cs3"), []byte("456123sssasdf"))
	fmt.Println(vmContext.UnsavedCache().Storage())
	fmt.Printf("%s\n", vmContext.GetStorage(nil, []byte("123")))
	fmt.Printf("%s\n", vmContext.GetStorage(nil, []byte("a12cs3")))

	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(10))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))

	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(12))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))

	vmContext.SubBalance(&mockTokenTypeId, big.NewInt(2))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))

	vmContext.SubBalance(&mockTokenTypeId, big.NewInt(9))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))

	vmContext.SetContractCode([]byte("HasdIamCodeasdfsadfdsajklfjadslkfj;"))
	fmt.Println(string(vmContext.GetContractCode(nil)))

	vmContext.SetContractCode([]byte("HasdIamCodeasdasdfasdfasdffsadfdsajklfjadslkfj;"))
	fmt.Println(string(vmContext.GetContractCode(nil)))

	vmContext.Reset()
	fmt.Println()
	fmt.Println(vmContext.GetStorage(nil, []byte("123")))
	fmt.Println(vmContext.GetStorage(nil, []byte("a12cs3")))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))
	fmt.Println(string(vmContext.GetContractCode(nil)))

	vmContext.SetContractCode([]byte("90823;123123231231"))
	fmt.Println(string(vmContext.GetContractCode(nil)))
	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(12))
	fmt.Println(vmContext.GetBalance(nil, &mockTokenTypeId))

	vmContext.SetContractGid(&mockGid, vmContext.Address())
	fmt.Printf("%+v\n", vmContext.UnsavedCache().ContractGidList()[0])

	vmContext.SetContractGid(&mockGid, vmContext.Address())
	fmt.Printf("%+v\n", vmContext.UnsavedCache().ContractGidList()[1])

	fmt.Printf("%+v\n", vmContext.UnsavedCache())
	//vmContext.AddBalance(&mockTokenTypeId, big.NewInt(10))
	//vmContext.AddBalance(&mockTokenTypeId, big.NewInt(11))
	//vmContext.AddBalance(&mockTokenTypeId, big.NewInt(12))
	//log.Println(vmContext.ActionList()[0].Params[0].(*types.TokenTypeId).String())
	//log.Println(vmContext.ActionList()[0].Params[1].(*big.Int).String())
	//
	//log.Println(vmContext.ActionList()[1].Params[0].(*types.TokenTypeId).String())
	//log.Println(vmContext.ActionList()[1].Params[1].(*big.Int).String())
	//
	//log.Println(vmContext.ActionList()[2].Params[0].(*types.TokenTypeId).String())
	//log.Println(vmContext.ActionList()[2].Params[1].(*big.Int).String())
}

func TestVmContextIterator(t *testing.T) {
	chainInstance := getChainInstance()
	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, &contracts.AddressConsensusGroup)
	if err != nil {
		t.Fatal(err)
	}
	//vmContext.SetStorage([]byte("123123123"), []byte("asdfsadfasdfasdf"))
	//vmContext.SetStorage([]byte("1231231254"), []byte("asdfsadfasdfasdf"))
	iterator := vmContext.NewStorageIterator(nil)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			return
		}
		fmt.Printf("%v : %v\n", hex.EncodeToString(key), hex.EncodeToString(value))
	}

}
