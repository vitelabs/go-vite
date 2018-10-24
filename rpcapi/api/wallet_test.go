package api

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/vm"
	"testing"

	"github.com/vitelabs/go-vite/vm_context"

	"time"

	"math/big"

	"strconv"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/wallet"
)

var wLog = log15.New("module", "walletTest")

var genesisAccountPrivKeyStr string
var accountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "g", "", "")
	flag.StringVar(&accountPrivKeyStr, "p", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)
}

func TestParse(t *testing.T) {
}

func TestWallet(t *testing.T) {
	w := wallet.New(nil)
	password := "123456"

	unlockAll(w)
	genesisAddr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")

	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	vite, err := startVite(w, &addr, t)

	t1, _ := time.Parse(time.RFC3339, "2018-10-12T16:19:28+08:00")

	vite.Consensus().ReadByTime(types.SNAPSHOT_GID, t1)

	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite)

	//l := NewLedgerApi(vite)
	t.Log(waApi.Status())

	vite.OnRoad().StartAutoReceiveWorker(genesisAddr, nil)
	for _, v := range vite.OnRoad().ListWorkingAutoReceiveWorker() {
		wLog.Info(v.String())
	}
	// if has no balance
	if printBalance(vite, genesisAddr).Sign() == 0 {
		waitOnroad(onRoadApi, genesisAddr, t)
		//time.Sleep(time.Minute)
	}

	// if has no quota
	if printQuota(vite, genesisAddr).Sign() == 0 {
		if printPledge(vite, genesisAddr, t).Sign() == 0 {
			waitContractOnroad(onRoadApi, contracts.AddressPledge, t)

			if printPledge(vite, genesisAddr, t).Sign() == 0 {
				// wait snapshot ++
				waitSnapshotInc(vite, t)

				byt, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, genesisAddr)
				parms := CreateTransferTxParms{
					SelfAddr:    genesisAddr,
					ToAddr:      contracts.AddressPledge,
					TokenTypeId: ledger.ViteTokenId,
					Passphrase:  password,
					Amount:      new(big.Int).Mul(big.NewInt(1e4), big.NewInt(1e18)).String(),
					Data:        byt,
					Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
				}
				err = waApi.CreateTxWithPassphrase(parms)
				if err != nil {
					t.Error(err)
					return
				}
			}
			waitContractOnroad(onRoadApi, contracts.AddressPledge, t)
		}
		waitQuota(vite, genesisAddr)
	}
}

