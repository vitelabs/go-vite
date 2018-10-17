package api

import (
	"testing"

	"math/big"

	"strconv"

	"path/filepath"

	"os"

	"fmt"

	"sync"

	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
)

//func init() {
//	log15.Root().SetHandler(
//		log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(filepath.Join(common.DefaultDataDir(), "log"), log15.TerminalFormat())),
//	)
//}

type benchmark struct {
	passwd        string
	genesisAddr   types.Address
	coinbase      *types.Address
	w             *wallet.Manager
	v             *vite.Vite
	walletTestApi *TestApi
	onRoadApi     *PrivateOnroadApi

	testAddrList map[types.Address]string
	normalErr    error
	mlog         log15.Logger
}

func TestBenchmark(t *testing.T) {
	genesisAddr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	b := &benchmark{passwd: "123456", w: wallet.New(nil), genesisAddr: genesisAddr, coinbase: &addr}
	b.unlockAll()
	b.init()

	//waApi := NewWalletApi(vite)
	//onRoadApi := NewPrivateOnroadApi(vite)

	b.benchmark()
}

func (self *benchmark) startVite() (*vite.Vite, error) {
	//p2pServer, err := p2p.New(&p2p.Config{
	//	BootNodes: []string{
	//		"vnode://6d72c01e467e5280acf1b63f87afd5b6dcf8a596d849ddfc9ca70aab08f10191@192.168.31.146:8483",
	//		"vnode://1ceabc6c2b751b352a6d719b4987f828bb1cf51baafa4efac38bc525ed61059d@192.168.31.190:8483",
	//		"vnode://8343b3f2bc4e8e521d460cadab3e9f1e61ba57529b3fb48c5c076845c92e75d2@192.168.31.193:8483",
	//	},
	//	DataDir: path.Join(common.DefaultDataDir(), "/p2p"),
	//	NetID:   10,
	//})

	ledgerDir := "ledger_" + strconv.Itoa(os.Getpid())
	fmt.Println("vite start", self.coinbase.String(), ledgerDir)

	dir := filepath.Join(common.DefaultDataDir(), ledgerDir)
	config := &config.Config{
		DataDir: dir,
		Producer: &config.Producer{
			Producer: true,
			Coinbase: self.coinbase.String(),
		},
		Vm: &config.Vm{IsVmTest: true},
		Net: &config.Net{
			Single: true,
		},
	}
	vite, err := vite.New(config, self.w)
	if err != nil {
		panic(err)
		return nil, err
	}

	//p2pServer.Protocols = append(p2pServer.Protocols, vite.Net().Protocols()...)

	err = vite.Init()
	if err != nil {
		panic(err)
		return nil, err
	}

	//err = vite.Start(p2pServer)
	err = vite.Start(nil)
	if err != nil {
		panic(err)
		return nil, err
	}
	//err = p2pServer.Start()
	if err != nil {
		panic(err)
		return nil, err
	}
	return vite, nil
}

func (b *benchmark) getBalance(addr types.Address) *big.Int {
	balance, _ := b.v.Chain().GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
	return balance
}
func (b *benchmark) getHeight(addr types.Address) uint64 {
	block, _ := b.v.Chain().GetLatestAccountBlock(&addr)
	if block == nil {
		return 0
	}
	return block.Height
}

func (self *benchmark) unlockAll() []types.Address {
	results := self.w.KeystoreManager.Addresses()

	for _, r := range results {
		err := self.w.KeystoreManager.Unlock(r, password, 0)
		if err != nil {
			log.Error("unlock fail.", "err", err, "address", r.String())
		}
	}
	return results
}

