package model

import (
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
	"sync"
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
	vm.InitVmConfig(isTest, false)
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
	caseTypeList := []byte{5, 0}
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
			blocks, err = createContract(vite, &ledger.GenesisAccountAddress, genesisAccountPrivKey, genesisAccountPubKey, defaultDifficulty)
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
	fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(vite.chain, nil)
	if err != nil {
		return nil, err
	}
	gen, err := generator.NewGenerator(vite.chain, fitestSnapshotBlockHash, nil, &im.AccountAddress)
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
		genBlock := blockList[0]
		fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", genBlock.VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

		if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
			return nil, errors.New("InsertChain failed")
		}

		if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
			return nil, errors.New("WriteOnroad failed")
		}

		fmt.Printf("success callTransfer.(BlockType:%v, Hash:%v, Height:%v, Addr：%v, ToAddr:%v)\n",
			genBlock.AccountBlock.BlockType, genBlock.AccountBlock.Hash, genBlock.AccountBlock.Height, genBlock.AccountBlock.AccountAddress, genBlock.AccountBlock.ToAddress)
		fmt.Printf("--addOnroad success\n")
		for _, v := range blockList {
			genBlockList = append(genBlockList, v.AccountBlock)
		}
		return genBlockList, nil
	}
	return nil, nil
}