func waitQuota(vite *vite.Vite, genesisAddr types.Address) {
	for {
		quota := printQuota(vite, genesisAddr)
		if quota.Sign() > 0 {
			break
		}
		printSnapshot(vite)
		printHeight(vite, contracts.AddressPledge)
	}
}
func printPledge(vite *vite.Vite, addr types.Address, t *testing.T) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	wLog.Info("print pledge", "height", strconv.FormatUint(head.Height, 10), "pledge", amount.String(), "addr", addr.String())
	return amount
}
func startVite(w *wallet.Manager, coinbase *types.Address, t *testing.T) (*vite.Vite, error) {
	//p2pServer, err := p2p.New(&p2p.Config{
	//	BootNodes: []string{
	//		"vnode://6d72c01e467e5280acf1b63f87afd5b6dcf8a596d849ddfc9ca70aab08f10191@192.168.31.146:8483",
	//	},
	//	FullSeedStoreFileName: path.Join(common.DefaultDataDir(), "/p2p"),
	//	NetID:   10,
	//})

	wLog.Info(coinbase.String(), "coinbase")
	config := &config.Config{
		DataDir: common.DefaultDataDir(),
		Producer: &config.Producer{
			Producer: true,
			Coinbase: coinbase.String(),
		},
		Vm: &config.Vm{IsVmTest: false, IsUseVmTestParam: true},
		Net: &config.Net{
			Single: true,
		},
	}
	vite, err := vite.New(config, w)
	if err != nil {
		t.Error(err)
		return nil, err
	}

	//p2pServer.Protocols = append(p2pServer.Protocols, vite.Net().Protocols()...)

	err = vite.Init()
	if err != nil {
		t.Error(err)
		return nil, err
	}

	//err = vite.Start(p2pServer)
	err = vite.Start(nil)
	if err != nil {
		t.Error(err)
		return nil, err
	}
	//err = p2pServer.Start()
	if err != nil {
		t.Error(err)
		return nil, err
	}
	return vite, nil
}
func printHeight(vite *vite.Vite, addr types.Address) uint64 {
	height, e := vite.Chain().GetLatestAccountBlock(&addr)
	if height == nil {
		wLog.Info("print height", "height", "0", "addr", addr.String(), "err", e)
		return 0
	} else {
		wLog.Info("print height", "height", strconv.FormatUint(height.Height, 10), "addr", addr.String(), "err", e)
	}
	return height.Height

}
func printQuota(vite *vite.Vite, addr types.Address) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	quota := vite.Chain().GetPledgeQuota(head.Hash, addr)
	wLog.Info("print quota", "amount", amount.String(), "quota", quota, "snapshotHash", head.Hash, "snapshotHeight", head.Height)
	return amount
}
func printSnapshot(vite *vite.Vite) uint64 {
	block := vite.Chain().GetLatestSnapshotBlock()
	wLog.Info("print snapshot", "height", strconv.FormatUint(block.Height, 10), "hash", block.Hash)
	return block.Height
}
func printBalance(vite *vite.Vite, addr types.Address) *big.Int {
	balance, _ := vite.Chain().GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
	//t.Log(balance)
	wLog.Info("print balance", "balance", balance.String(), "addr", addr.String())
	return balance
}

func unlockAddr(w *wallet.Manager, passwd string, priKey string) types.Address {
	w.SeedStoreManagers.ImportPriv(priKey, passwd)
	accountPrivKey, _ := ed25519.HexToPrivateKey(priKey)
	accountPubKey := accountPrivKey.PubByte()
	addr := types.PubkeyToAddress(accountPubKey)

	w.SeedStoreManagers.Lock(addr)
	err := w.SeedStoreManagers.Unlock(addr, passwd, 0)
	wLog.Info("unlock address", "address", addr.String(), "r", err)
	return addr
}

