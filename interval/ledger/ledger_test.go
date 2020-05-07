package ledger

import (
	"fmt"
	"testing"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/consensus"
	"github.com/vitelabs/go-vite/interval/miner"
	"github.com/vitelabs/go-vite/interval/p2p"
	"github.com/vitelabs/go-vite/interval/syncer"
	"github.com/vitelabs/go-vite/interval/test"
	"github.com/vitelabs/go-vite/interval/utils"
)

type TestSyncer struct {
	Blocks map[string]*test.TestBlock
	f      syncer.Fetcher
}

func (self *TestSyncer) BroadcastAccountBlocks(string, []*common.AccountStateBlock) error {
	return nil
}

func (self *TestSyncer) BroadcastSnapshotBlocks([]*common.SnapshotBlock) error {
	return nil
}

func (self *TestSyncer) SendAccountBlocks(string, []*common.AccountStateBlock, p2p.Peer) error {
	panic("implement me")
}

func (self *TestSyncer) SendSnapshotBlocks([]*common.SnapshotBlock, p2p.Peer) error {
	panic("implement me")
}

func (self *TestSyncer) SendAccountHashes(string, []common.HashHeight, p2p.Peer) error {
	panic("implement me")
}

func (self *TestSyncer) SendSnapshotHashes([]common.HashHeight, p2p.Peer) error {
	panic("implement me")
}

func (self *TestSyncer) RequestAccountHash(string, common.HashHeight, int) error {
	panic("implement me")
}

func (self *TestSyncer) RequestSnapshotHash(common.HashHeight, int) error {
	panic("implement me")
}

func (self *TestSyncer) RequestAccountBlocks(string, []common.HashHeight) error {
	panic("implement me")
}

func (self *TestSyncer) RequestSnapshotBlocks([]common.HashHeight) error {
	panic("implement me")
}

func (self *TestSyncer) Init(face.ChainRw) {

}

func (self *TestSyncer) Done() bool {
	return true
}

func (self *TestSyncer) DefaultHandler() syncer.MsgHandler {
	panic("implement me")
}

func NewTestSync() *TestSyncer {
	testSyncer := &TestSyncer{Blocks: make(map[string]*test.TestBlock)}
	testSyncer.f = &TestFetcher{}
	return testSyncer
}

func (self *TestSyncer) Fetcher() syncer.Fetcher {
	return self.f
}

func (self *TestSyncer) Sender() syncer.Sender {
	return self
}

func (self *TestSyncer) Handlers() syncer.Handlers {
	panic("implement me")
}

type TestFetcher struct {
}

func (*TestFetcher) FetchAccount(address string, hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}

func (*TestFetcher) FetchSnapshot(hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}

func TestTime(t *testing.T) {
	now := time.Now()
	fmt.Printf("%d\n", now.Unix())
	block := common.NewSnapshotBlock(0, "460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b", "", "viteshan", time.Unix(1533550878, 0), nil)
	hash := utils.CalculateSnapshotHash(block)
	fmt.Printf("hash:%s\n", hash) //460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b
}

func TestLedger(t *testing.T) {
	testSyncer := NewTestSync()
	ledger := NewLedger()
	ledger.Init(testSyncer)
	ledger.Start()

	ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	viteshan := "viteshan"
	reqs := ledger.reqPool.getReqs(viteshan)
	if len(reqs) > 0 {
		t.Errorf("reqs should be empty. reqs:%v", reqs)
	}

	viteshan1 := "viteshan1"
	time.Sleep(5 * time.Second)
	{
		var err error
		err = ledger.RequestAccountBlock(viteshan1, viteshan, -105)
		if err == nil {
			t.Error("expected error.")
		} else {
			log.Info("error:%v", err)
		}
	}
	{
		err := ledger.RequestAccountBlock(viteshan1, viteshan, -90)
		if err != nil {
			t.Errorf("expected error.%v", err)
		}
	}
	{
		reqs = ledger.reqPool.getReqs(viteshan)
		if len(reqs) != 1 {
			t.Errorf("reqs should be empty. reqs:%v", reqs)
		}
		req := reqs[0]

		err := ledger.ResponseAccountBlock(viteshan1, viteshan, req.ReqHash)
		if err != nil {
			t.Errorf("expected error.%v", err)
		}
	}

	time.Sleep(10 * time.Second)
}

func TestSnapshotFork(t *testing.T) {
	testSyncer := NewTestSync()
	ledger := NewLedger()
	ledger.Init(testSyncer)
	ledger.Start()
	time.Sleep(time.Second)

	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	block := ledger.sc.head
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)

	viteshan := "viteshan1"
	accountH0, _ := ledger.HeadAccount(viteshan)

	block2 := block

	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)
	//block = genSnapshotBlockBy(block)

	accountH1 := genAccountBlockBy(viteshan, block, accountH0, 0)
	ledger.AddAccountBlock(viteshan, accountH1)
	block = genSnapAccounts(block, accountH1)
	ledger.AddSnapshotBlock(block)
	time.Sleep(2 * time.Second)
	by := genSnapshotBlockBy(block2)
	ledger.AddSnapshotBlock(by)
	by = genSnapshotBlockBy(by)
	ledger.AddSnapshotBlock(by)
	time.Sleep(10 * time.Second)
	by = genSnapshotBlockBy(by)
	ledger.AddSnapshotBlock(by)

	c := make(chan int)
	c <- 1
	//time.Sleep(10 * time.Second)
}

