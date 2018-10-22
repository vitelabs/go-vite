package model

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newOnroadBlocksPool() *OnroadBlocksPool {
	chain := chain.NewChain(&config.Config{
		Net:     nil,
		DataDir: common.GoViteTestDataDir(),
	})

	uAccess := NewUAccess()
	uAccess.Init(chain)

	chain.Init()
	chain.Start()

	fullCacheExpireTime = 5 * time.Second
	simpleCacheExpireTime = 7 * time.Second

	return NewOnroadBlocksPool(uAccess)
}

func TestOnroadBlocksPool_AcquireFullOnroadBlocksCache(t *testing.T) {
	addr, _, _ := types.CreateAddress()
	pool := newOnroadBlocksPool()
	for i := 0; i < 10; i++ {
		go func() {
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
		}()
		go func() {
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
		}()
	}

	time.Sleep(20 * time.Second)

}

var (
	genesisAccountPrivKeyStr string
	addr1, privKey1, _       = types.CreateAddress()
	addr1PrivKey, _          = ed25519.HexToPrivateKey(privKey1.Hex())
	addr1PubKey              = addr1PrivKey.PubByte()
	addr2, _, _              = types.CreateAddress()

	attovPerVite = big.NewInt(1e18)
	pledgeAmount = new(big.Int).Mul(big.NewInt(10), attovPerVite)

	defaultDifficulty = new(big.Int).SetUint64(pow.FullThreshold)
)

type VitePrepared struct {
	chain      chain.Chain
	onroadPool *OnroadBlocksPool
}

func init() {
	var isTest bool
	flag.BoolVar(&isTest, "vm.test", false, "test net gets unlimited balance and quota")
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")

	flag.Parse()
	vm.InitVmConfig(isTest)
}

func PrepareVite() *VitePrepared {
	dataDir := filepath.Join(common.HomeDir(), "testvite")
	fmt.Printf("\n----dataDir:%+v\n", dataDir)
	os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))
	c := chain.NewChain(&config.Config{DataDir: dataDir})

	c.Init()
	uAccess := NewUAccess()
	uAccess.Init(c)
	orPool := NewOnroadBlocksPool(uAccess)
	c.Start()

	return &VitePrepared{
		chain:      c,
		onroadPool: orPool,
	}
}

func TestOnroadBlocksPool_WriteAndRevertOnroad(t *testing.T) {
	vite := PrepareVite()
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	subLedger := make(map[types.Address][]*ledger.AccountBlock)
	caseTypeList := []byte{1, 3, 2, 0, 1, 1, 5, 0}
Loop:
	for _, caseType := range caseTypeList {
		var err error
		var blocks []*ledger.AccountBlock
		switch caseType {
		case 0:
			err = revertAllAbove(vite, subLedger)
		case 1:
			blocks, err = callTransfer(vite, &ledger.GenesisAccountAddress, &addr1, genesisAccountPrivKey, genesisAccountPubKey, defaultDifficulty)
		case 2:
			blocks, err = callTransfer(vite, &addr1, &addr2, addr1PrivKey, addr1PubKey, defaultDifficulty)
		case 3:
			blocks, err = receiveTransferSendBlocks(vite, &addr1)
		case 4:
			blocks, err = receiveTransferSendBlocks(vite, &addr2)
		case 5:
			//blocks, err = createContract(vite, &addr1, addr1PrivKey, addr1PubKey, defaultDifficulty)
		default:
			break Loop
		}
		if err != nil {
			t.Error(err)
			return
		}
		if caseType != 4 && len(blocks) > 0 {
			subLedger[blocks[0].AccountAddress] = append(subLedger[blocks[0].AccountAddress], blocks...)
		}
		if caseType == 4 {
			for k := range subLedger {
				delete(subLedger, k)
			}
		}
	}
}

func callTransfer(vite *VitePrepared, fromAddr, toAddr *types.Address,
	fromAddrPrivKey ed25519.PrivateKey, fromAddrPubKey []byte, difficulty *big.Int) ([]*ledger.AccountBlock, error) {

	var genBlockList []*ledger.AccountBlock
	im := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: *fromAddr,
		ToAddress:      toAddr,
		Amount:         big.NewInt(10),
		TokenId:        &ledger.ViteTokenId,
		Difficulty:     difficulty,
	}

	gen, err := generator.NewGenerator(vite.chain, nil, nil, &im.AccountAddress)
	if err != nil {
		return nil, err
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(fromAddrPrivKey, data), fromAddrPubKey, nil
	})
	if err != nil {
		return nil, err
	}
	if genResult.Err != nil {
		fmt.Printf("genResult err:%v\n", genResult.Err)
	}
	blockList := genResult.BlockGenList
	if len(blockList) > 0 {
		fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", blockList[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

		if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
			return nil, errors.New("InsertChain failed")
		}

		if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
			return nil, errors.New("WriteOnroad failed")
		}
		fmt.Printf("--addOnroad success\n")
		for _, v := range blockList {
			genBlockList = append(genBlockList, v.AccountBlock)
		}
		return genBlockList, nil
	}
	return nil, nil
}

