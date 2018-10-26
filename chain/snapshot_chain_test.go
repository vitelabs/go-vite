package chain

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

func TestGetNeedSnapshotContent(t *testing.T) {
	chainInstance := getChainInstance()
	content := chainInstance.GetNeedSnapshotContent()
	for addr, item := range content {
		fmt.Printf("%s: %+v\n", addr.String(), item)
	}
}

func TestInsertSnapshotBlock(t *testing.T) {
	chainInstance := getChainInstance()

	makeBlocks(chainInstance, 10000)

}

func TestGetSnapshotBlocksByHash(t *testing.T) {
	chainInstance := getChainInstance()
	//blocks, err := chainInstance.GetSnapshotBlocksByHash(nil, 400, false, false)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//for _, block := range blocks {
	//	fmt.Printf("%d | %s | %s  | %d\n", block.Height, block.Timestamp, block.Producer(), block.Timestamp.UnixNano())
	//}
	//fmt.Println()
	fmt.Println(chainInstance.GetLatestSnapshotBlock())
	blocks2, err2 := chainInstance.GetSnapshotBlocksByHash(nil, 1, true, true)
	if err2 != nil {
		t.Fatal(err2)
	}
	for index, block := range blocks2 {
		fmt.Printf("%d: %+v\n", index, block)
	}

	//blocks3, err3 := chainInstance.GetSnapshotBlocksByHash(nil, 100, false, true)
	//if err3 != nil {
	//	t.Fatal(err3)
	//}
	//for index, block := range blocks3 {
	//	fmt.Printf("%d: %+v\n", index, block)
	//}
}

func TestGetSnapshotBlocksByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	blocks, err := chainInstance.GetSnapshotBlocksByHeight(2, 10, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetSnapshotBlockByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetSnapshotBlockByHeight(902)
	if err != nil {
		t.Fatal(err)
	}

	//for i := 0; i < 10000; i++ {
	//	fmt.Println(block.ComputeHash())
	//}
	fmt.Printf("%+v\n", block)

	block2, err2 := chainInstance.GetSnapshotBlockByHeight(2)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)
}

func TestGetSnapshotBlockByHash(t *testing.T) {
	chainInstance := getChainInstance()

	hash, _ := types.HexToHash("a43978e9e7c63cdae2e1b49c29ae724736a86aaa35e04b7a0a463fe7daa39502")
	block, err := chainInstance.GetSnapshotBlockByHash(&hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	for addr, hashHeight := range block.SnapshotContent {
		fmt.Printf("%s: %d %s\n", addr, hashHeight.Height, hashHeight.Hash.String())
	}
	//hash2, _ := types.HexToHash("f34e00c283f11728e28ccf2cf2138a7976b9ed7daaf7dbcc2ca598f66139f80d")
	//block2, err2 := chainInstance.GetSnapshotBlockByHash(&hash2)
	//if err2 != nil {
	//	t.Fatal(err2)
	//}
	//fmt.Printf("%+v\n", block2)
}

func TestGetLatestSnapshotBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block := chainInstance.GetLatestSnapshotBlock()
	newSb, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(newSb)
	fmt.Printf("%+v\n", block)
	newSb2, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(newSb2)
	fmt.Printf("%+v\n", block)
	newSb3, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(newSb3)
	fmt.Printf("%+v\n", block)
	newSb4, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(newSb4)
	fmt.Printf("%+v\n", block)
	newSb5, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(newSb5)
	fmt.Printf("%+v\n", block)
}

func TestGetGenesisSnapshotBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block := chainInstance.GetGenesisSnapshotBlock()
	fmt.Printf("%+v\n", block)
}

func TestGetConfirmBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetConfirmBlock(&GenesisMintageSendBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	hash, _ := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	block2, err2 := chainInstance.GetConfirmBlock(&hash)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)

	block3, err3 := chainInstance.GetConfirmBlock(&GenesisMintageBlock.Hash)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", block3)
}

