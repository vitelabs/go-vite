package api

import (
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
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

import _ "net/http/pprof"

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
	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()
	log15.Root().SetHandler(
		log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(filepath.Join(common.DefaultDataDir(), "log"), log15.TerminalFormat())),
	)

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
	//	StaticNodes: []string{
	//		"vnode://6d72c01e467e5280acf1b63f87afd5b6dcf8a596d849ddfc9ca70aab08f10191@192.168.31.146:8483",
	//		"vnode://a0cab03dfb22ae4294efe30e7408b752fb659676751a8d36c943594a25dc23b4@192.168.31.46:8483",
	//		//"vnode://1ceabc6c2b751b352a6d719b4987f828bb1cf51baafa4efac38bc525ed61059d@192.168.31.190:8483",
	//		"vnode://8343b3f2bc4e8e521d460cadab3e9f1e61ba57529b3fb48c5c076845c92e75d2@192.168.31.193:8483",
	//	},
	//	DataDir: path.Join(common.DefaultDataDir(), "/p2p"),
	//	NetID:   10,
	//	//Discovery: true,
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
		Vm: &config.Vm{IsVmTest: true, IsUseVmTestParam: true},
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
	//err = vite.Start(nil)
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
	prikeys := []string{
		"3117f293b1f1aeea9ae65cc1bebe8512b98cce300356ac0489876feac0981120633a5dbed75e1727cc437cfdfb8d92bb5de217a6642d5e9f1ff5702cd1f8a479",
		"cdd493eac75fc34eb534ac68bb84d59e4a345ef6b8e2ab04a2e0db7230d2b2aff4200175f484e11e5bf956dea15afa73fd96e469fe95fa87671472266cb0405f",
		"f50f51b45eb344872208b4e6a0388d9248ff89cf167489e43c6db01a23f8f7297a202e6fc7ed56191b1c26853313be9e1d3f301172f062af66084a8547ae4fd5",
		"be500095018c15390784aca3ba520315590d908fa3765ac1075af039b9a91781e32883810aa64d340306c737acd24e36a5086f46d03b339a3951e6a134365bae",
		"7eaa722695c8a4c77aecd90d8b8ce583d5a5c20cef36bd168e40e3a494c6e942cf750e209a411a23810709453f5958383599da5b5f3bc72d8aa780ebd3ee252f",
		"9f675261cc2da29e00a82bacb4d42ee42d15f6c4185fafa9dc032a22067570cf0f5071b58979d20d29d7de19d26e7fedefee2ff90ae54e4de3b2deaf36b0dc08",
		"3b82d1ebf96560e43ea7bb631f8bbe39d21472c47e261a31fe7e5cfdcf8df7751d4ffc3587b527a4f0374cff7504ebff23231b3468c030ba54e7975f16255ed9",
		"8cc0c24d239afc3eda93fe3f5592e6bac78998b1d6f95fe237465c4e30950d882165b1da324018ecc689a5af108148ce334ee5103ac89fd5ff739f8ff6439633",
		"9166cfbdf68f5dce3b9f903953549436f34cc88e6f7a425be73625d452d3c21dde5084fd60d181ad3edb6359a8224fb5ffd321494890210c13ff43ccacfde0e6",
		"119511ca34dc8ff4e41e86ac7ccc0c000dd54612d12b8b12255923d4ea5eb2f53c6f346fb8fe541c544232c96c940ca0cefe8378572a78b2405fac81a8e2ce03",
		"2614996b86af7fee8a749b52e72d4f68e46a4d899234f19291afb33f0e49cd8f548a2f9f145487880e11255154f67c9caccd670558e58f7b5086d843a2bef65b",
		"5293930e7175315f58571636968df715e9e047878d0a29dbbff2c52d43144cf346897cf598021174b23d8b0da31c64eca5e7b0a963c8bbcf27bd9eed7d2fe340",
		"bd6b8559af2e12ee9e64694a0d8a0c4a9d3cce01d0af92abe7fdb747bda36943caeab9487a2c6041fbc79ee4fdc2ecdca8f3b76478dcdc8671adca72adee23d2",
		"a519e52896cbb213c29f10bb098cbdc17fd7a6b9846f022457d28c47daca15ab54b1ccba28447ec1d481652222fb4d6a18a8667f88e305fbd289a74932a722ad",
		"acd70f768e8df22d40f4de0a5a9a4eea8f07ee77b5796e82ba85fb528acdb1ded1bc874bfb8ab8a44c0b29d6eba39e018cb4bb2deded13f36ad244aadd42f2d0",
		"ae474e560aabf7db18e33d87b9725ba9e9a9e22015b28c51dea27615973860001340dc484a18e1af9b06fe42e9f63a827318e9bc3dd479ee7e7b1202ae206550",
		"5ada7978dca836774be972dc9531c03c440907b8d3e439748fb421b8440050b24b97405ccacffb26884bb16ca9b24f20bb1e58ca4fa8a68e34b8d0f8ccdd6af4",
		"064d85cf46e3879e371dd49eecff6b7e78ea698b7a4739fe65ee645a231c183d7eb0dee4811b50ac3ed942d928738b438f29c8301b3cefbef238fe0a7c8afca9",
		"0dcf90efdcdc6cb8439edfec774a4742ec2b19f1ec3c7060526957dcb8642b3430f462d21f1ba3a4a8249dea4ab58285f97913a840299d8a71a58d010beab659",
		"250b7a817887f11f18b9c93fbc7bd8f175fa3e074581f3a92c74db3fbffdb16a8f46d6fa7db0d0e32b35335d669b615f25e9ceb094213b12c71e9baec5cd63d5",
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
	b.mlog.Info("print balance genesis", "balance", b.getBalance(b.genesisAddr))
	err = b.transferTo(b.genesisAddr, genesisPriKey, b.testAddrList, 1000)

	if err != nil && err != b.normalErr {
		panic(err)
	}
	for k, key := range b.testAddrList {
		err := b.receive(k, key)
		if err != b.normalErr {
			b.mlog.Error("receive error", "err", err)
		}
		balance := b.getBalance(k)
		b.mlog.Info("print balance first", "balance", balance.String(), "addr", k.String())
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	common.Go(func() {
		defer wg.Done()
		for {
			for a, _ := range b.testAddrList {
				balance := b.getBalance(a)
				b.mlog.Error("print balance", "balance", balance.String(), "addr", a.String(), "height", b.getHeight(a))
			}
			time.Sleep(time.Second)
		}
	})

	//b.v.OnRoad().Stop()
	//for {
	//	err := b.transfer(b.genesisAddr, genesisPriKey, b.genesisAddr, 0)
	//	if err != nil {
	//		panic(err)
	//	}
	//}

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
					err := b.transferTo(addr, addrKey, b.testAddrList, 40)
					if err == b.normalErr {
						break
					}
					if err != nil {
						l.Error("transferErr", "err", err)
						break
					}
				}
				time.Sleep(time.Millisecond * 50)
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
				time.Sleep(time.Millisecond * 50)
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

	for _, v := range ss {
		vvv := v
		common.Go(func() {
			fmt.Println(vvv, v)
		})
	}
	time.Sleep(time.Second * 10)
}
