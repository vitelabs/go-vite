package main

import (
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"math/big"
	"time"
	"fmt"
	"math/rand"
	"github.com/vitelabs/go-vite/common/types"
)

var snapshotblockchain = access.GetSnapshotChainAccess()

func writeGenesisSnapshotBlock () {
	//var hash = []byte("000000000000000000")
	//var prevHash = []byte("000000000000000000")
	//block := createSnapshotBlock(hash, prevHash, big.NewInt(1))
	block := ledger.GetGenesisSnapshot()
	err := snapshotblockchain.WriteBlock(block)
	if err != nil {
		log.Fatal(err)
	}
}

func writeSnapshotChain()  {
	preBlock, glbErr := snapshotblockchain.GetLatestBlock()
	if glbErr != nil {
		log.Fatal(glbErr)
	}
	var height = &big.Int{}
	height = height.Add(preBlock.Height, big.NewInt(1))
	block := createSnapshotBlock(createHash(), preBlock.Hash, height)
	err := snapshotblockchain.WriteBlock(block)
	if err != nil {
		log.Fatal(err)
	}
}

func createSnapshotBlock (hash []byte, prevHash []byte, height *big.Int) *ledger.SnapshotBlock{
	snapshotBLock := &ledger.SnapshotBlock{
		Hash: createHash(),
		PrevHash: prevHash,
		Height: height,
		Producer: createSnapshotBlockProducer(),
		Snapshot: createSnapshot(),
		Signature: createAccountBlockSignature(),
		Timestamp: uint64(time.Time{}.Unix()),
	}
	return snapshotBLock
}

func createSnapshot () map[string] []byte{
	var snapshot map[string] [] byte
	accountList, err := snapshotblockchain.GetAccountList()
	if err != nil {
		fmt.Println("GetAccountList error")
		return nil
	}
	for _, data := range accountList {
		snapshot[data.String()] = data.Bytes()
	}
	return snapshot
}


func createHash () []byte {
	var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 18)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

func createSnapshotBlockProducer () []byte {
	accountAddressList := getAccountAddressList()
	if accountAddressList == nil {
		return []byte("000000000000000000")
	}
	return accountAddressList[rand.Intn(len(accountAddressList))].Bytes()

}

func getAccountAddressList () []*types.Address {
	var accountAddressList []*types.Address
	accountList, err := snapshotblockchain.GetAccountList()
	if err != nil {
		fmt.Println("GetAccountList error")
		return nil
	}
	for _, accountAddress := range accountList {
		accountAddressList = append(accountAddressList, accountAddress)
		fmt.Println("accountAddress: }\n", accountAddress)
	}

	return accountAddressList
}

func getSnapshotChainTest () {
	snapshotblockchain, gbErr := snapshotblockchain.GetBlockList(0,1,4)
	if gbErr !=nil {
		log.Fatal(gbErr)
	}
	fmt.Println("Length of the snapshotblockchain: ", len(snapshotblockchain))
	for _, block := range snapshotblockchain {
		fmt.Printf("Data{ Height: %d, Hash: %s, PrevHash: %s, Producer: %s }\n",
			block.Height, string(block.Hash), block.PrevHash, string(block.Producer))
	}
}