func TestGetConfirmTimes(t *testing.T) {
	chainInstance := getChainInstance()
	times1, err := chainInstance.GetConfirmTimes(&GenesisMintageSendBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", times1)

	hash, _ := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	times2, err2 := chainInstance.GetConfirmTimes(&hash)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", times2)

	times3, err3 := chainInstance.GetConfirmTimes(&GenesisMintageBlock.Hash)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", times3)
}

// TODO
func TestGetSnapshotBlockBeforeTime(t *testing.T) {
	chainInstance := getChainInstance()
	//time1 := time.Now()
	//block, err := chainInstance.GetSnapshotBlockBeforeTime(&time1)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Printf("%+v\n", block)

	latestBlock := chainInstance.GetLatestSnapshotBlock()

	now := latestBlock.Timestamp.Add(time.Duration(10) * time.Second)
	count := 100
	createdBlockLen := 0
	for i := 1; i <= count; i++ {
		// not insert snapshotblock
		if i%20 == 0 {
			continue
		}
		newSb, _ := newSnapshotBlock()
		ts := now.Add(time.Duration(i) * time.Second)
		newSb.Timestamp = &ts
		chainInstance.InsertSnapshotBlock(newSb)
		createdBlockLen++
	}

	offset := 0
	nocreate := 0
	latestBlock = chainInstance.GetLatestSnapshotBlock()
	t.Logf("latestBlockHeight is %d", latestBlock.Height)
	go func() {
		for i := 0; i < 1000; i++ {
			newSb, _ := newSnapshotBlock()
			ts := now.Add(time.Duration(i+100) * time.Second)
			newSb.Timestamp = &ts

			chainInstance.InsertSnapshotBlock(newSb)
		}

	}()
	for i := 1; i <= count; i++ {

		offset = createdBlockLen - (i - 1 - nocreate)
		ts := now.Add(time.Duration(i) * time.Second)
		block, err2 := chainInstance.GetSnapshotBlockBeforeTime(&ts)
		if err2 != nil {
			t.Fatal(err2)
		}
		leftValue := block.Height
		rightValue := latestBlock.Height - uint64(offset)
		if leftValue != rightValue {
			t.Errorf("%d error!! %d %d %d", i, leftValue, rightValue, offset)

			t.Logf("%s %+v\n", ts, block)
		} else {
			t.Logf("right: %d", block.Height)
		}

		if i%20 == 0 {
			nocreate++
		}
		time.Sleep(time.Millisecond)
	}

	//time3 := GenesisSnapshotBlock.Timestamp.Add(time.Second * 100)
	//block3, err3 := chainInstance.GetSnapshotBlockBeforeTime(&time3)
	//if err3 != nil {
	//	t.Fatal(err3)
	//}
	//fmt.Printf("%+v\n", block3)
}

func TestGetConfirmAccountBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height, &GenesisMintageSendBlock.AccountAddress)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	block2, err2 := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height+3, &GenesisRegisterBlock.AccountAddress)
	if err2 != nil {
		t.Fatal(err2)
	}

	fmt.Printf("%+v\n", block2)

	block3, err3 := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height+10, &GenesisMintageSendBlock.AccountAddress)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", block3)

	block4, err4 := chainInstance.GetConfirmAccountBlock(0, &GenesisMintageSendBlock.AccountAddress)
	if err4 != nil {
		t.Fatal(err4)
	}
	fmt.Printf("%+v\n", block4)
}