func createContract(vite *VitePrepared, addr *types.Address, addrPrivKey ed25519.PrivateKey, addrPubKey []byte, difficulty *big.Int) ([]*ledger.AccountBlock, error) {
	var genBlockList []*ledger.AccountBlock

	// send create
	bytecode := []byte("PUSH1 0x80 PUSH1 0x40 MSTORE CALLVALUE DUP1 ISZERO PUSH2 0x10 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST POP PUSH2 0x2D7 DUP1 PUSH2 0x20 PUSH1 0x0 CODECOPY PUSH1 0x0 RETURN STOP PUSH1 0x80 PUSH1 0x40 MSTORE PUSH1 0x4 CALLDATASIZE LT PUSH2 0x4C JUMPI PUSH1 0x0 CALLDATALOAD PUSH29 0x100000000000000000000000000000000000000000000000000000000 SWAP1 DIV PUSH4 0xFFFFFFFF AND DUP1 PUSH4 0x954AB4B2 EQ PUSH2 0x51 JUMPI DUP1 PUSH4 0xA777D0DC EQ PUSH2 0xE1 JUMPI JUMPDEST PUSH1 0x0 DUP1 REVERT JUMPDEST CALLVALUE DUP1 ISZERO PUSH2 0x5D JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST POP PUSH2 0x66 PUSH2 0x14A JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP1 PUSH1 0x20 ADD DUP3 DUP2 SUB DUP3 MSTORE DUP4 DUP2 DUP2 MLOAD DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP DUP1 MLOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP1 DUP4 DUP4 PUSH1 0x0 JUMPDEST DUP4 DUP2 LT ISZERO PUSH2 0xA6 JUMPI DUP1 DUP3 ADD MLOAD DUP2 DUP5 ADD MSTORE PUSH1 0x20 DUP2 ADD SWAP1 POP PUSH2 0x8B JUMP JUMPDEST POP POP POP POP SWAP1 POP SWAP1 DUP2 ADD SWAP1 PUSH1 0x1F AND DUP1 ISZERO PUSH2 0xD3 JUMPI DUP1 DUP3 SUB DUP1 MLOAD PUSH1 0x1 DUP4 PUSH1 0x20 SUB PUSH2 0x100 EXP SUB NOT AND DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP JUMPDEST POP SWAP3 POP POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE DUP1 ISZERO PUSH2 0xED JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST POP PUSH2 0x148 PUSH1 0x4 DUP1 CALLDATASIZE SUB DUP2 ADD SWAP1 DUP1 DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP3 ADD DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP1 DUP1 PUSH1 0x1F ADD PUSH1 0x20 DUP1 SWAP2 DIV MUL PUSH1 0x20 ADD PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 SWAP4 SWAP3 SWAP2 SWAP1 DUP2 DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP4 DUP1 DUP3 DUP5 CALLDATACOPY DUP3 ADD SWAP2 POP POP POP POP POP POP SWAP2 SWAP3 SWAP2 SWAP3 SWAP1 POP POP POP PUSH2 0x1EC JUMP JUMPDEST STOP JUMPDEST PUSH1 0x60 PUSH1 0x0 DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP1 PUSH1 0x1F ADD PUSH1 0x20 DUP1 SWAP2 DIV MUL PUSH1 0x20 ADD PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 SWAP3 SWAP2 SWAP1 DUP2 DUP2 MSTORE PUSH1 0x20 ADD DUP3 DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP1 ISZERO PUSH2 0x1E2 JUMPI DUP1 PUSH1 0x1F LT PUSH2 0x1B7 JUMPI PUSH2 0x100 DUP1 DUP4 SLOAD DIV MUL DUP4 MSTORE SWAP2 PUSH1 0x20 ADD SWAP2 PUSH2 0x1E2 JUMP JUMPDEST DUP3 ADD SWAP2 SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 JUMPDEST DUP2 SLOAD DUP2 MSTORE SWAP1 PUSH1 0x1 ADD SWAP1 PUSH1 0x20 ADD DUP1 DUP4 GT PUSH2 0x1C5 JUMPI DUP3 SWAP1 SUB PUSH1 0x1F AND DUP3 ADD SWAP2 JUMPDEST POP POP POP POP POP SWAP1 POP SWAP1 JUMP JUMPDEST DUP1 PUSH1 0x0 SWAP1 DUP1 MLOAD SWAP1 PUSH1 0x20 ADD SWAP1 PUSH2 0x202 SWAP3 SWAP2 SWAP1 PUSH2 0x206 JUMP JUMPDEST POP POP JUMP JUMPDEST DUP3 DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 PUSH1 0x1F ADD PUSH1 0x20 SWAP1 DIV DUP2 ADD SWAP3 DUP3 PUSH1 0x1F LT PUSH2 0x247 JUMPI DUP1 MLOAD PUSH1 0xFF NOT AND DUP4 DUP1 ADD OR DUP6 SSTORE PUSH2 0x275 JUMP JUMPDEST DUP3 DUP1 ADD PUSH1 0x1 ADD DUP6 SSTORE DUP3 ISZERO PUSH2 0x275 JUMPI SWAP2 DUP3 ADD JUMPDEST DUP3 DUP2 GT ISZERO PUSH2 0x274 JUMPI DUP3 MLOAD DUP3 SSTORE SWAP2 PUSH1 0x20 ADD SWAP2 SWAP1 PUSH1 0x1 ADD SWAP1 PUSH2 0x259 JUMP JUMPDEST JUMPDEST POP SWAP1 POP PUSH2 0x282 SWAP2 SWAP1 PUSH2 0x286 JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST PUSH2 0x2A8 SWAP2 SWAP1 JUMPDEST DUP1 DUP3 GT ISZERO PUSH2 0x2A4 JUMPI PUSH1 0x0 DUP2 PUSH1 0x0 SWAP1 SSTORE POP PUSH1 0x1 ADD PUSH2 0x28C JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST SWAP1 JUMP STOP LOG1 PUSH6 0x627A7A723058 KECCAK256 0xeb 0xd6 PUSH20 0x5BDA3262137FFE14765109F0CA80F22F490B2D9E PUSH16 0x622A8D109B5A46850029000000000000")
	data := contracts.GetNewContractData(bytecode, types.DELEGATE_GID)

	latestSb := vite.chain.GetLatestSnapshotBlock()
	if latestSb == nil {
		return nil, errors.New("the latestSnapshotBlock can't be nil")
	}

	latestAb, err := vite.chain.GetLatestAccountBlock(addr)
	if err != nil {
		return nil, err
	}
	var preHash types.Hash
	height := uint64(0)
	if latestAb != nil {
		preHash = latestAb.Hash
		height = latestAb.Height
	}
	height++

	toAddress := contracts.NewContractAddress(*addr, height, preHash, latestSb.Hash)

	im := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCreate,
		AccountAddress: *addr,
		ToAddress:      &toAddress,
		Amount:         big.NewInt(1e18),
		TokenId:        &ledger.ViteTokenId,
		Difficulty:     difficulty,
		Data:           data,
	}
	fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(vite.chain, nil)
	if err != nil {
		return nil, err
	}
	gen, err := generator.NewGenerator(vite.chain, fitestSnapshotBlockHash, nil, &im.AccountAddress)
	if err != nil {
		return nil, err
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(addrPrivKey, data), addrPubKey, nil
	})
	if err != nil {
		return nil, err
	}
	if genResult.Err != nil {
		fmt.Printf("genResult err:%v\n", genResult.Err)
	}
	blockList := genResult.BlockGenList
	if len(blockList) > 0 {
		genBlock := blockList[0]
		fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", genBlock.VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

		if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
			return nil, errors.New("InsertChain failed")
		}

		if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
			return nil, errors.New("WriteOnroad failed")
		}

		gid := contracts.GetGidFromCreateContractData(genBlock.AccountBlock.Data)
		addrList, err := vite.onroadPool.dbAccess.GetContractAddrListByGid(&gid)
		if err != nil {
			return nil, err
		}
		for _, addr := range addrList {
			if addr == blockList[0].AccountBlock.ToAddress {
				fmt.Printf("success createContract.(BlockType:%v, Hash:%v, Height:%v, Addr：%v, ToAddr:%v), gid:%v\n",
					genBlock.AccountBlock.BlockType, genBlock.AccountBlock.Hash, genBlock.AccountBlock.Height, genBlock.AccountBlock.AccountAddress, genBlock.AccountBlock.ToAddress, gid)
			}
		}
		fmt.Printf("--addOnroad success\n")
		genBlockList = append(genBlockList, genBlock.AccountBlock)
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

		fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(vite.chain, nil)
		if err != nil {
			return nil, err
		}
		gen, err := generator.NewGenerator(vite.chain, fitestSnapshotBlockHash, nil, &v.ToAddress)
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
			genBlock := blockList[0]
			fmt.Printf("blocksList[0] balance:%+v,tokenId:%+v\n", genBlock.VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

			if err := vite.chain.InsertAccountBlocks(blockList); err != nil {
				return nil, errors.New("InsertChain failed")
			}

			if err := vite.onroadPool.WriteOnroad(nil, blockList); err != nil {
				return nil, errors.New("WriteOnroad failed")
			}
			fmt.Printf("success receiveTransfer.(BlockType:%v, Hash:%v, Height:%v, Addr：%v, ToAddr:%v)\n",
				genBlock.AccountBlock.BlockType, genBlock.AccountBlock.Hash, genBlock.AccountBlock.Height, genBlock.AccountBlock.AccountAddress, genBlock.AccountBlock.ToAddress)
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
				fmt.Printf("revert receiveBlock.(BlockType:%v, Hash:%v, Height:%v, Addr：%v, FromHash:%v)\n",
					v.BlockType, v.Hash, v.Height, v.AccountAddress, v.FromBlockHash)
			} else {
				if !vite.chain.IsSuccessReceived(&v.ToAddress, &v.Hash) {
					return errors.New("revert failed")
				}
				if v.BlockType == ledger.BlockTypeSendCreate {
					if err := checkRevertSendCreateGidToAddress(vite, v); err != nil {
						return err
					}
				}
				fmt.Printf("revert sendBlock.(BlockType:%v, Hash:%v, Height:%v, Addr：%v, ToAddr:%v)\n",
					v.BlockType, v.Hash, v.Height, v.AccountAddress, v.ToAddress)
			}
			revertSuccessCount++
		}
	}
	fmt.Printf("--revert success, count:%v\n", revertSuccessCount)
	return nil
}

func checkRevertSendCreateGidToAddress(vite *VitePrepared, block *ledger.AccountBlock) error {
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
	fmt.Printf("revert newAddressToGid.(gid:%v, toAddress:%v)\n", gid, block.ToAddress)
	return nil
}

func TestDeleteSyncMap(t *testing.T) {
	s := &sync.Map{}
	s.Store("1", 2)
	s.Store("2", 3)
	s.Range(func(key, value interface{}) bool {
		fmt.Println(key, value)
		return true
	})

	s.Range(func(key, value interface{}) bool {
		s.Delete(key)
		return true
	})

	fmt.Println("after delete")
	s.Range(func(key, value interface{}) bool {
		fmt.Println(key, value)
		return true
	})

	alarm := time.AfterFunc(5*time.Second, func() {
		fmt.Println("time alarm")
	})
	alarm.Stop()
	alarm.Stop()
	alarm1 := time.AfterFunc(5*time.Second, func() {
		fmt.Println("time alarm 1")
	})
	time.Sleep(10 * time.Second)
	alarm1.Stop()
	alarm1.Stop()

}