// genesis receive
func (b *benchmark) init() {
	var err error
	b.v, err = b.startVite()
	if err != nil {
		panic(err)
	}
	prikeys := []string{"a2bf4b491654e0482cbc42d580fea5398ccb12b30b33c2caee39a4fbe2a4c1d8b193a8a33d61ecc4bfcc9a7ee6001227808408a3ee7f9543f693f6cb0cbe4d6f",
		"3b464703211b88cdc59e099e2c7fe23099a443b832eeb11c3c1c946fffa9ca85e3e9b3e0c1e49cf0f9554820a62b0fde286f43260c15fc286fd035398f7633e3",
		"58ca699195a2f6dbbc239d8d3d796d1b2d1df6a00a4809c3522dce14e59b649aea116fae16e8d7b7846082271c8dcbaf392b17784fc0ec30f5a3762cbb46abce",
		"8754650891209ff942de512d1ce5435672896976b94abd3e4fcfbe767d071b94321323e54c76640d49c021a133b58d36d31a9e909516768e7e3595489fe4edab",
		"3685004265218f9417761dd1e722b02fd606a2730124c5e2b5bfd9cd8ca365b02b68a7bb739b8899569bd9922d49a606b5f88f805ee8923b8eca046fe2993a4a",
		"51ed7aae63713f499c229fc0dfbd33c398befd66c99d2b8e12c2efc2aae9c66388bc27cc8c9bfe4a9c0b0934264814694fe3769580d1e0ae3a74f8ad85a78840",
		"415883413b0b3062d54de6302169b6725e7917c5f580a28af7b6a08d49b92cfcd4150cf94cf256093ceb7f653a9f3fac4f8b39a988ea5e09c27a4da496b65e36",
		"6ff8b6ab56d4b98b3217dc86995f98b6ced29a603bf6a5991eeda0b13876120c96f51b4edc1173f54f2bf7de3600858648773ed528346b8cb42303b2bb683e9f",
		"cdb4f58aa12f4aa5a76cc97536af0f898fd4770f68f2176270e7440deaa1e2c3b1b86393c952a3b93ce5d1ad7b7acbbe12f8d1522c5e5fa0d72d58626ec86869",
		"af88714174391e4e11277a06031a9b7a01e340c0c0835a46f8b2893b75b8b664a9d59b0d7492b3fd2a5e82abc1e5dd65cce0ba918ad4ba98fcf819a0f75a87c6",
	}
	b.testAddrList = make(map[types.Address]string)
	for _, v := range prikeys {
		privateKey, e := ed25519.HexToPrivateKey(v)
		if e != nil {
			panic(e)
		}
		address := types.PrikeyToAddress(privateKey)
		b.testAddrList[address] = v
	}
	b.walletTestApi = NewTestApi(NewWalletApi(b.v))
	b.onRoadApi = NewPrivateOnroadApi(b.v)
	b.normalErr = errors.New("normal error")
	b.mlog = log15.New("module", "benchmark")
}
func (b *benchmark) benchmark() {
	genesisPriKey, e := b.walletTestApi.walletApi.ExportPriv(b.genesisAddr, b.passwd)
	if e != nil {
		panic(e)
	}
	err := b.receive(b.genesisAddr, genesisPriKey)
	if err != nil && err != b.normalErr {
		panic(err)
	}
	err = b.transferTo(b.genesisAddr, genesisPriKey, b.testAddrList, 1000)

	if err != nil && err != b.normalErr {
		panic(err)
	}
	for k, key := range b.testAddrList {
		b.receive(k, key)
		balance := b.getBalance(k)
		b.mlog.Info("print balance", "balance", balance.String(), "addr", k.String())
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	common.Go(func() {
		defer wg.Done()
		for {
			for a, _ := range b.testAddrList {
				balance := b.getBalance(a)
				b.mlog.Info("print balance", "balance", balance.String(), "addr", a.String(), "height", b.getHeight(a))
			}
			time.Sleep(time.Second)
		}
	})

	for a, aKey := range b.testAddrList {
		tmp := a

		tmpKey := aKey
		wg.Add(1)
		common.Go(func() {
			for {
				defer wg.Done()
				addr := tmp
				addrKey := tmpKey
				l := b.mlog.New("address", addr.String())
				for {
					err := b.transferTo(addr, addrKey, b.testAddrList, 10)
					if err == b.normalErr {
						break
					}
					if err != nil {
						l.Error("transferErr", "err", err)
						break
					}
				}
				for {
					err := b.receive(addr, addrKey)
					if err == b.normalErr {
						break
					}
					if err != nil {
						l.Error("receiveErr", "err", err)
						break
					}
				}
				time.Sleep(time.Millisecond * 20)
			}
		})
	}

	wg.Wait()
}

func (b *benchmark) transferTo(from types.Address, fromKey string, toList map[types.Address]string, amount int) error {
	if b.getBalance(from).Cmp(big.NewInt(int64(amount*len(toList)))) < 0 {
		return b.normalErr
	}

	i := 0
	for to, _ := range toList {
		i++
		prevHeight := b.getHeight(from)
		err := b.transfer(from, fromKey, to, amount)
		if err != nil {
			return err
		}
		height := b.getHeight(from)
		if height-prevHeight != 1 {
			b.mlog.Error("account height", "height", height, "prevHeight", prevHeight, "diff", height-prevHeight, "addr", from.String())
		}
	}
	return nil
}

func (b *benchmark) transfer(from types.Address, fromKey string, to types.Address, amount int) error {
	param := CreateTxWithPrivKeyParmsTest{
		SelfAddr:    from,
		ToAddr:      to,
		TokenTypeId: ledger.ViteTokenId,
		PrivateKey:  fromKey,
		Amount:      strconv.Itoa(amount),
		Data:        nil,
		Difficulty:  big.NewInt(0),
	}
	err := b.walletTestApi.CreateTxWithPrivKey(param)
	return err
}
func (b *benchmark) receive(addr types.Address, akey string) error {
	for {
		blocks, err := b.onRoadApi.GetOnroadBlocksByAddress(addr, 0, 10)
		if err != nil {
			return err
		}
		if len(blocks) == 0 {
			return b.normalErr
		}
		for _, block := range blocks {
			param := CreateReceiveTxParms{
				SelfAddr:   addr,
				PrivKeyStr: akey,
				FromHash:   block.Hash,
			}
			err := b.walletTestApi.ReceiveOnroadTx(param)
			if err != nil {
				return err
			}
		}
	}
}

func TestGo(t *testing.T) {
	ss := []string{"a1", "a2", "a3", "a4"}

	for _, v := range ss {
		vvv := v
		go func() {
			fmt.Println(vvv, v)
		}()
	}
	time.Sleep(time.Second * 10)
}