func randomSendViteBlock(snapshotBlockHash types.Hash, addr1 *types.Address, addr2 *types.Address) ([]*vm_context.VmAccountBlock, []types.Address, error) {
	chainInstance := getChainInstance()
	now := time.Now()

	if addr1 == nil {
		accountAddress, _, _ := types.CreateAddress()
		addr1 = &accountAddress
	}
	if addr2 == nil {
		accountAddress, _, _ := types.CreateAddress()
		addr2 = &accountAddress
	}

	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, addr1)
	if err != nil {
		return nil, nil, err
	}
	latestBlock, _ := chainInstance.GetLatestAccountBlock(addr1)
	nextHeight := uint64(1)
	var prevHash types.Hash
	if latestBlock != nil {
		nextHeight = latestBlock.Height + 1
		prevHash = latestBlock.Hash
	}

	sendAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9))
	var sendBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: *addr1,
		ToAddress:      *addr2,
		Amount:         sendAmount,
		TokenId:        ledger.ViteTokenId,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		PublicKey:      []byte("public key"),
		SnapshotHash:   snapshotBlockHash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&ledger.ViteTokenId, sendAmount)

	sendBlock.StateHash = *vmContext.GetStorageHash()
	sendBlock.Hash = sendBlock.ComputeHash()
	return []*vm_context.VmAccountBlock{{
		AccountBlock: sendBlock,
		VmContext:    vmContext,
	}}, []types.Address{*addr1, *addr2}, nil
}

func newReceiveBlock(snapshotBlockHash types.Hash, accountAddress types.Address, fromHash types.Hash) ([]*vm_context.VmAccountBlock, error) {
	chainInstance := getChainInstance()
	latestBlock, _ := chainInstance.GetLatestAccountBlock(&accountAddress)
	nextHeight := uint64(1)
	var prevHash types.Hash
	if latestBlock != nil {
		nextHeight = latestBlock.Height + 1
		prevHash = latestBlock.Hash
	}

	now := time.Now()

	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, &accountAddress)
	if err != nil {
		return nil, err
	}

	var receiveBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: accountAddress,
		FromBlockHash:  fromHash,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		SnapshotHash:   snapshotBlockHash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&ledger.ViteTokenId, big.NewInt(100))

	receiveBlock.StateHash = *vmContext.GetStorageHash()
	receiveBlock.Hash = receiveBlock.ComputeHash()

	return []*vm_context.VmAccountBlock{{
		AccountBlock: receiveBlock,
		VmContext:    vmContext,
	}}, nil
}

func newSnapshotBlock() (*ledger.SnapshotBlock, error) {
	chainInstance := getChainInstance()

	latestBlock := chainInstance.GetLatestSnapshotBlock()
	now := time.Now()
	snapshotBlock := &ledger.SnapshotBlock{
		Height:    latestBlock.Height + 1,
		PrevHash:  latestBlock.Hash,
		Timestamp: &now,
	}

	content := chainInstance.GetNeedSnapshotContent()
	snapshotBlock.SnapshotContent = content

	trie, err := chainInstance.GenStateTrie(latestBlock.StateHash, content)
	if err != nil {
		return nil, err
	}

	snapshotBlock.StateTrie = trie
	snapshotBlock.StateHash = *trie.Hash()
	snapshotBlock.Hash = snapshotBlock.ComputeHash()

	return snapshotBlock, err
}

func TestDeleteSnapshotBlocksToHeight3(t *testing.T) {
	chainInstance := getChainInstance()
	addr1, _, _ := types.CreateAddress()
	addr2, _, _ := types.CreateAddress()
	snapshotBlock, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock)

	blocks, addressList, _ := randomSendViteBlock(snapshotBlock.Hash, &addr1, &addr2)
	chainInstance.InsertAccountBlocks(blocks)

	receiveBlock, _ := newReceiveBlock(snapshotBlock.Hash, addressList[1], blocks[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock)

	snapshotBlock2, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock2)

	blocks2, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks2)

	receiveBlock2, _ := newReceiveBlock(snapshotBlock.Hash, addressList[1], blocks2[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock2)

	needContent := chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	snapshotBlock3, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock3)

	snapshotBlock4, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock4)

	chainInstance.DeleteSnapshotBlocksToHeight(snapshotBlock3.Height)
	needContent = chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	blockMeta, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks2[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta)

	blockMeta1, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&receiveBlock2[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta1)

}
func TestDeleteSnapshotBlocksToHeight2(t *testing.T) {
	chainInstance := getChainInstance()
	snapshotBlock, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock)

	blocks, addressList, _ := randomSendViteBlock(snapshotBlock.Hash, nil, nil)
	chainInstance.InsertAccountBlocks(blocks)

	blocks2, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks2)

	needContent := chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	snapshotBlock2, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock2)

	receiveBlock, _ := newReceiveBlock(snapshotBlock2.Hash, addressList[1], blocks[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock)

	receiveBlock2, _ := newReceiveBlock(snapshotBlock2.Hash, addressList[1], blocks2[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock2)

	needContent = chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	snapshotBlock3, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock3)

	chainInstance.DeleteSnapshotBlocksToHeight(snapshotBlock2.Height)
	needContent = chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	blockMeta1, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta1)

}