func waitOnroad(api *PrivateOnroadApi, addr types.Address, t *testing.T) {
	for {
		info, e := api.GetAccountOnroadInfo(addr)
		if e != nil {
			panic(e)
			return
		}

		if info == nil {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		total := big.NewInt(0)
		total.SetString(info.TotalNumber, 10)
		for total.Sign() == 0 {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		wLog.Info("print onroad size", "size", total.String(), "addr", addr.String())
		time.Sleep(time.Second)
	}
}

func waitContractOnroad(api *PrivateOnroadApi, addr types.Address, t *testing.T) {
	for {
		info, e := api.GetOnroadBlocksByAddress(addr, 0, 1000)
		if e != nil {
			panic(e)
			return
		}

		if len(info) == 0 {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		wLog.Info("print onroad size", "size", len(info), "addr", addr.String())
		time.Sleep(time.Second)
	}
}

func waitSnapshotInc(vite *vite.Vite, t *testing.T) {
	// wait snapshot ++
	prevHeight := printSnapshot(vite)
	for {
		if printSnapshot(vite) > prevHeight {
			break
		}
		time.Sleep(time.Second)
	}
}

func onroadNum(api *PrivateOnroadApi, addr types.Address, t *testing.T) int {
	info, e := api.GetAccountOnroadInfo(addr)
	if e != nil {
		panic(e)
		return 0
	}

	if info == nil {
		wLog.Info("print onroadNum size", "size", 0, "addr", addr.String())
		return 0
	}
	total := big.NewInt(0)
	total.SetString(info.TotalNumber, 10)

	wLog.Info("print onroadNum size", "size", total.String(), "addr", addr.String())
	return int(total.Int64())
}

func contractOnroadNum(api *PrivateOnroadApi, addr types.Address, t *testing.T) int {
	info, e := api.GetOnroadBlocksByAddress(addr, 0, 1000)
	if e != nil {
		panic(e)
		return 0
	}
	if len(info) == 0 {
		wLog.Info("print contractOnroadNum size", "size", 0, "addr", addr.String())
		return 0
	}
	wLog.Info("print contractOnroadNum size", "size", len(info), "addr", addr.String())
	return len(info)
}

func TestGenData(t *testing.T) {
	w := wallet.New(nil)

	unlockAll(w)

	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")

	vite, err := startVite(w, &addr, t)
	if err != nil {
		panic(err)
	}
	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite)

	printBalance(vite, addr)

	genesisAddr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	vite.OnRoad().StartAutoReceiveWorker(genesisAddr, nil)

	// if has no balance
	if printBalance(vite, genesisAddr).Sign() == 0 {
		waitOnroad(onRoadApi, genesisAddr, t)
	}

	waitSnapshotInc(vite, t)

	parms := CreateTransferTxParms{
		SelfAddr:    genesisAddr,
		ToAddr:      addr,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      new(big.Int).Mul(big.NewInt(3300000), big.NewInt(1e18)).String(),
		Data:        nil,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err = waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		panic(err)
	}

	vite.OnRoad().StartAutoReceiveWorker(addr, nil)
	waitOnroad(onRoadApi, addr, t)
	printBalance(vite, addr)
	waitSnapshotInc(vite, t)
}

var password = "123456"

func unlockAll(w *wallet.Manager) []types.Address {
	results := w.SeedStoreManagers.Addresses()

	for _, r := range results {
		err := w.SeedStoreManagers.Unlock(r, password, 0)
		if err != nil {
			log.Error("unlock fail.", "err", err, "address", r.String())
		}
	}
	return results
}

func TestQuota(t *testing.T) {
	w := wallet.New(nil)
	unlockAll(w)
	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	vite, _ := startVite(w, &addr, t)

	//waitQuota(vite, addr)

	snapshotBlock := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(snapshotBlock.Hash, addr)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	db, _ := vm_context.NewVmContext(vite.Chain(), &snapshotBlock.Hash, &prevBlock.Hash, &addr)
	pledgeAmount := contracts.GetPledgeBeneficialAmount(db, addr)

	wLog.Debug("print pledge amount", "chain", amount, "vm", pledgeAmount)
}

func TestContracts(t *testing.T) {
	vm.InitVmConfig(false, true)
	vite, _, waApi, onRoadApi, addr := contractsInit(t)
	contractsPledge(vite, waApi, onRoadApi, addr, t)
	tokenId := contractsMintage(vite, waApi, onRoadApi, addr, t)
	contractsCancelMintage(vite, waApi, onRoadApi, addr, t, tokenId)
	gid := contractsCreateConsensusGroup(vite, waApi, onRoadApi, addr, t)
	contractsCancelConsensusGroup(vite, waApi, onRoadApi, addr, t, gid)
	contractsRecreateConsensusGroup(vite, waApi, onRoadApi, addr, t, gid)
	nodeName := "MySuperNode1"
	contractsRegister(vite, waApi, onRoadApi, addr, t, gid, nodeName)
	contractsUpdateRegister(vite, waApi, onRoadApi, addr, t, gid, nodeName)
	contractsVote(vite, waApi, onRoadApi, addr, t, gid, nodeName)
	contractsCancelVote(vite, waApi, onRoadApi, addr, t, gid)
	contractsCancelRegister(vite, waApi, onRoadApi, addr, t, gid, nodeName)
	contractsCancelPledge(vite, waApi, onRoadApi, addr, t)
	contractsReward(vite, waApi, onRoadApi, addr, t)
}

func contractsInit(t *testing.T) (*vite.Vite, *wallet.Manager, *WalletApi, *PrivateOnroadApi, types.Address) {
	wLog.Debug("contracts init")
	w := wallet.New(nil)
	unlockAll(w)
	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	vite, err := startVite(w, &addr, t)
	if err != nil {
		panic(err)
	}

	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite)

	vite.OnRoad().StartAutoReceiveWorker(addr, nil)
	waitContractOnroad(onRoadApi, contracts.AddressPledge, t)
	waitOnroad(onRoadApi, addr, t)

	waitSnapshotInc(vite, t)
	return vite, w, waApi, onRoadApi, addr
}
func contractsPledge(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T) {
	wLog.Debug("contracts pledge")
	pledgeAmount := printPledge(vite, addr, t)
	balance := printBalance(vite, addr)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr)
	amount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressPledge,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      amount.String(),
		Data:        pledgeData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Error(err)
		return
	}

	waitContractOnroad(onRoadApi, contracts.AddressPledge, t)
	waitSnapshotInc(vite, t)

	newBalance := printBalance(vite, addr)
	balance.Sub(balance, amount)
	if balance.Cmp(newBalance) != 0 {
		t.Fatalf("pledge balance error, expected %v, got %v", balance, newBalance)
	}

	newPledgeAmount := printPledge(vite, addr, t)
	pledgeAmount.Add(pledgeAmount, amount)
	if pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatal("pledge amount error, expected: %v, got %v", pledgeAmount, newPledgeAmount)
	}
}
func contractsCancelPledge(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T) {
	wLog.Debug("contracts cancel pledge")
	pledgeAmount := printPledge(vite, addr, t)
	balance := printBalance(vite, addr)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	amount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNameCancelPledge,
		addr, amount)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressPledge,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        pledgeData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Error(err)
		return
	}

	waitContractOnroad(onRoadApi, contracts.AddressPledge, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	newBalance := printBalance(vite, addr)
	balance.Add(balance, amount)
	if balance.Cmp(newBalance) != 0 {
		t.Fatalf("pledge balance error, expected %v, got %v", balance, newBalance)
	}

	newPledgeAmount := printPledge(vite, addr, t)
	pledgeAmount.Sub(pledgeAmount, amount)
	if pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatal("pledge amount error, expected: %v, got %v", pledgeAmount, newPledgeAmount)
	}
}
func contractsMintage(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T) types.TokenTypeId {
	wLog.Debug("contracts mintage")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	balance := printBalance(vite, addr)
	printQuota(vite, addr)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	tokenId := contracts.NewTokenId(addr, prevBlock.Height+1, prevBlock.Hash, vite.Chain().GetLatestSnapshotBlock().Hash)
	mintageData, _ := contracts.ABIMintage.PackMethod(contracts.MethodNameMintage,
		tokenId,
		"MyToken",
		"mt",
		big.NewInt(1e18),
		uint8(0))
	mintagePledgeAmount := new(big.Int).Mul(big.NewInt(1e5), big.NewInt(1e18))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressMintage,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      mintagePledgeAmount.String(),
		Data:        mintageData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressMintage, t)
	waitSnapshotInc(vite, t)

	amount, err := vite.Chain().GetAccountBalanceByTokenId(&addr, &tokenId)
	if amount.Cmp(big.NewInt(1e18)) != 0 {
		t.Fatal("token amount error: %v", amount)
	}

	balance.Sub(balance, mintagePledgeAmount)
	if balance.Cmp(printBalance(vite, addr)) != 0 {
		t.Fatal("mintage fee error")
	}
	printQuota(vite, addr)
	tokenInfo := vite.Chain().GetTokenInfoById(&tokenId)
	if tokenInfo == nil {
		t.Fatal("token info not exist")
	}
	wLog.Debug("token info", tokenId.String(), tokenInfo)
	return tokenId
}
func contractsCancelMintage(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, tokenId types.TokenTypeId) {
	wLog.Debug("contracts cancel mintage")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	balance := printBalance(vite, addr)
	printQuota(vite, addr)

	cancelMintageData, _ := contracts.ABIMintage.PackMethod(contracts.MethodNameMintageCancelPledge, tokenId)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressMintage,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        cancelMintageData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressMintage, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	mintagePledgeAmount := new(big.Int).Mul(big.NewInt(1e5), big.NewInt(1e18))
	balance.Add(balance, mintagePledgeAmount)
	if balance.Cmp(printBalance(vite, addr)) != 0 {
		t.Fatal("mintage cancel pledge error")
	}
	printQuota(vite, addr)
	tokenInfo := vite.Chain().GetTokenInfoById(&tokenId)
	wLog.Debug("token info", tokenId.String(), tokenInfo)
}
func contractsCreateConsensusGroup(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T) types.Gid {
	wLog.Debug("contracts create consensus group")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	balance := printBalance(vite, addr)
	list := vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash)
	length := len(list)
	wLog.Debug("init consensus group list", "length", length)
	for _, g := range list {
		wLog.Debug("init consensus group list", g.Gid.String(), g)
	}

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	gid := contracts.NewGid(addr, prevBlock.Height+1, prevBlock.Hash, vite.Chain().GetLatestSnapshotBlock().Hash)
	registerConditionData, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionRegisterOfPledge, big.NewInt(1), ledger.ViteTokenId, uint64(1))
	createConsensusGroupData, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCreateConsensusGroup,
		gid,
		uint8(3),
		int64(1),
		int64(1),
		uint8(0),
		uint8(0),
		ledger.ViteTokenId,
		uint8(1),
		registerConditionData,
		uint8(1),
		[]byte{})
	pledgeAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressConsensusGroup,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      pledgeAmount.String(),
		Data:        createConsensusGroupData,
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressConsensusGroup, t)
	waitSnapshotInc(vite, t)

	list = vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash)
	wLog.Debug("create consensus group list", "length", len(list))
	for _, g := range list {
		wLog.Debug("create consensus group list", g.Gid.String(), g)
	}
	if len(list) != length+1 {
		t.Fatalf("create consensus group failed")
	}

	balance.Sub(balance, pledgeAmount)
	if balance.Cmp(printBalance(vite, addr)) != 0 {
		t.Fatal("create consensus group fee error")
	}
	printQuota(vite, addr)
	return gid
}
func contractsCancelConsensusGroup(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid) {
	wLog.Debug("contracts cancel consensus group")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	balance := printBalance(vite, addr)
	length := len(vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash))

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	cancelConsensusGroupData, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCancelConsensusGroup, gid)
	pledgeAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressConsensusGroup,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        cancelConsensusGroupData,
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressConsensusGroup, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	list := vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash)
	wLog.Debug("cancel consensus group list", "length", len(list))
	for _, g := range list {
		wLog.Debug("cancel consensus group list", g.Gid.String(), g)
	}
	if len(list) != length-1 {
		t.Fatalf("cancel consensus group failed")
	}

	balance.Add(balance, pledgeAmount)
	if balance.Cmp(printBalance(vite, addr)) != 0 {
		t.Fatal("cancel consensus group get fee error")
	}
	printQuota(vite, addr)
}
func contractsRecreateConsensusGroup(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid) {
	wLog.Debug("contracts recreate consensus group")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	balance := printBalance(vite, addr)
	length := len(vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash))

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	recreateConsensusGroupData, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameReCreateConsensusGroup, gid)
	pledgeAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressConsensusGroup,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      pledgeAmount.String(),
		Data:        recreateConsensusGroupData,
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressConsensusGroup, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	list := vite.Chain().GetConsensusGroupList(vite.Chain().GetLatestSnapshotBlock().Hash)
	wLog.Debug("recreate consensus group list", "length", len(list))
	for _, g := range list {
		wLog.Debug("recreate consensus group list", g.Gid.String(), g)
	}
	if len(list) != length+1 {
		t.Fatalf("recreate consensus group failed")
	}

	balance.Sub(balance, pledgeAmount)
	if balance.Cmp(printBalance(vite, addr)) != 0 {
		t.Fatal("recreate consensus group get fee error")
	}
	printQuota(vite, addr)
}
func contractsRegister(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid, name string) {
	wLog.Debug("contracts register")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	balance := printBalance(vite, addr)
	if printQuota(vite, addr).Sign() == 0 {
		t.Fatalf("no pledge")
	}
	registerPledgeAmount := big.NewInt(1)
	registerList := vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	length := len(registerList)
	wLog.Debug("init register list", "length", len(registerList))
	for _, registration := range registerList {
		wLog.Debug("init register list", registration.NodeAddr.String(), registration)
	}

	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	nodeAddr, privateKey, _ := types.CreateAddress()
	publicKey := privateKey.PubByte()
	signature := ed25519.Sign(privateKey, contracts.GetRegisterMessageForSignature(addr, gid))
	registerData, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameRegister,
		gid,
		name,
		nodeAddr,
		publicKey,
		signature)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressRegister,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      registerPledgeAmount.String(),
		Data:        registerData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Error(err)
		return
	}

	waitContractOnroad(onRoadApi, contracts.AddressRegister, t)
	waitSnapshotInc(vite, t)

	balance.Sub(balance, registerPledgeAmount)
	newBalance := printBalance(vite, addr)
	if balance.Cmp(newBalance) != 0 {
		t.Fatalf("register pledge amount error: expected %v , got %v", balance, newBalance)
	}

	printQuota(vite, addr)
	registerList = vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	wLog.Debug("register list", "length", len(registerList))
	for _, registration := range registerList {
		wLog.Debug("register list", registration.NodeAddr.String(), registration)
	}
	if len(registerList) != length+1 {
		t.Fatal("register error")
	}
}
func contractsUpdateRegister(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid, name string) {
	wLog.Debug("contracts update register")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	balance := printBalance(vite, addr)
	length := len(vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid))

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	nodeAddr, privateKey, _ := types.CreateAddress()
	publicKey := privateKey.PubByte()
	signature := ed25519.Sign(privateKey, contracts.GetRegisterMessageForSignature(addr, gid))
	registerData, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration,
		gid,
		name,
		nodeAddr,
		publicKey,
		signature)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressRegister,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        registerData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressRegister, t)
	waitSnapshotInc(vite, t)

	newBalance := printBalance(vite, addr)
	if balance.Cmp(newBalance) != 0 {
		t.Fatalf("update register cost 0: expected %v , got %v", balance, newBalance)
	}

	printQuota(vite, addr)
	registerList := vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	wLog.Debug("update register list", "length", len(registerList))
	for _, registration := range registerList {
		wLog.Debug("update register list", registration.NodeAddr.String(), registration)
	}
	if len(registerList) != length {
		t.Fatal("update register error")
	}
}
func contractsCancelRegister(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid, name string) {
	wLog.Debug("contracts cancel register")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	balance := printBalance(vite, addr)
	length := len(vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid))

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	registerData, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister,
		gid,
		name)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressRegister,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        registerData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressRegister, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	newBalance := printBalance(vite, addr)
	registerPledgeAmount := big.NewInt(1)
	balance.Add(balance, registerPledgeAmount)
	if balance.Cmp(newBalance) != 0 {
		t.Fatalf("cancel register gets pledge error: expected %v , got %v", balance, newBalance)
	}

	printQuota(vite, addr)
	registerList := vite.Chain().GetRegisterList(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	wLog.Debug("update register list", "length", len(registerList))
	for _, registration := range registerList {
		wLog.Debug("cancel register list", registration.NodeAddr.String(), registration)
	}
	if len(registerList) != length-1 {
		t.Fatal("cancel register error")
	}
}
func contractsVote(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid, name string) {
	wLog.Debug("contracts vote")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	printBalance(vite, addr)
	printQuota(vite, addr)
	voteList := vite.Chain().GetVoteMap(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	length := len(voteList)
	wLog.Debug("init vote list", "length", len(voteList))
	for _, vote := range voteList {
		wLog.Debug("init vote list", vote.NodeName, vote.VoterAddr)
	}

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	voteData, _ := contracts.ABIVote.PackMethod(contracts.MethodNameVote,
		gid,
		name)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressVote,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        voteData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
		return
	}

	waitContractOnroad(onRoadApi, contracts.AddressVote, t)
	waitSnapshotInc(vite, t)

	printBalance(vite, addr)
	printQuota(vite, addr)
	voteList = vite.Chain().GetVoteMap(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	wLog.Debug("vote list", "length", len(voteList))
	for _, vote := range voteList {
		wLog.Debug("vote list", vote.NodeName, vote.VoterAddr)
	}
	if len(voteList) != length+1 {
		t.Fatal("vote error")
	}
}
func contractsCancelVote(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T, gid types.Gid) {
	wLog.Debug("contracts cancel vote")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)
	length := len(vite.Chain().GetVoteMap(vite.Chain().GetLatestSnapshotBlock().Hash, gid))

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	voteData, _ := contracts.ABIVote.PackMethod(contracts.MethodNameCancelVote,
		gid)
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressVote,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        voteData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
		return
	}

	waitContractOnroad(onRoadApi, contracts.AddressVote, t)
	waitSnapshotInc(vite, t)

	printBalance(vite, addr)
	printQuota(vite, addr)
	voteList := vite.Chain().GetVoteMap(vite.Chain().GetLatestSnapshotBlock().Hash, gid)
	wLog.Debug("cancel vote list", "length", len(voteList))
	for _, vote := range voteList {
		wLog.Debug("cancel vote list", vote.NodeName, vote.VoterAddr)
	}
	if len(voteList) != length-1 {
		t.Fatal("cancel vote error")
	}
}
func contractsReward(vite *vite.Vite, waApi *WalletApi, onRoadApi *PrivateOnroadApi, addr types.Address, t *testing.T) {
	wLog.Debug("contracts reward")
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	balance := printBalance(vite, addr)

	count := int64(0)
	snapshotBlockList, _ := vite.Chain().GetSnapshotBlocksByHeight(1, 10, true, false)
	for _, b := range snapshotBlockList {
		if b.Producer() == addr {
			count = count + 1
		}
	}
	if count == 0 {
		t.Fatalf("no snapshot block")
	}
	wLog.Debug("print produce snapshot block count", "count", count)

	prevBlock, _ := vite.Chain().GetLatestAccountBlock(&addr)
	if prevBlock == nil {
		t.Fatalf("prev block not exist")
	}
	rewardAmount := new(big.Int).Div(new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)), big.NewInt(1051200000))
	rewardAmount = rewardAmount.Mul(rewardAmount, big.NewInt(count))

	rewardData, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameReward, types.SNAPSHOT_GID, "s3", addr, uint64(10), uint64(1))
	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressRegister,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      big.NewInt(0).String(),
		Data:        rewardData,
		Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
	}
	err := waApi.CreateTxWithPassphrase(parms)
	if err != nil {
		t.Fatal(err)
	}

	waitContractOnroad(onRoadApi, contracts.AddressRegister, t)
	waitOnroad(onRoadApi, addr, t)
	waitSnapshotInc(vite, t)

	newBalance := printBalance(vite, addr)
	balance.Add(balance, rewardAmount)
	if newBalance.Cmp(balance) != 0 {
		t.Fatalf("get reward balance error, expected %v, got %v", balance, newBalance)
	}
}