func TestAccountFork(t *testing.T) {
	testSyncer := NewTestSync()
	ledger := NewLedger()
	ledger.Init(testSyncer)
	ledger.Start()
	time.Sleep(time.Second)

	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	block := ledger.sc.head
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)

	viteshan := "viteshan1"
	accountH0, _ := ledger.HeadAccount(viteshan)
	accountH1 := genAccountBlockBy(viteshan, block, accountH0, 0)
	accountH20 := genAccountBlockBy(viteshan, block, accountH1, 0)
	ledger.AddAccountBlock(viteshan, accountH1)
	ledger.AddAccountBlock(viteshan, accountH20)
	time.Sleep(2 * time.Second)
	accountH21 := genAccountBlockBy(viteshan, block, accountH1, 1)
	ledger.AddAccountBlock(viteshan, accountH21)

	block = genSnapAccounts(block, accountH1)
	ledger.AddSnapshotBlock(block)
	time.Sleep(2 * time.Second)
	block = genSnapAccounts(block, accountH21)
	ledger.AddSnapshotBlock(block)

	c := make(chan int)
	c <- 1
	//time.Sleep(10 * time.Second)
}

func genSnapshotBlockBy(block *common.SnapshotBlock) *common.SnapshotBlock {
	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), nil)
	snapshot.SetHash(utils.CalculateSnapshotHash(snapshot))
	return snapshot
}

func genSnapAccounts(block *common.SnapshotBlock, stateBlocks ...*common.AccountStateBlock) *common.SnapshotBlock {
	var accounts []*common.AccountHashH
	for _, v := range stateBlocks {
		accounts = append(accounts, common.NewAccountHashH(v.Signer(), v.Hash(), v.Height()))
	}
	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), accounts)
	snapshot.SetHash(utils.CalculateSnapshotHash(snapshot))
	return snapshot
}

func genAccountBlockBy(address string, prev *common.AccountStateBlock, modifiedAmount int) *common.AccountStateBlock {
	to := "viteshan"
	block := common.NewAccountBlock(prev.Height()+1, "", prev.Hash(), address, time.Now(),
		prev.Amount+modifiedAmount, modifiedAmount, common.SEND, address, to, "", -1)
	block.SetHash(utils.CalculateAccountHash(block))
	return block
}

func genSnapshotBlock(ledger *ledger) *common.SnapshotBlock {
	block := ledger.sc.head

	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), nil)
	snapshot.SetHash(utils.CalculateSnapshotHash(snapshot))
	return snapshot
}

func TestMap(t *testing.T) {
	stMap := make(map[int]string)
	stMap[12] = "23"
	stMap[13] = "23"
	for k, _ := range stMap {
		delete(stMap, k)
	}
	println(stMap[13])
}

func TestLedger_MiningSnapshotBlock(t *testing.T) {
	testSyncer := NewTestSync()
	ledger := NewLedger()
	ledger.Init(testSyncer)
	ledger.Start()

	committee := genCommitee()
	miner, bus := genMiner(committee, ledger, testSyncer)

	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	var c chan int = make(chan int)
	select {
	case c <- 0:
	case <-time.After(5 * time.Second):
		println("timeout and downloader finish.")
		//miner.downloaderRegisterCh <- 0
		bus.Publish(common.DwlDone)
		println("-----------timeout")
	}

	c <- 0
}

func genMiner(committee *consensus.Committee, rw miner.SnapshotChainRW, status face.SyncStatus) (miner.Miner, EventBus.Bus) {
	bus := EventBus.New()
	coinbase := common.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	miner := miner.NewMiner(rw, status, bus, coinbase, committee)
	return miner, bus
}

func genCommitee() *consensus.Committee {
	genesisTime := GetGenesisSnapshot().Timestamp()
	committee := consensus.NewCommittee(genesisTime, 1, int32(len(consensus.DefaultMembers)))
	return committee
}

func TestNewMiner(t *testing.T) {
	testSyncer := NewTestSync()
	ledger := NewLedger()
	ledger.Init(testSyncer)
	ledger.Start()

	committee := genCommitee()
	miner, bus := genMiner(committee, ledger, testSyncer)

	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	var c chan int = make(chan int)
	select {
	case c <- 0:
	case <-time.After(1 * time.Second):
		println("timeout and downloader finish.")
		//miner.downloaderRegisterCh <- 0
		bus.Publish(common.DwlDone)
		println("-----------timeout")
	}

	time.Sleep(10 * time.Second)
	println("-----------add")

	viteshan := "viteshan1"
	accountH0, _ := ledger.HeadAccount(viteshan)

	block := ledger.sc.head

	accountH1 := genAccountBlockBy(viteshan, block, accountH0, 0)
	ledger.AddAccountBlock(viteshan, accountH1)

	c <- 0
}