func createContract(vite *VitePrepared, fromAddr *types.Address, fromAddrPrivKey ed25519.PrivateKey, fromAddrPubKey []byte, difficulty *big.Int) ([]*ledger.AccountBlock, error) {
	var genBlockList []*ledger.AccountBlock

	// send create
	data, _ := hex.DecodeString("00000000000000000002608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")

	im := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCreate,
		AccountAddress: *fromAddr,
		ToAddress:      nil,
		Amount:         big.NewInt(1e18),
		TokenId:        &ledger.ViteTokenId,
		Difficulty:     difficulty,
		Data:           data,
	}
	gen, err := generator.NewGenerator(vite.chain, nil, nil, &im.AccountAddress)
	if err != nil {
		return nil, err
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(fromAddrPrivKey, data), fromAddrPubKey, nil
	})
	if err != nil {
		return nil, err
	}
	if genResult.Err != nil {
		fmt.Printf("genResult err:%v\n", genResult.Err)
	}
	blockList := genResult.BlockGenList
	if len(blockList) > 0 {
		fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", blockList[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

		if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
			return nil, errors.New("InsertChain failed")
		}

		if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
			return nil, errors.New("WriteOnroad failed")
		}
		fmt.Printf("--addOnroad success\n")
		for _, v := range blockList {
			genBlockList = append(genBlockList, v.AccountBlock)
		}
		return genBlockList, nil
	}
	return nil, nil
}

func receiveTransferSendBlocks(vite *VitePrepared, addr *types.Address) ([]*ledger.AccountBlock, error) {
	var genBlockList []*ledger.AccountBlock

	sBlocks, err := vite.onroadPool.dbAccess.GetAllOnroadBlocks(*addr)
	if err != nil {
		return nil, err
	}
	if len(sBlocks) <= 0 {
		return nil, nil
	}
	for _, v := range sBlocks {
		genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
		genesisAccountPubKey := genesisAccountPrivKey.PubByte()

		gen, err := generator.NewGenerator(vite.chain, nil, nil, &v.ToAddress)
		if err != nil {
			return nil, err
		}
		genResult, err := gen.GenerateWithOnroad(*v, nil, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
		}, nil)
		if err != nil {
			return nil, err
		}
		if genResult.Err != nil {
			fmt.Printf("sendBlock.(Hash:%v, Height:%v), genResult err:%v\n", v.Hash, v.Height, genResult.Err)
		}
		blockList := genResult.BlockGenList
		if len(blockList) > 0 {
			fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", blockList[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

			if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
				return nil, errors.New("InsertChain failed")
			}

			if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
				return nil, errors.New("WriteOnroad failed")
			}
			fmt.Printf("--addOnroad success\n")
			for _, v := range blockList {
				genBlockList = append(genBlockList, v.AccountBlock)
			}
			return genBlockList, nil
		}
	}
	return nil, nil
}

func revertAllAbove(vite *VitePrepared, subLedger map[types.Address][]*ledger.AccountBlock) error {
	if err := vite.onroadPool.RevertOnroad(nil, subLedger); err != nil {
		return err
	}
	cutMap := excludeSubordinate(subLedger)

	revertSuccessCount := uint64(0)
	for _, blocks := range cutMap {
		// the blockList is sorted by height with ascending order
		for i := 0; i < len(blocks); i++ {
			v := blocks[i]

			if v.IsReceiveBlock() {
				if vite.chain.IsSuccessReceived(&v.AccountAddress, &v.FromBlockHash) {
					return errors.New("revert failed")
				}
				fmt.Printf("revert receiveBlock.(Hash:%v, Height:%v, Addr：%v, FromHash:%v)\n",
					v.Hash, v.Height, v.AccountAddress, v.FromBlockHash)
			} else {
				if !vite.chain.IsSuccessReceived(&v.ToAddress, &v.Hash) {
					return errors.New("revert failed")
				}
				if err := checkRevertSendCreateGidToAddress(vite, v); err != nil {
					return err
				}
				fmt.Printf("revert sendBlock.(Hash:%v, Height:%v, Addr：%v, ToAddr:%v)\n",
					v.Hash, v.Height, v.AccountAddress, v.ToAddress)
			}
			revertSuccessCount++
		}
	}
	fmt.Printf("--revert success, count:%v\n", revertSuccessCount)
	return nil
}

func checkRevertSendCreateGidToAddress(vite *VitePrepared, block *ledger.AccountBlock) error {
	if block.BlockType == ledger.BlockTypeSendCreate {
		gid := contracts.GetGidFromCreateContractData(block.Data)
		addrList, err := vite.onroadPool.dbAccess.GetContractAddrListByGid(&gid)
		if err != nil {
			return err
		}
		for _, addr := range addrList {
			if addr == block.ToAddress {
				return errors.New("revert failed")
			}
		}
	}
	return nil
}
