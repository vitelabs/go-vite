package gvite

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"fmt"
	"github.com/vitelabs/go-vite/vite"
	rand2 "math/rand"
	"time"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"log"

	"flag"
	"github.com/vitelabs/go-vite/p2p"
)

func Start (cfg *p2p.Config)  {
	//publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	//addr := types.PubkeyToAddress(publicKey)
	//
	//fmt.Println(publicKey.Hex())
	//fmt.Println(privateKey.Hex())
	//fmt.Println(addr.Hex())


	v, err := vite.New(cfg)
	if err != nil {
		log.Fatal(err)
	}


	//v.WalletManager().KeystoreManager.ImportPriv(AccountMockDataList[0].PrivateKey.Hex(), "123456")
	//v.WalletManager().KeystoreManager.Unlock(AccountMockDataList[0].Addr, "123456", 0)
	//for  {
	//	v.Ledger().Ac().CreateTx(createSendBlock(&AccountMockDataList[0].Addr, &AccountMockDataList[1].Addr, big.NewInt(int64(1000))))
	//}


	var isMinting bool
	flag.BoolVar(&isMinting,"ism", false, "Is minting")

	var accountIndex int
	flag.IntVar(&accountIndex,"account", 0, "Mock what account")
	flag.Parse()

	if isMinting {
		mockSnapshot(v)
	} else {
		mockAccount(v, accountIndex)
	}
}

type AccountMockData struct {
	PublicKey ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	Addr types.Address
}

var AccountMockDataList []*AccountMockData

func init ()  {
	var GPublicKey, _ = ed25519.HexToPublicKey("3af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a")
	var GPrivateKey, _ = ed25519.HexToPrivateKey("ab565d7d8819a3548dbdae8561796ccb090692086ff7d5a47eb7b034497cabe73af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a")
	var GAddr = types.PubkeyToAddress(GPublicKey)

	AccountMockDataList = append(AccountMockDataList, &AccountMockData{
		PublicKey: GPublicKey,
		PrivateKey: GPrivateKey,
		Addr: GAddr,
	})


	var publicKey, _ = ed25519.HexToPublicKey("6cbdba33180d9a83156dbaa363e8100721edd76959b10cd4f541f5e44b688ad4")
	var privateKey, _ =  ed25519.HexToPrivateKey("812072f06fc86ad266b2806be2756e0992e01bfdc83334c229fcc9d8e4689ed96cbdba33180d9a83156dbaa363e8100721edd76959b10cd4f541f5e44b688ad4")
	var addr, _ = types.HexToAddress("vite_e1b2f973857f30f0ecdad8481d7c57f6b6d8caf0f2d2351d7b")

	AccountMockDataList = append(AccountMockDataList, &AccountMockData{
		PublicKey: publicKey,
		PrivateKey: privateKey,
		Addr: addr,
	})

	var publicKey2, _ = ed25519.HexToPublicKey("ced3a94608cefc07df000431ea1724599da00161195e61080b07413be5f274f1")
	var privateKey2, _ =  ed25519.HexToPrivateKey("d02d30ef2274693061ff11400a8f1a6db42ae6084e76e82fffe2b06b2251ba55ced3a94608cefc07df000431ea1724599da00161195e61080b07413be5f274f1")
	var addr2, _ = types.HexToAddress("vite_119924a94a91070ab7b4c23f4bcee88d3765959986199e85b7")

	AccountMockDataList = append(AccountMockDataList, &AccountMockData{
		PublicKey: publicKey2,
		PrivateKey: privateKey2,
		Addr: addr2,
	})

}



func createSendBlock (addr *types.Address, toAddr *types.Address, Amount *big.Int) *ledger.AccountBlock  {
	return &ledger.AccountBlock{
		AccountAddress: addr,
		To: toAddr,
		Amount: Amount,
		TokenId: &ledger.MockViteTokenId,
	}
}

func createReceiveBlock (addr *types.Address, fromHash *types.Hash) *ledger.AccountBlock {
	return &ledger.AccountBlock{
		AccountAddress: addr,
		FromHash: fromHash,
	}
}

func createSnapshotBlock (v *vite.Vite) *ledger.SnapshotBlock {
	accountBlockList, err := v.Ledger().Sc().GetNeedSnapshot()
	var snapshot = make(map[string]*ledger.SnapshotItem, len(accountBlockList))
	for _, accountBlock := range accountBlockList {
		snapshot[accountBlock.AccountAddress.Hex()] = &ledger.SnapshotItem{
			AccountBlockHeight: accountBlock.Meta.Height,
			AccountBlockHash: accountBlock.Hash,
		}
	}

	if err != nil {
		log.Println(err)
	}
	return &ledger.SnapshotBlock{
		Producer: &AccountMockDataList[0].Addr,
		Snapshot: snapshot,
	}
}

func mockSnapshot (v *vite.Vite)  {
	fmt.Println("Current AccountAddress: ", AccountMockDataList[0].Addr.Hex())
	fmt.Println("Current PublicKey: ", AccountMockDataList[0].PublicKey.Hex())
	fmt.Println("Current PrivateKey: ", AccountMockDataList[0].PrivateKey.Hex())

	channel := make(chan int)
	go func(signal chan<- int) {
		fmt.Println(v.WalletManager().KeystoreManager)
		fmt.Println(AccountMockDataList)

		v.WalletManager().KeystoreManager.ImportPriv(AccountMockDataList[0].PrivateKey.Hex(), "123456")
		v.WalletManager().KeystoreManager.Unlock(AccountMockDataList[0].Addr, "123456", 0)

		for {
			log.Println("Mock minting.")
			time.Sleep(time.Duration(30 * time.Second))
			snapshotBlock := createSnapshotBlock(v)
			if snapshotBlock.Snapshot == nil  || len(snapshotBlock.Snapshot) == 0 {
				log.Println("No new account blocks. Doesn't snapshot.")
			} else {
				v.Ledger().Sc().WriteMiningBlock(snapshotBlock)
				log.Println("The snapshot block " + snapshotBlock.Hash.String() + " create success.")
			}
		}
	}(channel)
	<- channel
}

func mockAccount (v *vite.Vite, index int) {
	//publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	// addr, _ := types.PubkeyToAddress(publicKey)
	fmt.Println("Current AccountAddress: ", AccountMockDataList[index].Addr.Hex())
	fmt.Println("Current PublicKey: ", AccountMockDataList[index].PublicKey.Hex())
	fmt.Println("Current PrivateKey: ", AccountMockDataList[index].PrivateKey.Hex())

	channel := make(chan int)
	go func(signal chan<- int) {
		v.WalletManager().KeystoreManager.ImportPriv(AccountMockDataList[index].PrivateKey.Hex(), "123456")
		v.WalletManager().KeystoreManager.Unlock(AccountMockDataList[index].Addr, "123456", 0)

		for {
			log.Println("Mock account.")
			num := rand2.Intn(3000)
			time.Sleep(time.Duration(num) * time.Millisecond)

			if index == 0 {
				accountIndex := rand2.Intn(2)
				amount := rand2.Intn(10000)
				v.Ledger().Ac().CreateTx(createSendBlock(&AccountMockDataList[0].Addr, &AccountMockDataList[accountIndex + 1].Addr, big.NewInt(int64(amount))))
			} else {
				//v.Ledger().Ac().CreateTx(mockReceiveBlock())
			}
		}
	}(channel)
	<- channel
}