func TestDeleteSnapshotBlocksToHeight(t *testing.T) {
	chainInstance := getChainInstance()

	snapshotBlock, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock)

	blocks, addressList, _ := randomSendViteBlock(snapshotBlock.Hash, nil, nil)
	chainInstance.InsertAccountBlocks(blocks)

	blocks2, addressList2, _ := randomSendViteBlock(snapshotBlock.Hash, nil, nil)
	chainInstance.InsertAccountBlocks(blocks2)

	snapshotBlock2, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock2)

	receiveBlock, _ := newReceiveBlock(snapshotBlock2.Hash, addressList[1], blocks[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock)

	receiveBlock2, _ := newReceiveBlock(snapshotBlock2.Hash, addressList2[1], blocks2[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock2)

	needContent := chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}

	fmt.Println()

	snapshotBlock3, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock3)

	var display = func() {
		//	dBlocks1, _ := chainInstance.GetAccountBlocksByHeight(blocks[0].AccountBlock.AccountAddress, 0, 10, true)
		//	for _, block := range dBlocks1 {
		//		fmt.Printf("%+v\n", block)
		//	}
		//	dBlocks2, _ := chainInstance.GetAccountBlocksByHeight(blocks2[0].AccountBlock.AccountAddress, 0, 10, true)
		//	for _, block := range dBlocks2 {
		//		fmt.Printf("%+v\n", block)
		//	}
		dBlocks3, _ := chainInstance.GetAccountBlocksByHeight(receiveBlock[0].AccountBlock.AccountAddress, 0, 10, true)
		for _, block := range dBlocks3 {
			fmt.Printf("%+v\n", block)
		}
		dBlocks4, _ := chainInstance.GetAccountBlocksByHeight(receiveBlock2[0].AccountBlock.AccountAddress, 0, 10, true)
		for _, block := range dBlocks4 {
			fmt.Printf("%+v\n", block)
		}
		fmt.Println()

		//	latestBlock := chainInstance.GetLatestSnapshotBlock()
		//	fmt.Printf("%+v\n", latestBlock)
	}

	//fmt.Println()

	blockMeta, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta)

	blockMeta1, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks2[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta1)

	needContent = chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	fmt.Println()

	display()

	chainInstance.DeleteSnapshotBlocksToHeight(snapshotBlock2.Height)
	display()
	//if deleteErr != nil {
	//	t.Fatal(deleteErr)
	//}
	//
	//for _, sb := range sbList {
	//	fmt.Printf("%+v\n", sb)
	//}
	//
	//for addr, abs := range abMap {
	//	fmt.Printf("%s\n", addr.String())
	//	for _, ab := range abs {
	//		fmt.Printf("%+v\n", ab)
	//	}
	//}
	//fmt.Println()
	//
	//display()
	//fmt.Println()
	//

	needContent = chainInstance.GetNeedSnapshotContent()
	for addr, content := range needContent {
		fmt.Printf("%s: %+v\n", addr.String(), content)
	}
	//fmt.Println()
	blockMeta_1, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta_1)

	blockMeta2_1, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&blocks2[0].AccountBlock.Hash)
	fmt.Printf("%+v\n", blockMeta2_1)

}
